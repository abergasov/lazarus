package routes

import (
	"fmt"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/service/authorization"
	docsvc "lazarus/internal/service/document"
	labsvc "lazarus/internal/service/lab"
	"lazarus/internal/service/user"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	medagent "lazarus/internal/agent"
)

type Server struct {
	appAddr    string
	log        logger.AppLogger
	httpEngine *fiber.App
	conf       *config.AppConfig

	googleOAuth *oauth2.Config

	srvAuth *authorization.Service
	srvUser *user.Service

	// MedHelp services
	db               *sqlx.DB
	orchestrator     *medagent.Orchestrator
	docSvc           *docsvc.Service
	labSvc           *labsvc.Service
	insightGenerator *medagent.InsightGenerator
}

var googleScopes = []string{
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/userinfo.email",
}

// MedHelpDeps holds the new medhelp services for injection
type MedHelpDeps struct {
	DB           *sqlx.DB
	Orchestrator *medagent.Orchestrator
	DocSvc       *docsvc.Service
	LabSvc       *labsvc.Service
}

// InitAppRouter initializes the HTTP Server.
func InitAppRouter(
	log logger.AppLogger,
	cfg *config.AppConfig,
	srvAuth *authorization.Service,
	srvUser *user.Service,
	address string,
	enableTelemetry bool,
	medHelp ...*MedHelpDeps,
) *Server {
	appPrefix := "http://"
	if cfg.SSLEnable {
		appPrefix = "https://"
	}
	app := &Server{
		appAddr: address,
		httpEngine: fiber.New(fiber.Config{
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}),
		srvAuth: srvAuth,
		srvUser: srvUser,
		log:     log.With(logger.WithService("http")),
		conf:    cfg,
		googleOAuth: &oauth2.Config{
			RedirectURL:  appPrefix + cfg.AppDomain + "/api/auth/google/callback",
			ClientID:     cfg.GoogleAppID,
			ClientSecret: cfg.GoogleAppSecret,
			Scopes:       googleScopes,
			Endpoint:     google.Endpoint,
		},
	}
	if len(medHelp) > 0 && medHelp[0] != nil {
		app.db = medHelp[0].DB
		app.orchestrator = medHelp[0].Orchestrator
		app.docSvc = medHelp[0].DocSvc
		app.labSvc = medHelp[0].LabSvc
		if app.db != nil {
			app.insightGenerator = medagent.NewInsightGenerator(app.db)
		}
	}

	app.httpEngine.Use(recover.New())
	if enableTelemetry {
		reg := prometheus.NewRegistry()
		reg.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
			collectors.NewBuildInfoCollector(),
		)
		app.httpEngine.Get("/metrics", adaptor.HTTPHandler(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})))
	}
	app.initRoutes()
	return app
}

func (s *Server) initRoutes() {
	s.httpEngine.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.SendString("pong")
	})

	s.httpEngine.Get("/api/auth/google/login", s.oauthGoogleLogin)
	s.httpEngine.Get("/api/auth/google/callback", s.oauthGoogleCallback)
	s.httpEngine.Post("/api/v1/auth/exchange", s.exchangeCode)
	s.httpEngine.Post("/api/v1/auth/logout", s.Logout)

	api := s.httpEngine.Group("/api/v1", s.jwtMiddleware())
	api.Get("/user/me", s.wrapAuth(s.handleUser))

	// MedHelp routes
	api.Post("/visits", s.wrapAuthUUID(s.handleCreateVisit))
	api.Get("/visits", s.wrapAuthUUID(s.handleListVisits))
	api.Get("/visits/:id", s.wrapAuthUUID(s.handleGetVisit))
	api.Put("/visits/:id/phase", s.wrapAuthUUID(s.handleUpdateVisitPhase))

	api.Post("/documents", s.wrapAuthUUID(s.handleDocumentUpload))
	api.Get("/documents", s.wrapAuthUUID(s.handleListDocuments))

	api.Get("/labs", s.wrapAuthUUID(s.handleListLabs))
	api.Get("/labs/:loinc/trend", s.wrapAuthUUID(s.handleLabTrend))

	api.Get("/medications", s.wrapAuthUUID(s.handleListMedications))
	api.Post("/medications", s.wrapAuthUUID(s.handleAddMedication))
	api.Delete("/medications/:id", s.wrapAuthUUID(s.handleDeleteMedication))

	api.Get("/profile", s.wrapAuthUUID(s.handleGetProfile))
	api.Put("/profile/demographics", s.wrapAuthUUID(s.handleUpdateDemographics))
	api.Put("/profile/conditions", s.wrapAuthUUID(s.handleUpdateConditions))

	api.Post("/agent/stream", s.wrapAuth(s.handleAgentStream))

	// Home (contextual surface)
	api.Get("/home", s.wrapAuthUUID(s.handleHome))

	// Insights
	api.Get("/insights", s.wrapAuthUUID(s.handleListInsights))
	api.Put("/insights/:id/dismiss", s.wrapAuthUUID(s.handleDismissInsight))

	// Conversations (scoped)
	api.Post("/conversations", s.wrapAuthUUID(s.handleCreateConversation))
	api.Get("/conversations/:id", s.wrapAuthUUID(s.handleGetConversation))
	api.Post("/conversations/:id/messages", s.wrapAuthUUID(s.handleConversationMessage))

	// Onboarding
	api.Post("/onboarding/upload", s.wrapAuthUUID(s.handleOnboardingUpload))
	api.Post("/onboarding/confirm", s.wrapAuthUUID(s.handleOnboardingConfirm))
}

// Run starts the HTTP Server.
func (s *Server) Run() error {
	s.log.Info("Starting HTTP server", logger.WithString("port", s.appAddr))
	return s.httpEngine.Listen(s.appAddr)
}

func (s *Server) Stop() error {
	return s.httpEngine.Shutdown()
}

func (s *Server) jwtMiddleware() fiber.Handler {
	key := []byte(s.conf.JWTKey)

	return func(c *fiber.Ctx) error {
		// 1) cookie
		raw := c.Cookies(TokenCookie) // "tc"

		// 2) fallback Authorization header
		if raw == "" {
			h := c.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(h), "bearer ") {
				raw = strings.TrimSpace(h[7:])
			}
		}

		if raw == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}

		claims := &entities.UserJWT{}
		tok, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS512.Alg() {
				return nil, fmt.Errorf("unexpected alg: %s", t.Method.Alg())
			}
			return key, nil
		})
		if err != nil || !tok.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		c.Locals("user", tok)
		return c.Next()
	}
}

func (s *Server) wrapAuth(route func(c *fiber.Ctx, userID int64) error) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(map[string]interface{}{"error": "unauthorized"})
		}
		claims, ok := token.Claims.(*entities.UserJWT)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(map[string]interface{}{"error": "unauthorized"})
		}
		return route(c, claims.GetUserID())
	}
}

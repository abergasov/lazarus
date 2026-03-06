package routes

import (
	"fmt"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/service/authorization"
	"lazarus/internal/service/user"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Server struct {
	appAddr    string
	log        logger.AppLogger
	httpEngine *fiber.App
	conf       *config.AppConfig

	googleOAuth *oauth2.Config

	srvAuth *authorization.Service
	srvUser *user.Service
}

var googleScopes = []string{
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/userinfo.email",
}

// InitAppRouter initializes the HTTP Server.
func InitAppRouter(
	log logger.AppLogger,
	cfg *config.AppConfig,
	srvAuth *authorization.Service,
	srvUser *user.Service,
	address string,
	enableTelemetry bool,
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

func (s *Server) wrapAuth(route func(c *fiber.Ctx, userID uuid.UUID) error) fiber.Handler {
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

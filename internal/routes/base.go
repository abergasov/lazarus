package routes

import (
	"lazarus/internal/config"
	"lazarus/internal/logger"
	"lazarus/internal/service/authorization"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	appAddr    string
	log        logger.AppLogger
	httpEngine *fiber.App
	conf       *config.AppConfig

	srvAuth *authorization.Service
}

// InitAppRouter initializes the HTTP Server.
func InitAppRouter(log logger.AppLogger, cfg *config.AppConfig, srvAuth *authorization.Service, address string, enableTelemetry bool) *Server {
	app := &Server{
		appAddr: address,
		httpEngine: fiber.New(fiber.Config{
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}),
		srvAuth: srvAuth,
		log:     log.With(logger.WithService("http")),
		conf:    cfg,
	}
	if cfg.AuthConfig.Mode == "development" {
		app.httpEngine.Use(cors.New(cors.Config{
			AllowOrigins:     "http://localhost:3000,http://127.0.0.1:3000",
			AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
			AllowHeaders:     "Authorization,Content-Type,Accept,Origin",
			ExposeHeaders:    "Content-Length",
			AllowCredentials: true,
			MaxAge:           600,
		}))
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

	s.httpEngine.Get("/auth/login/:provider", s.authLogin)
	s.httpEngine.Get("/auth/callback", s.authCallback)
	s.httpEngine.All("/auth/logout", s.authLogout)
}

// Run starts the HTTP Server.
func (s *Server) Run() error {
	s.log.Info("Starting HTTP server", logger.WithString("port", s.appAddr))
	return s.httpEngine.Listen(s.appAddr)
}

func (s *Server) Stop() error {
	return s.httpEngine.Shutdown()
}

package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gomantics/semantix/api/health"
	"github.com/gomantics/semantix/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Run(lc fx.Lifecycle, l *zap.Logger) error {
	e := echo.New()

	if !config.IsDev() {
		e.HideBanner = true
		e.HidePort = true
	}

	configureMiddleware(e, l)
	configureRoutes(e, l)

	server := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", config.Server.Port()),
		Handler:           e,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				l.Info("starting API server", zap.String("addr", server.Addr))
				if err := e.StartServer(server); err != nil && !errors.Is(err, http.ErrServerClosed) {
					l.Error("error starting echo server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			l.Info("shutdown signal received")
			return e.Shutdown(ctx)
		},
	})

	return nil
}

func configureMiddleware(e *echo.Echo, l *zap.Logger) {
	// Request ID must come first
	e.Use(middleware.RequestID())

	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 12, // 4 KB
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			l.Error("recovered from panic",
				zap.Error(err),
				zap.ByteString("stack", stack),
				zap.String("request_id", c.Response().Header().Get(echo.HeaderXRequestID)),
			)
			return nil
		},
	}))

	// Request logging
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			l.Info("request",
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
				zap.String("remote_ip", v.RemoteIP),
				zap.String("request_id", v.RequestID),
			)
			return nil
		},
		LogLatency:   true,
		LogRemoteIP:  true,
		LogMethod:    true,
		LogURI:       true,
		LogRequestID: true,
		LogStatus:    true,
	}))

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: config.Server.CorsAllowedOrigins(),
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodPatch,
		},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Origin", "X-Request-ID"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"Content-Length"},
		MaxAge:           int((24 * time.Hour).Seconds()),
	}))

	if config.IsDev() {
		e.IPExtractor = echo.ExtractIPDirect()
	} else {
		e.IPExtractor = echo.ExtractIPFromXFFHeader()
	}
}

func configureRoutes(e *echo.Echo, l *zap.Logger) {
	health.Configure(e, l)

	// TODO: Phase 1-3 - Add routes as they are implemented
	// workspaces.Configure(e, l)
	// gittokens.Configure(e, l)
	// repositories.Configure(e, l)
	// search.Configure(e, l)
}

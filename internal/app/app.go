package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"

	"github.com/ToxicSozo/avito-test/api"
	"github.com/ToxicSozo/avito-test/internal/config"
	"github.com/ToxicSozo/avito-test/internal/domain/service"
	"github.com/ToxicSozo/avito-test/internal/storage/postgres"
	httpHandlers "github.com/ToxicSozo/avito-test/internal/transport/http/handlers"
)

func Run(ctx context.Context, cfg *config.Config, log *logrus.Logger) error {
	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	swagger, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("load swagger: %w", err)
	}

	svc := service.New(db.Conn())

	handlerImpl := httpHandlers.NewHandler(
		svc,
		svc,
		svc,
		log.WithField("component", "http"),
	)

	baseRouter := chi.NewRouter()
	baseRouter.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.Timeout(30*time.Second),
		nethttpmiddleware.OapiRequestValidator(swagger),
	)

	router := api.HandlerWithOptions(handlerImpl, api.ChiServerOptions{
		BaseRouter: baseRouter,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.WithField("addr", srv.Addr).Info("HTTP server listening")
		if serveErr := srv.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- serveErr
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("listen and serve: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	log.Info("server gracefully stopped")
	return nil
}

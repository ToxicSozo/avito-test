package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ToxicSozo/avito-test/internal/api"
	"github.com/ToxicSozo/avito-test/internal/config"
	"github.com/ToxicSozo/avito-test/internal/server"
	"github.com/ToxicSozo/avito-test/internal/service"
	"github.com/ToxicSozo/avito-test/internal/storage/postgres"
)

// Run wires dependencies, starts HTTP server and blocks until ctx cancellation.
func Run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	svc := service.New(db.Pool())

	handlerImpl := server.NewHandler(
		svc,
		svc,
		svc,
		log,
		cfg.AdminToken,
		cfg.UserToken,
	)

	baseRouter := chi.NewRouter()
	baseRouter.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.Timeout(30*time.Second),
	)
	baseRouter.Get("/stats/assignments", handlerImpl.GetAssignmentStats)

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
		log.Info("HTTP server listening", "addr", srv.Addr)
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	log.Info("server gracefully stopped")
	return nil
}

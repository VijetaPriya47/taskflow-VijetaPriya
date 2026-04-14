package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpapi "taskflow-backend/internal/transport/http"
	postgres "taskflow-backend/internal/storage/postgres"
	service "taskflow-backend/internal/application"
	"taskflow-backend/internal/infrastructure/logging"
)

func main() {
	addr := envString("HTTP_ADDR", ":4000")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := logging.New()
	slog.SetDefault(logger)

	pg, err := postgres.Connect(ctx)
	if err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pg.Pool.Close()

	userRepo := postgres.NewPostgresUserRepository(pg.Pool)
	projectRepo := postgres.NewPostgresProjectRepository(pg.Pool)
	taskRepo := postgres.NewPostgresTaskRepository(pg.Pool)
	activityRepo := postgres.NewPostgresActivityRepository(pg.Pool)

	authSvc := service.NewAuthService(userRepo)
	activitySvc := service.NewActivityService(activityRepo)
	projectSvc := service.WithProjectActivity(service.NewProjectService(projectRepo, taskRepo), activitySvc)
	taskSvc := service.WithTaskActivity(service.NewTaskService(projectRepo, taskRepo), activitySvc)

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewRouter(httpapi.Deps{Auth: authSvc, Projects: projectSvc, Tasks: taskSvc}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("taskflow backend listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}
}

func envString(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

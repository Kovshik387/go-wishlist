package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/wishtrack/internal/bot"
	"github.com/example/wishtrack/internal/config"
	"github.com/example/wishtrack/internal/httpapi"
	"github.com/example/wishtrack/internal/notification"
	"github.com/example/wishtrack/internal/platform"
	"github.com/example/wishtrack/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}
	db, err := platform.OpenDatabase(cfg.DatabasePath)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := platform.RunMigrations(db, cfg.MigrationsDir); err != nil {
		logger.Error("run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("database ready", "path", cfg.DatabasePath, "mode", cfg.Mode)
	if cfg.Mode == "migrate" {
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	dataStore := store.New(db)
	if cfg.Mode == "worker" {
		worker := &notification.Worker{
			Store:        dataStore,
			Bot:          &bot.Client{Token: cfg.TelegramBotToken},
			Logger:       logger,
			DigestWindow: cfg.NotificationDigest,
			PollInterval: cfg.WorkerPollInterval,
			PublicURL:    cfg.PublicURL,
		}
		if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("worker stopped", "error", err)
			os.Exit(1)
		}
		return
	}

	api := httpapi.New(cfg, db, dataStore, logger)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown", "error", err)
		}
	}()
	logger.Info("HTTP server started", "addr", cfg.HTTPAddr, "env", cfg.Env)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}

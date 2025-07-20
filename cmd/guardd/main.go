package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"base-treasury-guard/internal/config"
	"base-treasury-guard/internal/httpserver"
	"base-treasury-guard/internal/logger"
	"base-treasury-guard/internal/metrics"
	"base-treasury-guard/internal/watcher"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reg := metrics.NewRegistry(cfg.MetricsNamespace)
	server := httpserver.Start(cfg.HTTPListenAddr, reg.Handler(), log)
	log.Info("http server listening", zap.String("addr", server.Addr()))

	w := watcher.New(cfg, log, reg)
	watcherErr := make(chan error, 1)
	go func() {
		watcherErr <- w.Run(ctx)
	}()

	select {
	case <-ctx.Done():
	case err := <-watcherErr:
		if err != nil {
			log.Error("watcher exited", zap.Error(err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown failed", zap.Error(err))
	} else {
		log.Info("http server stopped")
	}
}

package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"

    "base-treasury-guard/internal/config"
    "base-treasury-guard/internal/httpserver"
    "base-treasury-guard/internal/logger"
    "base-treasury-guard/internal/metrics"
    "base-treasury-guard/internal/watcher"
)

func main() {
    cfg := config.Load()
    log := logger.New(cfg.LogLevel)
    metrics := metrics.New(cfg.MetricsNamespace)

    if missing := config.MissingRequired(cfg); len(missing) > 0 {
        log.Error("missing required env vars", "vars", missing)
        os.Exit(1)
    }

    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    srv := httpserver.New(cfg.HTTPListenAddr, log, metrics.Handler())
    go func() {
        if err := srv.Start(ctx); err != nil {
            log.Error("http server exited", "err", err)
            cancel()
        }
    }()

    w := watcher.New(cfg, log, metrics)
    if err := w.Run(ctx); err != nil {
        log.Error("watcher exited", "err", err)
        os.Exit(1)
    }
}

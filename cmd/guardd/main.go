package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"

    "base-treasury-guard/internal/config"
    "base-treasury-guard/internal/logger"
    "base-treasury-guard/internal/watcher"
)

func main() {
    cfg := config.Load()
    log := logger.New(cfg.LogLevel)

    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    w := watcher.New(cfg, log)
    if err := w.Run(ctx); err != nil {
        log.Error("watcher exited", "err", err)
        os.Exit(1)
    }
}

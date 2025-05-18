package watcher

import (
    "context"

    "base-treasury-guard/internal/config"
)

type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
}

type Watcher struct {
    cfg config.Config
    log Logger
}

func New(cfg config.Config, log Logger) *Watcher {
    return &Watcher{cfg: cfg, log: log}
}

func (w *Watcher) Run(ctx context.Context) error {
    w.log.Info("watcher started")
    <-ctx.Done()
    w.log.Info("watcher stopped")
    return nil
}

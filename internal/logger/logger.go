package logger

import (
    "go.uber.org/zap"
)

type Logger interface {
    Info(msg string, fields ...any)
    Error(msg string, fields ...any)
}

type zapSugar struct {
    *zap.SugaredLogger
}

func (l zapSugar) Info(msg string, fields ...any) {
    l.SugaredLogger.Infow(msg, fields...)
}

func (l zapSugar) Error(msg string, fields ...any) {
    l.SugaredLogger.Errorw(msg, fields...)
}

func New(level string) Logger {
    cfg := zap.NewProductionConfig()
    if level != "" {
        if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
            cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
        }
    }
    log, _ := cfg.Build()
    return zapSugar{log.Sugar()}
}

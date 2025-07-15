package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string) *zap.Logger {
	cfg := zap.NewProductionConfig()
	lvl := strings.ToLower(strings.TrimSpace(level))
	if lvl == "" {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	} else if err := cfg.Level.UnmarshalText([]byte(lvl)); err != nil {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	log, err := cfg.Build()
	if err != nil {
		return zap.NewNop()
	}
	return log
}

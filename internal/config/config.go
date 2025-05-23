package config

import "os"

type Config struct {
    LogLevel string
}

func Load() Config {
    level := os.Getenv("LOG_LEVEL")
    if level == "" {
        level = "info"
    }
    return Config{LogLevel: level}
}

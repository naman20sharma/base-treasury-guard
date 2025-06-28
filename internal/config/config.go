package config

import (
    "os"
    "strconv"
)

type Config struct {
    HTTPListenAddr   string
    MetricsNamespace string
    RPCURL           string
    WSURL            string
    ChainID          string
    ContractAddress  string
    GuardianKey      string
    ExecutorKey      string
    LogLevel         string
    TokenAllowlist   string
    MaxAmountWei     string
    CooldownSeconds  int64
    GasFloor         uint64
    GasPerTx         uint64
}

var RequiredEnv = []string{
    "RPC_URL",
    "WS_URL",
    "CHAIN_ID",
    "CONTRACT_ADDRESS",
    "GUARDIAN_KEY",
    "EXECUTOR_KEY",
}

func Load() Config {
    return Config{
        HTTPListenAddr:   envOr("HTTP_LISTEN_ADDR", "127.0.0.1:9000"),
        MetricsNamespace: envOr("METRICS_NAMESPACE", "treasury_guard"),
        RPCURL:           os.Getenv("RPC_URL"),
        WSURL:            os.Getenv("WS_URL"),
        ChainID:          os.Getenv("CHAIN_ID"),
        ContractAddress:  os.Getenv("CONTRACT_ADDRESS"),
        GuardianKey:      os.Getenv("GUARDIAN_KEY"),
        ExecutorKey:      os.Getenv("EXECUTOR_KEY"),
        LogLevel:         envOr("LOG_LEVEL", "info"),
        TokenAllowlist:   envOr("TOKEN_ALLOWLIST", ""),
        MaxAmountWei:     envOr("MAX_AMOUNT_WEI", ""),
        CooldownSeconds:  envInt64("COOLDOWN_SECONDS", 0),
        GasFloor:         envUint64("GAS_FLOOR", 300_000),
        GasPerTx:         envUint64("GAS_PER_TX", 250_000),
    }
}

func MissingRequired(cfg Config) []string {
    missing := make([]string, 0)
    if cfg.RPCURL == "" {
        missing = append(missing, "RPC_URL")
    }
    if cfg.WSURL == "" {
        missing = append(missing, "WS_URL")
    }
    if cfg.ChainID == "" {
        missing = append(missing, "CHAIN_ID")
    }
    if cfg.ContractAddress == "" {
        missing = append(missing, "CONTRACT_ADDRESS")
    }
    if cfg.GuardianKey == "" {
        missing = append(missing, "GUARDIAN_KEY")
    }
    if cfg.ExecutorKey == "" {
        missing = append(missing, "EXECUTOR_KEY")
    }
    return missing
}

func envOr(key, fallback string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return fallback
}

func envInt64(key string, fallback int64) int64 {
    if val := os.Getenv(key); val != "" {
        if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
            return parsed
        }
    }
    return fallback
}

func envUint64(key string, fallback uint64) uint64 {
    if val := os.Getenv(key); val != "" {
        if parsed, err := strconv.ParseUint(val, 10, 64); err == nil {
            return parsed
        }
    }
    return fallback
}

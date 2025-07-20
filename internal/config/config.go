package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	RPCUrl          string
	WSUrl           string
	ChainID         uint64
	ContractAddress string

	GuardianKey string
	ExecutorKey string

	MaxBatch     int
	PollInterval time.Duration
	GasFloor     uint64

	HTTPListenAddr   string
	LogLevel         string
	MetricsNamespace string
	MetricsAddr      string

	PolicyMaxAmount     string
	PolicyAllowedTokens []string
	Network             string
}

func Load() Config {
	loadDotEnv(".env")

	cfg := Config{}

	cfg.RPCUrl = getenvDefault("RPC_URL", "http://127.0.0.1:8545")
	cfg.WSUrl = getenvDefault("WS_URL", "ws://127.0.0.1:8545")
	cfg.ChainID = getenvUint64("CHAIN_ID", 31337)
	cfg.ContractAddress = getenvDefault("CONTRACT_ADDRESS", "0x0000000000000000000000000000000000000000")

	cfg.GuardianKey = getenvDefault("GUARDIAN_KEY", "")
	cfg.ExecutorKey = getenvDefault("EXECUTOR_KEY", "")

	cfg.MaxBatch = getenvInt("MAX_BATCH", 10)
	cfg.PollInterval = getenvDuration("POLL_INTERVAL", 5*time.Second)
	cfg.GasFloor = getenvUint64("GAS_FLOOR", 50000)

	cfg.HTTPListenAddr = getenvDefault("HTTP_LISTEN_ADDR", "127.0.0.1:9000")
	cfg.LogLevel = getenvDefault("LOG_LEVEL", "info")
	cfg.MetricsNamespace = getenvDefault("METRICS_NAMESPACE", "treasury_guard")
	cfg.MetricsAddr = getenvDefault("METRICS_ADDR", cfg.HTTPListenAddr)

	cfg.PolicyMaxAmount = getenvDefault("POLICY_MAX_AMOUNT", "0")
	cfg.PolicyAllowedTokens = splitCSV(getenvDefault("POLICY_ALLOWED_TOKENS", ""))
	cfg.Network = getenvDefault("NETWORK", "base-sepolia")

	return cfg
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func getenvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getenvUint64(key string, fallback uint64) uint64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

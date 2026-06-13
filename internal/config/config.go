package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	Port        string
	DiscoveryBatchSize int
	DiscoveryDelayMs   int
}

func Load() (*Config, error) {
	batchSize, _ := strconv.Atoi(os.Getenv("DISCOVERY_BATCH_SIZE"))
	if batchSize <= 0 {
		batchSize = 50
	}

	delayMs, _ := strconv.Atoi(os.Getenv("DISCOVERY_DELAY_MS"))
	if delayMs < 0 {
		delayMs = 100
	}

	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		Port:               os.Getenv("PORT"),
		DiscoveryBatchSize: batchSize,
		DiscoveryDelayMs:   delayMs,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}

package config

import (
	"log"
	"os"
)

type Config struct {
	Port              string
	Environment       string
	PythonServiceURL  string
	AlphaVantageKey   string
	FirestoreProject  string
	GCSBucket         string
	CacheTTLHours     int
	MaxConcurrentFetches int
}

func Load() *Config {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		Environment:       getEnv("ENVIRONMENT", "production"),
		PythonServiceURL:  getEnv("PYTHON_SERVICE_URL", ""),
		AlphaVantageKey:   getEnv("ALPHA_VANTAGE_KEY", ""),
		FirestoreProject:  getEnv("FIRESTORE_PROJECT_ID", "pfc-portfolio-forecast"),
		GCSBucket:         getEnv("GCS_BUCKET_NAME", "pfc-forecast-cache-abc123"),
		CacheTTLHours:     24,
		MaxConcurrentFetches: 10,
	}

	// Validate required fields
	if cfg.AlphaVantageKey == "" {
		log.Println("⚠️  ALPHA_VANTAGE_KEY not set, using Yahoo Finance only")
	}

	if cfg.PythonServiceURL == "" {
		log.Fatal("❌ PYTHON_SERVICE_URL is required")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

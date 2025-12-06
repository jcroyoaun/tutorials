// Package config handles application configuration from environment variables.
package config

import (
	"os"
	"strings"
)

// Config holds all configuration values for the sentinel application.
type Config struct {
	// S3Bucket is the name of the S3 bucket for storing discovery.json
	S3Bucket string

	// S3FileName is the name of the file to store in S3
	S3FileName string

	// IgnoredNamespaces is a list of namespaces to exclude from discovery
	IgnoredNamespaces []string
}

// Load reads configuration from environment variables and returns a Config.
// Environment Variables:
//   - S3_BUCKET: The S3 bucket name (default: "posadev-go-sentinel")
//   - S3_FILE_NAME: The file name in S3 (default: "discovery.json")
//   - IGNORED_NAMESPACES: Comma-separated list of namespaces to ignore
func Load() (*Config, error) {
	cfg := Config{
		S3Bucket:   getEnvOrDefault("S3_BUCKET", "posadev-go-sentinel"),
		S3FileName: getEnvOrDefault("S3_FILE_NAME", "discovery.json"),
	}

	// Parse ignored namespaces from comma-separated string
	ignoredNS := os.Getenv("IGNORED_NAMESPACES")
	if ignoredNS != "" {
		namespaces := strings.Split(ignoredNS, ",")
		for i := range namespaces {
			namespaces[i] = strings.TrimSpace(namespaces[i])
		}
		cfg.IgnoredNamespaces = namespaces
	}

	return &cfg, nil
}

// getEnvOrDefault returns the value of an environment variable or a default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

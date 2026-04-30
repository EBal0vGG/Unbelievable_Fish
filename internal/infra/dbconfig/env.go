package dbconfig

import (
	"os"
	"strconv"
	"time"
)

// EnvOrDefault returns os.Getenv(key) or def if empty.
func EnvOrDefault(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

// EnvDurationMinutes parses an env var as positive integer minutes; invalid or empty uses def minutes.
func EnvDurationMinutes(key string, def int) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(def) * time.Minute
	}
	minutes, err := strconv.Atoi(value)
	if err != nil || minutes <= 0 {
		return time.Duration(def) * time.Minute
	}
	return time.Duration(minutes) * time.Minute
}

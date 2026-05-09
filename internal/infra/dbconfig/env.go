package dbconfig

import (
	"os"
	"strconv"
	"strings"
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

// EnvBool parses common truthy/falsey strings; empty or invalid returns def.
func EnvBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	return def
}

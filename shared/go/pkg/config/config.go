package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetEnv gets an environment variable with a default value
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetEnvAsInt gets an environment variable as an integer
func GetEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetEnvAsBool gets an environment variable as a boolean
func GetEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetEnvAsFloat gets an environment variable as a float64
func GetEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetEnvAsDuration gets an environment variable as a time.Duration
func GetEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetEnvAsSlice gets an environment variable as a string slice (comma-separated)
func GetEnvAsSlice(key string, defaultValue []string, separator string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	if separator == "" {
		separator = ","
	}

	values := strings.Split(valueStr, separator)
	result := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// MustGetEnv gets an environment variable or panics if not set
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Environment variable %s is required but not set", key))
	}
	return value
}

// IsProduction checks if the environment is production
func IsProduction() bool {
	env := strings.ToLower(GetEnv("ENVIRONMENT", "development"))
	return env == "production" || env == "prod"
}

// IsDevelopment checks if the environment is development
func IsDevelopment() bool {
	env := strings.ToLower(GetEnv("ENVIRONMENT", "development"))
	return env == "development" || env == "dev"
}

// IsTest checks if the environment is test
func IsTest() bool {
	env := strings.ToLower(GetEnv("ENVIRONMENT", "development"))
	return env == "test" || env == "testing"
}

// GetEnvironment returns the current environment
func GetEnvironment() string {
	return GetEnv("ENVIRONMENT", "development")
}

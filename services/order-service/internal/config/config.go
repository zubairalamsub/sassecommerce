package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Kafka     KafkaConfig
	Services  ServicesConfig
	LogLevel  string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
	Enabled       bool
}

// ServicesConfig holds external service URLs
type ServicesConfig struct {
	InventoryURL string
	PaymentURL   string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvAsInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Database: getEnv("DB_NAME", "order_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Kafka: KafkaConfig{
			Brokers:       getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			Topic:         getEnv("KAFKA_TOPIC", "order-events"),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "order-service-projections"),
			Enabled:       getEnvAsBool("KAFKA_ENABLED", true),
		},
		Services: ServicesConfig{
			InventoryURL: getEnv("INVENTORY_SERVICE_URL", "http://localhost:8082"),
			PaymentURL:   getEnv("PAYMENT_SERVICE_URL", "http://localhost:8084"),
		},
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	return config, nil
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// GetServerAddress returns the server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// Helper functions to read environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}

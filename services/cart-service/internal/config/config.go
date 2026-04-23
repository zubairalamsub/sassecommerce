package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server ServerConfig
	Redis  RedisConfig
	Kafka  KafkaConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
	GroupID string
}

func Load() *Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8089"),
			Env:  getEnv("ENVIRONMENT", "development"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", "redis123"),
			DB:       redisDB,
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKER", "localhost:9092")},
			GroupID: getEnv("KAFKA_GROUP_ID", "cart-service"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

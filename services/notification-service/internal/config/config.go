package config

import (
	"os"
)

type Config struct {
	Server  ServerConfig
	MongoDB MongoDBConfig
	Kafka   KafkaConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type MongoDBConfig struct {
	URI    string
	DBName string
}

type KafkaConfig struct {
	Brokers []string
	GroupID string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8087"),
			Env:  getEnv("ENVIRONMENT", "development"),
		},
		MongoDB: MongoDBConfig{
			URI:    getEnv("MONGO_URI", "mongodb://admin:admin123@localhost:27017"),
			DBName: getEnv("MONGO_DB_NAME", "notification_db"),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKER", "localhost:9092")},
			GroupID: getEnv("KAFKA_GROUP_ID", "notification-service"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

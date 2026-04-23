package config

import (
	"os"
)

type Config struct {
	Server        ServerConfig
	Elasticsearch ElasticsearchConfig
	Kafka         KafkaConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type ElasticsearchConfig struct {
	URL   string
	Index string
}

type KafkaConfig struct {
	Brokers []string
	GroupID string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8090"),
			Env:  getEnv("ENVIRONMENT", "development"),
		},
		Elasticsearch: ElasticsearchConfig{
			URL:   getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
			Index: getEnv("ELASTICSEARCH_INDEX", "products"),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKER", "localhost:9092")},
			GroupID: getEnv("KAFKA_GROUP_ID", "search-service"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// allConfigEnvVars lists every variable Load reads. Tests blank every one of
// these before asserting on defaults, so results do not depend on whatever
// happens to be exported in the ambient shell running the tests.
var allConfigEnvVars = []string{
	"PORT", "ENVIRONMENT",
	"MONGO_URI", "MONGO_DB_NAME",
	"KAFKA_BROKER", "KAFKA_GROUP_ID",
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range allConfigEnvVars {
		t.Setenv(key, "")
	}
}

// === getEnv ===

func TestGetEnv(t *testing.T) {
	const key = "NOTIFICATION_SVC_TEST_STRING"

	t.Run("returns the env value when set", func(t *testing.T) {
		t.Setenv(key, "custom-value")
		assert.Equal(t, "custom-value", getEnv(key, "default-value"))
	})

	t.Run("returns the default when unset or empty", func(t *testing.T) {
		t.Setenv(key, "")
		assert.Equal(t, "default-value", getEnv(key, "default-value"))
	})
}

// === Load ===

func TestLoad_DefaultsWhenEnvUnset(t *testing.T) {
	clearConfigEnv(t)

	cfg := Load()

	assert.Equal(t, "8087", cfg.Server.Port)
	assert.Equal(t, "development", cfg.Server.Env)
	// This default matches the documented docker-compose MongoDB credentials
	// (admin/admin123) rather than embedding a new secret, so it is a
	// reasonable local-dev fallback -- unlike order-service's DB_PASSWORD
	// default, which does not match its own documented Postgres credential.
	assert.Equal(t, "mongodb://admin:admin123@localhost:27017", cfg.MongoDB.URI)
	assert.Equal(t, "notification_db", cfg.MongoDB.DBName)
	assert.Equal(t, []string{"localhost:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "notification-service", cfg.Kafka.GroupID)
}

func TestLoad_EnvOverridesDefaults(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("PORT", "9000")
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("MONGO_URI", "mongodb://user:pass@mongo-prod:27017")
	t.Setenv("MONGO_DB_NAME", "notifications_prod")
	t.Setenv("KAFKA_BROKER", "kafka-prod:9092")
	t.Setenv("KAFKA_GROUP_ID", "notification-service-prod")

	cfg := Load()

	assert.Equal(t, "9000", cfg.Server.Port)
	assert.Equal(t, "production", cfg.Server.Env)
	assert.Equal(t, "mongodb://user:pass@mongo-prod:27017", cfg.MongoDB.URI)
	assert.Equal(t, "notifications_prod", cfg.MongoDB.DBName)
	assert.Equal(t, []string{"kafka-prod:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "notification-service-prod", cfg.Kafka.GroupID)
}

// Real finding: unlike order-service's KAFKA_BROKERS (plural, comma-split),
// this service's KAFKA_BROKER (singular) never splits on comma. An operator
// who writes a comma-separated list -- following the sibling service's
// convention -- gets exactly one slice element holding the whole string,
// which kafka-go would try to dial as a single, invalid broker address.
func TestLoad_KafkaBrokerDoesNotSupportACommaSeparatedList(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("KAFKA_BROKER", "broker1:9092,broker2:9092")

	cfg := Load()

	assert.Len(t, cfg.Kafka.Brokers, 1, "KAFKA_BROKER is never split on comma, unlike order-service's KAFKA_BROKERS")
	assert.Equal(t, []string{"broker1:9092,broker2:9092"}, cfg.Kafka.Brokers)
}

func TestLoad_EmptyEnvValueFallsBackToDefault(t *testing.T) {
	// getEnv treats "" the same as unset, so a deployment template that
	// renders an unset variable as an empty string (common in Helm/compose)
	// falls back silently rather than producing an empty/broken value.
	clearConfigEnv(t)
	t.Setenv("MONGO_DB_NAME", "")

	cfg := Load()

	assert.Equal(t, "notification_db", cfg.MongoDB.DBName)
}

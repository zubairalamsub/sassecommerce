package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// allConfigEnvVars lists every variable LoadConfig reads. Tests blank every
// one of these before asserting on defaults, so results do not depend on
// whatever happens to be exported in the ambient shell running the tests.
var allConfigEnvVars = []string{
	"SERVER_HOST", "SERVER_PORT",
	"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
	"KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_CONSUMER_GROUP", "KAFKA_ENABLED",
	"INVENTORY_SERVICE_URL", "PAYMENT_SERVICE_URL",
	"LOG_LEVEL",
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range allConfigEnvVars {
		t.Setenv(key, "")
	}
}

// === getEnv ===

func TestGetEnv(t *testing.T) {
	const key = "ORDER_SVC_TEST_STRING"

	t.Run("returns the env value when set", func(t *testing.T) {
		t.Setenv(key, "custom-value")
		assert.Equal(t, "custom-value", getEnv(key, "default-value"))
	})

	t.Run("returns the default when unset", func(t *testing.T) {
		t.Setenv(key, "")
		assert.Equal(t, "default-value", getEnv(key, "default-value"))
	})
}

// === getEnvAsInt ===

// getEnvAsInt silently falls back to the default whenever strconv.Atoi
// cannot parse the value, instead of failing config load. A typo'd
// SERVER_PORT would boot the service on the wrong port rather than erroring
// at startup -- see TestLoadConfig_MalformedPortSilentlyFallsBackToDefault
// for the same behavior through the public LoadConfig entry point.
func TestGetEnvAsInt(t *testing.T) {
	const key = "ORDER_SVC_TEST_INT"
	const defaultValue = 4242

	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "valid positive integer", value: "9090", want: 9090},
		{name: "valid negative integer is accepted -- LoadConfig applies no range validation", value: "-1", want: -1},
		{name: "leading zero is parsed as decimal, not octal", value: "007", want: 7},
		{name: "non-numeric value falls back to the default", value: "not-a-number", want: defaultValue},
		{name: "decimal value falls back to the default (Atoi rejects floats)", value: "9090.5", want: defaultValue},
		{name: "hex-prefixed value falls back to the default (Atoi is base-10 only)", value: "0x1F90", want: defaultValue},
		{name: "empty value falls back to the default", value: "", want: defaultValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(key, tt.value)

			got := getEnvAsInt(key, defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}

// === getEnvAsBool ===

func TestGetEnvAsBool(t *testing.T) {
	const key = "ORDER_SVC_TEST_BOOL"
	const defaultValue = true

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "true", value: "true", want: true},
		{name: "1 means true", value: "1", want: true},
		{name: "TRUE uppercase", value: "TRUE", want: true},
		{name: "false overrides a true default", value: "false", want: false},
		{name: "0 means false", value: "0", want: false},
		{name: "malformed value falls back to the default", value: "yes", want: defaultValue},
		{name: "empty value falls back to the default", value: "", want: defaultValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(key, tt.value)

			got := getEnvAsBool(key, defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}

// === getEnvAsSlice ===

func TestGetEnvAsSlice(t *testing.T) {
	const key = "ORDER_SVC_TEST_SLICE"
	defaultValue := []string{"localhost:9092"}

	t.Run("unset falls back to the default", func(t *testing.T) {
		t.Setenv(key, "")
		assert.Equal(t, defaultValue, getEnvAsSlice(key, defaultValue))
	})

	t.Run("single value", func(t *testing.T) {
		t.Setenv(key, "broker1:9092")
		assert.Equal(t, []string{"broker1:9092"}, getEnvAsSlice(key, defaultValue))
	})

	t.Run("multiple values with no spaces", func(t *testing.T) {
		t.Setenv(key, "broker1:9092,broker2:9092,broker3:9092")
		assert.Equal(t, []string{"broker1:9092", "broker2:9092", "broker3:9092"}, getEnvAsSlice(key, defaultValue))
	})

	// Real finding: a broker list written the natural way -- with a space
	// after the comma, as most YAML/docker-compose authors would write it --
	// keeps that leading space on every element but the first. kafka-go would
	// then try to dial " broker2:9092" verbatim and fail to resolve it.
	t.Run("whitespace after the comma is not trimmed", func(t *testing.T) {
		t.Setenv(key, "broker1:9092, broker2:9092")

		got := getEnvAsSlice(key, defaultValue)

		assert.Equal(t, []string{"broker1:9092", " broker2:9092"}, got,
			"a comma-separated list written with a space after the comma silently produces an unreachable broker address")
	})
}

// === LoadConfig ===

func TestLoadConfig_DefaultsWhenEnvUnset(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "postgres", cfg.Database.User)
	// Real finding: this default does not match the documented dev Postgres
	// password (postgres123, per docker-compose / the platform's own infra
	// docs). An operator who forgets to set DB_PASSWORD gets a silent, wrong
	// credential instead of a clear failure.
	assert.Equal(t, "postgres", cfg.Database.Password)
	assert.Equal(t, "order_db", cfg.Database.Database)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, []string{"localhost:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "order-events", cfg.Kafka.Topic)
	assert.Equal(t, "order-service-projections", cfg.Kafka.ConsumerGroup)
	assert.True(t, cfg.Kafka.Enabled)
	assert.Equal(t, "http://localhost:8082", cfg.Services.InventoryURL)
	assert.Equal(t, "http://localhost:8084", cfg.Services.PaymentURL)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoadConfig_EnvOverridesDefaults(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_USER", "order_svc")
	t.Setenv("DB_PASSWORD", "s3cret")
	t.Setenv("DB_NAME", "orders")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")
	t.Setenv("KAFKA_TOPIC", "custom-order-events")
	t.Setenv("KAFKA_CONSUMER_GROUP", "custom-group")
	t.Setenv("KAFKA_ENABLED", "false")
	t.Setenv("INVENTORY_SERVICE_URL", "http://inventory:9000")
	t.Setenv("PAYMENT_SERVICE_URL", "http://payment:9001")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "db.internal", cfg.Database.Host)
	assert.Equal(t, 6543, cfg.Database.Port)
	assert.Equal(t, "order_svc", cfg.Database.User)
	assert.Equal(t, "s3cret", cfg.Database.Password)
	assert.Equal(t, "orders", cfg.Database.Database)
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.Equal(t, []string{"broker1:9092", "broker2:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "custom-order-events", cfg.Kafka.Topic)
	assert.Equal(t, "custom-group", cfg.Kafka.ConsumerGroup)
	assert.False(t, cfg.Kafka.Enabled)
	assert.Equal(t, "http://inventory:9000", cfg.Services.InventoryURL)
	assert.Equal(t, "http://payment:9001", cfg.Services.PaymentURL)
	assert.Equal(t, "debug", cfg.LogLevel)
}

// Real finding, demonstrated end-to-end through the public entry point: a
// typo'd SERVER_PORT does not fail config load, it silently boots on 8080.
func TestLoadConfig_MalformedPortSilentlyFallsBackToDefault(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SERVER_PORT", "not-a-number")

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.Equal(t, 8080, cfg.Server.Port, "a malformed SERVER_PORT should not silently boot on an unexpected port with no error or warning")
}

// Real finding: no range validation is applied to the parsed port, so an
// impossible value is accepted as "valid" and only fails later, when the
// server tries to bind it -- a confusing place to discover a config typo.
func TestLoadConfig_OutOfRangePortIsAcceptedWithoutValidation(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SERVER_PORT", "-1")

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.Equal(t, -1, cfg.Server.Port)
}

func TestLoadConfig_MalformedKafkaEnabledSilentlyFallsBackToDefault(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("KAFKA_ENABLED", "not-a-bool")

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.True(t, cfg.Kafka.Enabled, "a malformed KAFKA_ENABLED should fall back to the documented default (true), not silently disable Kafka")
}

// LoadConfig's signature includes an error return, but no code path in it
// ever produces one -- every helper falls back to a default instead of
// failing. This pins that current contract; see the QA report for why that
// is worth revisiting for the variables that matter most (port, credentials).
func TestLoadConfig_NeverReturnsAnError(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SERVER_PORT", "garbage")
	t.Setenv("KAFKA_ENABLED", "garbage")
	t.Setenv("DB_PORT", "garbage")

	_, err := LoadConfig()

	assert.NoError(t, err)
}

// === GetDatabaseDSN / GetServerAddress ===

func TestGetDatabaseDSN_FormatsAllFieldsInOrder(t *testing.T) {
	cfg := &Config{Database: DatabaseConfig{
		Host:     "db.internal",
		Port:     6543,
		User:     "order_svc",
		Password: "s3cret",
		Database: "orders",
		SSLMode:  "require",
	}}

	dsn := cfg.GetDatabaseDSN()

	assert.Equal(t, "host=db.internal port=6543 user=order_svc password=s3cret dbname=orders sslmode=require", dsn)
}

func TestGetServerAddress_JoinsHostAndPort(t *testing.T) {
	cfg := &Config{Server: ServerConfig{Host: "0.0.0.0", Port: 8080}}

	assert.Equal(t, "0.0.0.0:8080", cfg.GetServerAddress())
}

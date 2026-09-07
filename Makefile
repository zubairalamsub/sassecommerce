.PHONY: help up up-monitoring down build logs clean test lint lint-fix lint-install vet grafana-lookup-setup grafana-open

# Enable BuildKit + parallel image builds for every docker/compose invocation.
export DOCKER_BUILDKIT := 1
export COMPOSE_DOCKER_CLI_BUILD := 1

# Default target
help:
	@echo "E-Commerce Platform - Makefile Commands"
	@echo ""
	@echo "Infrastructure:"
	@echo "  make infra-up         - Start all infrastructure services (PostgreSQL, MongoDB, Redis, Kafka, etc.)"
	@echo "  make infra-down       - Stop all infrastructure services"
	@echo "  make infra-logs       - View infrastructure logs"
	@echo ""
	@echo "Services:"
	@echo "  make up               - Start app services + core infra (no monitoring stack)"
	@echo "  make up-core          - Start core infra + only named services (SERVICES=\"...\")"
	@echo "  make up-monitoring    - Start everything incl. Prometheus/Grafana/exporters"
	@echo "  make down             - Stop all services"
	@echo "  make build            - Build all services"
	@echo "  make rebuild          - Rebuild all services from scratch"
	@echo "  make logs             - View all service logs"
	@echo ""
	@echo "Individual Services:"
	@echo "  make tenant-up        - Start tenant service"
	@echo "  make tenant-logs      - View tenant service logs"
	@echo "  make tenant-build     - Build tenant service"
	@echo ""
	@echo "Database:"
	@echo "  make db-psql          - Access PostgreSQL CLI"
	@echo "  make db-mongo         - Access MongoDB CLI"
	@echo "  make db-redis         - Access Redis CLI"
	@echo ""
	@echo "Kafka:"
	@echo "  make kafka-topics     - List Kafka topics"
	@echo "  make kafka-consume    - Consume messages from tenant-events topic"
	@echo ""
	@echo "Testing:"
	@echo "  make test-tenant           - Run all tenant service tests"
	@echo "  make test-tenant-unit      - Run unit tests only"
	@echo "  make test-tenant-e2e       - Run E2E tests only"
	@echo "  make test-tenant-coverage  - Generate coverage report"
	@echo "  make test-tenant-race      - Run tests with race detection"
	@echo "  make test-tenant-report    - Generate comprehensive HTML test report"
	@echo "  make test-tenant-badges    - Generate test coverage badges"
	@echo "  make view-tenant-report    - View HTML test report in browser"
	@echo "  make view-tenant-coverage  - View coverage report in browser"
	@echo ""
	@echo "Observability:"
	@echo "  make grafana-lookup-setup - Create the grafana_ro role + support views (run once)"
	@echo "  make grafana-open         - Open Grafana (Support Lookup dashboard)"
	@echo ""
	@echo "Code quality (all 15 Go modules):"
	@echo "  make lint             - Run golangci-lint across every Go module"
	@echo "  make lint-fix         - Same, applying auto-fixes (then ALWAYS run make vet)"
	@echo "  make vet              - go vet every module (compiles tests, unlike go build)"
	@echo "  make lint-install     - Install the golangci-lint version CI pins"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean            - Remove containers and volumes"
	@echo "  make clean-all        - Remove everything including images"

# Infrastructure
infra-up:
	docker-compose up -d postgres mongodb redis zookeeper kafka elasticsearch

infra-down:
	docker-compose stop postgres mongodb redis zookeeper kafka elasticsearch

infra-logs:
	docker-compose logs -f postgres mongodb redis kafka elasticsearch

# All services (monitoring stack is gated behind the "monitoring" profile,
# so a normal dev bring-up skips Prometheus/Grafana/Loki/exporters).
up:
	docker-compose up -d

# Minimal dev loop: core infra plus only the services you name. Compose pulls
# in each named service's depends_on automatically. Examples:
#   make up-core                                     # infra only
#   make up-core SERVICES="tenant-service frontend"  # infra + what you need
CORE_INFRA = postgres mongodb redis zookeeper kafka
up-core:
	docker-compose up -d $(CORE_INFRA) $(SERVICES)

# Full stack including the monitoring + exporter profile.
up-monitoring:
	docker-compose --profile monitoring up -d

down:
	docker-compose down

build:
	docker-compose build

rebuild:
	docker-compose build --no-cache

logs:
	docker-compose logs -f

# Tenant service
tenant-up:
	docker-compose up -d tenant-service

tenant-logs:
	docker-compose logs -f tenant-service

tenant-build:
	docker-compose build tenant-service

tenant-restart:
	docker-compose restart tenant-service

# Database access
db-psql:
	docker exec -it ecommerce-postgres psql -U postgres

db-psql-tenant:
	docker exec -it ecommerce-postgres psql -U postgres -d tenant_db

db-mongo:
	docker exec -it ecommerce-mongodb mongosh -u admin -p admin123

db-redis:
	docker exec -it ecommerce-redis redis-cli -a redis123

# Kafka
kafka-topics:
	docker exec -it ecommerce-kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-consume:
	docker exec -it ecommerce-kafka kafka-console-consumer \
		--bootstrap-server localhost:9092 \
		--topic tenant-events \
		--from-beginning

kafka-create-topic:
	docker exec -it ecommerce-kafka kafka-topics \
		--bootstrap-server localhost:9092 \
		--create \
		--topic tenant-events \
		--partitions 3 \
		--replication-factor 1

# Cleanup
clean:
	docker-compose down -v

clean-all:
	docker-compose down -v --rmi all

# Health checks
health:
	@echo "Checking service health..."
	@curl -s http://localhost:8081/health | jq '.' || echo "Tenant service not responding"
	@curl -s http://localhost:5432 > /dev/null && echo "PostgreSQL: OK" || echo "PostgreSQL: DOWN"
	@curl -s http://localhost:27017 > /dev/null && echo "MongoDB: OK" || echo "MongoDB: DOWN"
	@curl -s http://localhost:6379 > /dev/null && echo "Redis: OK" || echo "Redis: DOWN"
	@curl -s http://localhost:9200 > /dev/null && echo "Elasticsearch: OK" || echo "Elasticsearch: DOWN"

# Development
dev-tenant:
	cd services/tenant-service && go run cmd/server/main.go

# Testing
test-tenant:
	@echo "Running all tenant service tests..."
	cd services/tenant-service && go test -v ./...

test-tenant-unit:
	@echo "Running unit tests..."
	cd services/tenant-service && go test -v ./internal/repository/... ./internal/service/...

test-tenant-e2e:
	@echo "Running E2E tests..."
	cd services/tenant-service && go test -v ./tests/e2e/...

test-tenant-coverage:
	@echo "Generating test coverage report..."
	cd services/tenant-service && go test -v -coverprofile=coverage.out ./...
	cd services/tenant-service && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: services/tenant-service/coverage.html"

test-tenant-race:
	@echo "Running tests with race detection..."
	cd services/tenant-service && go test -v -race ./...

# Test Reports
test-tenant-report:
	@echo "Generating comprehensive test report..."
	cd services/tenant-service && ./scripts/generate_test_report.sh

test-tenant-badges:
	@echo "Generating test badges..."
	cd services/tenant-service && ./scripts/generate_badges.sh

test-tenant-ci:
	@echo "Running CI test suite..."
	cd services/tenant-service && go install gotest.tools/gotestsum@latest
	cd services/tenant-service && gotestsum --format testname --jsonfile test-results.json -- -coverprofile=coverage.out -race ./...
	@echo "✅ CI tests complete"

view-tenant-report:
	@echo "Opening test report..."
	@if [ -f services/tenant-service/test-reports/latest/index.html ]; then \
		open services/tenant-service/test-reports/latest/index.html || xdg-open services/tenant-service/test-reports/latest/index.html; \
	else \
		echo "❌ No report found. Run 'make test-tenant-report' first"; \
	fi

view-tenant-coverage:
	@echo "Opening coverage report..."
	@if [ -f services/tenant-service/coverage.html ]; then \
		open services/tenant-service/coverage.html || xdg-open services/tenant-service/coverage.html; \
	else \
		cd services/tenant-service && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html; \
		open services/tenant-service/coverage.html || xdg-open services/tenant-service/coverage.html; \
	fi

# API Testing
test-api:
	@echo "Testing Tenant Service API..."
	@echo "\n1. Health Check:"
	curl -s http://localhost:8081/health | jq '.'
	@echo "\n2. Create Tenant:"
	curl -s -X POST http://localhost:8081/api/v1/tenants \
		-H "Content-Type: application/json" \
		-d '{"name": "Test Store", "email": "test@example.com", "tier": "free"}' | jq '.'
	@echo "\n3. List Tenants:"
	curl -s http://localhost:8081/api/v1/tenants | jq '.'

# ---------------------------------------------------------------------------
# Code quality
#
# One .golangci.yml at the repo root governs all 15 modules -- golangci-lint
# walks up from each module directory to find it, so these targets just visit
# each module in turn.
# ---------------------------------------------------------------------------

GOLANGCI_VERSION := v2.5.0
GO_MODULES := shared/go $(wildcard services/*-service)

lint-install:
	@echo "Installing golangci-lint $(GOLANGCI_VERSION) (the version CI pins)..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
		| sh -s -- -b "$$(go env GOPATH)/bin" $(GOLANGCI_VERSION)

lint:
	@fail=0; \
	for m in $(GO_MODULES); do \
		[ -f "$$m/go.mod" ] || continue; \
		printf '%-28s' "$$m"; \
		if out=$$(cd "$$m" && golangci-lint run --timeout=5m 2>&1); then \
			echo "clean"; \
		else \
			echo "ISSUES"; echo "$$out"; fail=1; \
		fi; \
	done; \
	exit $$fail

# --fix does not add imports and is not import-alias aware, so it can leave the
# tree uncompilable; worse, one errorlint fix can invert a test's meaning while
# still compiling and still passing. vet runs straight afterwards, and reading
# the resulting diff is not optional.
lint-fix:
	@for m in $(GO_MODULES); do \
		[ -f "$$m/go.mod" ] || continue; \
		echo "==> $$m"; \
		(cd "$$m" && golangci-lint run --fix --timeout=5m) || true; \
	done
	@$(MAKE) --no-print-directory vet

# go vet, not go build: vet compiles _test.go files too, which is exactly where
# a missing import left behind by --fix hides.
vet:
	@fail=0; \
	for m in $(GO_MODULES); do \
		[ -f "$$m/go.mod" ] || continue; \
		printf '%-28s' "$$m"; \
		if out=$$(cd "$$m" && go vet ./... 2>&1); then \
			echo "ok"; \
		else \
			echo "FAIL"; echo "$$out"; fail=1; \
		fi; \
	done; \
	exit $$fail

# ---------------------------------------------------------------------------
# Observability
# ---------------------------------------------------------------------------

# Creates the read-only grafana_ro role and the support_* views the Support
# Lookup dashboard reads.
#
# Not a docker-entrypoint-initdb.d script, deliberately: the tables it grants
# on are created by the services at startup, long after Postgres finishes
# initialising, so granting at init time would fail on tables that do not exist
# yet. Idempotent -- re-run it after a schema change.
grafana-lookup-setup:
	@set -e; \
	PW="$${GRAFANA_DB_READONLY_PASSWORD:-$$(grep -E '^GRAFANA_DB_READONLY_PASSWORD=' .env 2>/dev/null | cut -d= -f2-)}"; \
	if [ -z "$$PW" ]; then \
		echo "GRAFANA_DB_READONLY_PASSWORD is not set."; \
		echo "Add it to .env (see .env.example), then run this again."; \
		exit 1; \
	fi; \
	echo "Creating grafana_ro and the support_* views..."; \
	docker-compose exec -T postgres psql -U postgres -q \
		-v grafana_ro_password="$$PW" \
		-f - < infrastructure/monitoring/postgres/grafana-lookup.sql; \
	echo ""; \
	echo "Done. Grafana reads the password from its own environment, so if it"; \
	echo "was already running when you set it:  docker-compose restart grafana"

grafana-open:
	@echo "Grafana:        http://localhost:3001"
	@echo "Support Lookup: http://localhost:3001/d/support-lookup"
	@open http://localhost:3001/d/support-lookup 2>/dev/null \
		|| xdg-open http://localhost:3001/d/support-lookup 2>/dev/null \
		|| powershell -NoProfile -Command "Start-Process 'http://localhost:3001/d/support-lookup'" 2>/dev/null \
		|| true

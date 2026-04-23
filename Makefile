.PHONY: help up down build logs clean test

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
	@echo "  make up               - Start all services"
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

# All services
up:
	docker-compose up -d

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

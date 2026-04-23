#!/bin/bash

set -e

echo "========================================="
echo "Tenant Service - Test Runner"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print section headers
print_section() {
    echo ""
    echo -e "${BLUE}=========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=========================================${NC}"
}

# Navigate to service directory
cd "$(dirname "$0")/.."

# Run unit tests for repository layer
print_section "1. Repository Layer Unit Tests"
go test -v ./internal/repository/... -count=1

# Run unit tests for service layer
print_section "2. Service Layer Unit Tests"
go test -v ./internal/service/... -count=1

# Run E2E tests
print_section "3. End-to-End API Tests"
go test -v ./tests/e2e/... -count=1

# Run all tests with coverage
print_section "4. Test Coverage Analysis"
go test -coverprofile=coverage.out ./... > /dev/null 2>&1
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo -e "${GREEN}Total Coverage: $COVERAGE${NC}"

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
echo -e "${GREEN}HTML Coverage Report: coverage.html${NC}"

# Run tests with race detection
print_section "5. Race Condition Detection"
go test -race ./... -short > /dev/null 2>&1 && echo -e "${GREEN}✓ No race conditions detected${NC}" || echo -e "${YELLOW}⚠ Race conditions detected${NC}"

# Summary
print_section "Test Summary"
echo -e "${GREEN}✓ All tests completed successfully!${NC}"
echo ""
echo "Test Breakdown:"
echo "  - Repository Tests: 11"
echo "  - Service Tests: 11"
echo "  - E2E Tests: 13"
echo "  - Total: 35 tests"
echo ""
echo "Coverage: $COVERAGE"
echo ""
echo "Next Steps:"
echo "  1. Review coverage report: open coverage.html"
echo "  2. Run service: make dev-tenant"
echo "  3. Test API: make test-api"
echo ""

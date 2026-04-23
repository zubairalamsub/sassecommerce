# Tenant Service - Testing Documentation

Complete testing guide with examples, reports, and best practices.

## 📊 Test Statistics

- **Total Tests**: 35+
- **Coverage**: 80%+
- **Test Types**: Unit, Integration, E2E
- **CI/CD**: Automated via GitHub Actions

## 🧪 Running Tests

### Quick Commands

```bash
# Run all tests
make test-tenant

# Run specific test types
make test-tenant-unit      # Unit tests only
make test-tenant-e2e       # E2E tests only

# With coverage
make test-tenant-coverage

# With race detection
make test-tenant-race
```

### Advanced Testing

```bash
# Specific package
cd services/tenant-service
go test -v ./internal/repository/...

# Specific test
go test -v -run TestCreate ./internal/repository/...

# With verbose output
go test -v ./...

# Short mode (skip long tests)
go test -short ./...

# Parallel execution
go test -v -parallel 4 ./...
```

## 📈 Test Reports

### 1. Generate Comprehensive HTML Report

```bash
make test-tenant-report
```

**What You Get**:
- ✅ Interactive HTML dashboard
- ✅ Test results by package
- ✅ Coverage visualization
- ✅ Performance metrics
- ✅ Raw JSON data

**View Report**:
```bash
make view-tenant-report
# Or
open services/tenant-service/test-reports/latest/index.html
```

### 2. Coverage Report

```bash
make test-tenant-coverage
make view-tenant-coverage
```

Shows line-by-line coverage with green/red highlighting.

### 3. Generate Badges

```bash
make test-tenant-badges
```

Creates SVG badges for:
- Coverage percentage
- Total tests
- Build status

## 📝 Test Structure

```
services/tenant-service/
├── internal/
│   ├── repository/
│   │   ├── tenant_repository.go
│   │   ├── tenant_repository_test.go   ✅ 11 tests
│   │   └── mocks/
│   │       └── tenant_repository_mock.go
│   └── service/
│       ├── tenant_service.go
│       ├── tenant_service_test.go       ✅ 11 tests
│       └── ...
├── tests/
│   └── e2e/
│       └── tenant_api_test.go            ✅ 13 tests
└── scripts/
    ├── run_tests.sh                      # Complete test suite
    ├── generate_test_report.sh           # HTML reports
    └── generate_badges.sh                # Badge generation
```

## 🎯 Test Coverage

### By Package

| Package | Coverage | Tests |
|---------|----------|-------|
| internal/repository | 85%+ | 11 |
| internal/service | 82%+ | 11 |
| internal/api | 78%+ | Covered by E2E |
| tests/e2e | 100% | 13 |

### What's Tested

**Repository Layer**:
- ✅ Create, Read, Update, Delete operations
- ✅ Get by ID, Slug, Domain
- ✅ List with pagination
- ✅ Error handling
- ✅ Edge cases

**Service Layer**:
- ✅ Business logic
- ✅ Tier-based limits
- ✅ Slug generation
- ✅ Kafka event publishing
- ✅ Configuration management
- ✅ Mocked dependencies

**E2E/API Layer**:
- ✅ All 9 endpoints
- ✅ Success scenarios
- ✅ Error scenarios
- ✅ Validation
- ✅ Pagination
- ✅ Real HTTP requests

## 🔄 CI/CD Integration

### GitHub Actions

Workflow runs automatically on:
- Push to main/develop
- Pull requests
- Manual trigger

**What It Does**:
1. Runs all tests
2. Checks race conditions
3. Generates coverage
4. Validates 80% threshold
5. Comments on PRs
6. Uploads artifacts

**View Results**:
- GitHub → Actions tab
- Click latest run
- Download test-results or coverage-reports artifacts

### Local CI Simulation

```bash
make test-tenant-ci
```

Runs tests exactly like CI:
- JSON output
- Coverage tracking
- Race detection
- Results in `test-results.json`

## 📊 Example Test Results

### Console Output

```
=== RUN   TestTenantRepositoryTestSuite
=== RUN   TestTenantRepositoryTestSuite/TestCreate
=== RUN   TestTenantRepositoryTestSuite/TestGetByID
=== RUN   TestTenantRepositoryTestSuite/TestGetBySlug
--- PASS: TestTenantRepositoryTestSuite (0.15s)
    --- PASS: TestTenantRepositoryTestSuite/TestCreate (0.01s)
    --- PASS: TestTenantRepositoryTestSuite/TestGetByID (0.01s)
    --- PASS: TestTenantRepositoryTestSuite/TestGetBySlug (0.01s)

=== RUN   TestTenantServiceTestSuite
--- PASS: TestTenantServiceTestSuite (0.05s)

=== RUN   TestE2ETestSuite
--- PASS: TestE2ETestSuite (0.25s)

PASS
coverage: 82.5% of statements
ok      github.com/ecommerce/tenant-service/internal/repository    0.450s
```

### HTML Report Preview

The HTML report includes:

**Summary Dashboard**:
```
Total Tests:    35
Passed:         35 ✅
Failed:         0
Skipped:        0
Coverage:       82.5%
Duration:       2.45s
```

**Test Results**:
```
✅ internal/repository
   ✅ TestCreate (0.010s)
   ✅ TestGetByID (0.012s)
   ✅ TestList (0.020s)

✅ internal/service
   ✅ TestCreateTenant_Success (0.008s)
   ✅ TestGetTenant_Success (0.005s)

✅ tests/e2e
   ✅ TestCreateTenant_Success (0.025s)
   ✅ TestGetTenant_Success (0.020s)
```

## 🛠️ Writing Tests

### Repository Test Example

```go
func (suite *TenantRepositoryTestSuite) TestCreate() {
    ctx := context.Background()

    tenant := &models.Tenant{
        ID:     uuid.New().String(),
        Name:   "Test Store",
        Slug:   "test-store-123",
        Email:  "test@example.com",
        Status: models.StatusActive,
        Tier:   models.TierFree,
    }

    err := suite.repo.Create(ctx, tenant)
    assert.NoError(suite.T(), err)
    assert.NotEmpty(suite.T(), tenant.CreatedAt)
}
```

### Service Test with Mocks

```go
func (suite *TenantServiceTestSuite) TestCreateTenant_Success() {
    ctx := context.Background()

    req := &models.CreateTenantRequest{
        Name:  "Test Store",
        Email: "test@example.com",
        Tier:  "free",
    }

    // Mock repository
    suite.mockRepo.On("GetBySlug", ctx, mock.AnythingOfType("string")).
        Return(nil, errors.New("not found"))
    suite.mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Tenant")).
        Return(nil)

    // Mock Kafka
    suite.mockKafka.On("Publish", ctx, "tenant-events",
        mock.AnythingOfType("string"),
        mock.AnythingOfType("[]uint8")).
        Return(nil)

    result, err := suite.service.CreateTenant(ctx, req)

    assert.NoError(suite.T(), err)
    assert.NotNil(suite.T(), result)
    assert.Equal(suite.T(), "Test Store", result.Name)
}
```

### E2E Test Example

```go
func (suite *E2ETestSuite) TestCreateTenant_Success() {
    reqBody := models.CreateTenantRequest{
        Name:  "E2E Test Store",
        Email: "e2e@example.com",
        Tier:  "free",
    }

    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequest("POST", "/api/v1/tenants",
        bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)

    assert.Equal(suite.T(), http.StatusCreated, w.Code)

    var response models.TenantResponse
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(suite.T(), "E2E Test Store", response.Name)
}
```

## 📚 Best Practices

### 1. Test Naming

```go
// Good
func TestCreateTenant_Success(t *testing.T)
func TestCreateTenant_DuplicateSlug(t *testing.T)
func TestGetTenant_NotFound(t *testing.T)

// Bad
func TestCreate(t *testing.T)
func TestTenant(t *testing.T)
```

### 2. Test Organization

```go
// Use test suites for setup/teardown
type TenantServiceTestSuite struct {
    suite.Suite
    mockRepo  *mocks.MockTenantRepository
    service   TenantService
}

func (suite *TenantServiceTestSuite) SetupTest() {
    // Runs before each test
}

func (suite *TenantServiceTestSuite) TearDownTest() {
    // Runs after each test
}
```

### 3. Assertions

```go
// Use testify assertions
assert.NoError(t, err)
assert.Equal(t, expected, actual)
assert.NotNil(t, result)
assert.True(t, condition)

// Check mock expectations
suite.mockRepo.AssertExpectations(suite.T())
```

### 4. Test Data

```go
// Use realistic test data
tenant := &models.Tenant{
    Name:  "Acme Corp Store",
    Email: "store@acmecorp.com",
    Tier:  models.TierProfessional,
}

// Not
tenant := &models.Tenant{
    Name:  "test",
    Email: "test@test.com",
}
```

### 5. Coverage Goals

- Aim for **80%+** overall coverage
- **100%** for critical paths (payments, auth)
- **60%+** for utilities and helpers

## 🐛 Debugging Tests

### Run Specific Test

```bash
go test -v -run TestCreateTenant_Success ./internal/service/...
```

### With Detailed Output

```bash
go test -v -count=1 ./... # -count=1 disables caching
```

### With Debugging

```go
func TestDebug(t *testing.T) {
    // Print debug info
    t.Logf("Debug: value = %v", someValue)

    // Fail and continue
    t.Error("Something failed but continue")

    // Fail and stop
    t.Fatal("Critical failure, stop now")
}
```

## 🎓 Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Go Test Coverage](https://blog.golang.org/cover)

## ✅ Checklist

Before committing:

- [ ] All tests pass (`make test-tenant`)
- [ ] Coverage > 80% (`make test-tenant-coverage`)
- [ ] No race conditions (`make test-tenant-race`)
- [ ] New code has tests
- [ ] Tests are documented

Before releasing:

- [ ] Generate full report (`make test-tenant-report`)
- [ ] Update badges (`make test-tenant-badges`)
- [ ] Review coverage report
- [ ] CI/CD passing

---

**Happy Testing!** 🧪

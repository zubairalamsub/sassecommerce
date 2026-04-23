package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/config-service/internal/models"
	repoMocks "github.com/ecommerce/config-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (*configService, *repoMocks.MockConfigRepository) {
	mockRepo := new(repoMocks.MockConfigRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &configService{
		repo:   mockRepo,
		logger: logger,
	}

	return svc, mockRepo
}

// === GetConfig Tests ===

func TestGetConfig_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	entry := &models.ConfigEntry{
		ID: "c-1", Namespace: "business.vendor", Key: "default_commission_rate",
		Value: "10", ValueType: "number", Environment: "all",
	}
	mockRepo.On("Get", ctx, "business.vendor", "default_commission_rate", "all", "").Return(entry, nil)

	result, err := svc.GetConfig(ctx, "business.vendor", "default_commission_rate", "", "")

	assert.NoError(t, err)
	assert.Equal(t, "10", result.Value)
	assert.Equal(t, "business.vendor", result.Namespace)
}

func TestGetConfig_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("Get", ctx, "bad", "key", "all", "").Return(nil, errors.New("record not found"))

	result, err := svc.GetConfig(ctx, "bad", "key", "", "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetConfig_WithEnvironment(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	entry := &models.ConfigEntry{
		ID: "c-1", Namespace: "global", Key: "pagination.default_page_size",
		Value: "50", Environment: "production",
	}
	mockRepo.On("Get", ctx, "global", "pagination.default_page_size", "production", "").Return(entry, nil)

	result, err := svc.GetConfig(ctx, "global", "pagination.default_page_size", "production", "")

	assert.NoError(t, err)
	assert.Equal(t, "50", result.Value)
}

func TestGetConfig_SecretMasked(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	entry := &models.ConfigEntry{
		ID: "c-1", Namespace: "global", Key: "database.password",
		Value: "super-secret", IsSecret: true, Environment: "all",
	}
	mockRepo.On("Get", ctx, "global", "database.password", "all", "").Return(entry, nil)

	result, err := svc.GetConfig(ctx, "global", "database.password", "", "")

	assert.NoError(t, err)
	assert.Equal(t, "********", result.Value)
	assert.True(t, result.IsSecret)
}

// === SetConfig Tests ===

func TestSetConfig_CreateNew(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("Get", ctx, "test", "key1", "all", "").Return(nil, errors.New("not found"))
	mockRepo.On("Set", ctx, mock.AnythingOfType("*models.ConfigEntry")).Return(nil)
	mockRepo.On("RecordAudit", ctx, mock.AnythingOfType("*models.ConfigAuditLog")).Return(nil)

	req := &models.SetConfigRequest{
		Namespace: "test", Key: "key1", Value: "value1",
		Description: "Test config", UpdatedBy: "admin",
	}

	result, err := svc.SetConfig(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "value1", result.Value)
	assert.Equal(t, 1, result.Version)
}

func TestSetConfig_UpdateExisting(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	existing := &models.ConfigEntry{
		ID: "c-1", Namespace: "test", Key: "key1",
		Value: "old_value", Version: 1, Environment: "all",
	}
	mockRepo.On("Get", ctx, "test", "key1", "all", "").Return(existing, nil)
	mockRepo.On("Set", ctx, mock.AnythingOfType("*models.ConfigEntry")).Return(nil)
	mockRepo.On("RecordAudit", ctx, mock.AnythingOfType("*models.ConfigAuditLog")).Return(nil)

	req := &models.SetConfigRequest{
		Namespace: "test", Key: "key1", Value: "new_value", UpdatedBy: "admin",
	}

	result, err := svc.SetConfig(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "new_value", result.Value)
	assert.Equal(t, 2, result.Version)
}

func TestSetConfig_InvalidValueType(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := &models.SetConfigRequest{
		Namespace: "test", Key: "key1", Value: "val",
		ValueType: "invalid",
	}

	result, err := svc.SetConfig(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid value_type")
}

func TestSetConfig_DefaultsValueType(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("Get", ctx, "test", "key1", "all", "").Return(nil, errors.New("not found"))
	mockRepo.On("Set", ctx, mock.MatchedBy(func(e *models.ConfigEntry) bool {
		return e.ValueType == "string"
	})).Return(nil)
	mockRepo.On("RecordAudit", ctx, mock.AnythingOfType("*models.ConfigAuditLog")).Return(nil)

	req := &models.SetConfigRequest{
		Namespace: "test", Key: "key1", Value: "value1",
	}

	result, err := svc.SetConfig(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "string", result.ValueType)
}

func TestSetConfig_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("Get", ctx, "test", "key1", "all", "").Return(nil, errors.New("not found"))
	mockRepo.On("Set", ctx, mock.AnythingOfType("*models.ConfigEntry")).Return(errors.New("db error"))

	req := &models.SetConfigRequest{
		Namespace: "test", Key: "key1", Value: "value1",
	}

	result, err := svc.SetConfig(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSetConfig_WithTenant(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("Get", ctx, "tenant.features.free", "wishlist", "all", "tenant-1").Return(nil, errors.New("not found"))
	mockRepo.On("Set", ctx, mock.MatchedBy(func(e *models.ConfigEntry) bool {
		return e.TenantID == "tenant-1"
	})).Return(nil)
	mockRepo.On("RecordAudit", ctx, mock.AnythingOfType("*models.ConfigAuditLog")).Return(nil)

	req := &models.SetConfigRequest{
		Namespace: "tenant.features.free", Key: "wishlist",
		Value: "true", TenantID: "tenant-1",
	}

	result, err := svc.SetConfig(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "tenant-1", result.TenantID)
}

// === DeleteConfig Tests ===

func TestDeleteConfig_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	entry := &models.ConfigEntry{
		ID: "c-1", Namespace: "test", Key: "key1",
		Value: "val", Environment: "all",
	}
	mockRepo.On("GetByID", ctx, "c-1").Return(entry, nil)
	mockRepo.On("Delete", ctx, "c-1").Return(nil)
	mockRepo.On("RecordAudit", ctx, mock.AnythingOfType("*models.ConfigAuditLog")).Return(nil)

	err := svc.DeleteConfig(ctx, "c-1")

	assert.NoError(t, err)
}

func TestDeleteConfig_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "bad").Return(nil, errors.New("not found"))

	err := svc.DeleteConfig(ctx, "bad")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// === ListByNamespace Tests ===

func TestListByNamespace_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	entries := []models.ConfigEntry{
		{ID: "c-1", Namespace: "kafka", Key: "topics.order_events", Value: "order-events"},
		{ID: "c-2", Namespace: "kafka", Key: "topics.cart_events", Value: "cart-events"},
	}
	mockRepo.On("ListByNamespace", ctx, "kafka", "", "").Return(entries, nil)

	results, err := svc.ListByNamespace(ctx, "kafka", "", "")

	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestListByNamespace_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("ListByNamespace", ctx, "bad", "", "").Return([]models.ConfigEntry{}, errors.New("db error"))

	results, err := svc.ListByNamespace(ctx, "bad", "", "")

	assert.Error(t, err)
	assert.Nil(t, results)
}

// === ListNamespaces Tests ===

func TestListNamespaces_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	summaries := []models.NamespaceSummary{
		{Namespace: "global", Count: 15},
		{Namespace: "kafka", Count: 20},
		{Namespace: "business.vendor", Count: 5},
	}
	mockRepo.On("ListNamespaces", ctx).Return(summaries, nil)

	result, err := svc.ListNamespaces(ctx)

	assert.NoError(t, err)
	assert.Len(t, result.Namespaces, 3)
}

// === SearchConfigs Tests ===

func TestSearchConfigs_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	entries := []models.ConfigEntry{
		{ID: "c-1", Namespace: "business.shipping", Key: "carrier.fedex.base_rate", Value: "7.99"},
	}
	mockRepo.On("Search", ctx, "fedex", "", "", 1, 50).Return(entries, int64(1), nil)

	results, total, err := svc.SearchConfigs(ctx, "fedex", "", "", 1, 50)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
}

func TestSearchConfigs_DefaultPagination(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("Search", ctx, "test", "", "", 1, 50).Return([]models.ConfigEntry{}, int64(0), nil)

	results, total, err := svc.SearchConfigs(ctx, "test", "", "", 0, 0)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, results, 0)
}

// === BulkGet Tests ===

func TestBulkGet_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	keys := []models.NamespaceKey{
		{Namespace: "global", Key: "pagination.default_page_size"},
		{Namespace: "kafka", Key: "topics.order_events"},
	}
	entries := []models.ConfigEntry{
		{ID: "c-1", Namespace: "global", Key: "pagination.default_page_size", Value: "20"},
		{ID: "c-2", Namespace: "kafka", Key: "topics.order_events", Value: "order-events"},
	}
	mockRepo.On("BulkGet", ctx, keys, "all", "").Return(entries, nil)

	req := &models.BulkGetRequest{Keys: keys}
	results, err := svc.BulkGet(ctx, req, "", "")

	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

// === BulkSet Tests ===

func TestBulkSet_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("Get", ctx, mock.Anything, mock.Anything, "all", "").Return(nil, errors.New("not found"))
	mockRepo.On("Set", ctx, mock.AnythingOfType("*models.ConfigEntry")).Return(nil)
	mockRepo.On("RecordAudit", ctx, mock.AnythingOfType("*models.ConfigAuditLog")).Return(nil)

	req := &models.BulkSetRequest{
		Entries: []models.SetConfigRequest{
			{Namespace: "test", Key: "k1", Value: "v1"},
			{Namespace: "test", Key: "k2", Value: "v2"},
		},
		UpdatedBy: "admin",
	}

	results, err := svc.BulkSet(ctx, req)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestBulkSet_PartialFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	// First succeeds
	mockRepo.On("Get", ctx, "test", "k1", "all", "").Return(nil, errors.New("not found"))
	mockRepo.On("Set", ctx, mock.MatchedBy(func(e *models.ConfigEntry) bool {
		return e.Key == "k1"
	})).Return(nil)

	// Second fails
	mockRepo.On("Get", ctx, "test", "k2", "all", "").Return(nil, errors.New("not found"))
	mockRepo.On("Set", ctx, mock.MatchedBy(func(e *models.ConfigEntry) bool {
		return e.Key == "k2"
	})).Return(errors.New("db error"))

	mockRepo.On("RecordAudit", ctx, mock.AnythingOfType("*models.ConfigAuditLog")).Return(nil)

	req := &models.BulkSetRequest{
		Entries: []models.SetConfigRequest{
			{Namespace: "test", Key: "k1", Value: "v1"},
			{Namespace: "test", Key: "k2", Value: "v2"},
		},
	}

	results, err := svc.BulkSet(ctx, req)

	assert.NoError(t, err)
	assert.Len(t, results, 1) // Only successful ones
}

// === GetAuditLog Tests ===

func TestGetAuditLog_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	logs := []models.ConfigAuditLog{
		{ID: "a-1", ConfigID: "c-1", Namespace: "test", Key: "k1", Action: "create", NewValue: "v1", CreatedAt: time.Now()},
	}
	mockRepo.On("GetAuditLog", ctx, "test", "k1", 1, 50).Return(logs, int64(1), nil)

	results, total, err := svc.GetAuditLog(ctx, "test", "k1", 1, 50)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "create", results[0].Action)
}

// === GetConfigHistory Tests ===

func TestGetConfigHistory_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	logs := []models.ConfigAuditLog{
		{ID: "a-1", ConfigID: "c-1", Action: "update", OldValue: "10", NewValue: "15"},
		{ID: "a-2", ConfigID: "c-1", Action: "create", NewValue: "10"},
	}
	mockRepo.On("GetAuditByConfigID", ctx, "c-1", 1, 50).Return(logs, int64(2), nil)

	results, total, err := svc.GetConfigHistory(ctx, "c-1", 1, 50)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)
}

// === ExportNamespace Tests ===

func TestExportNamespace_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	entries := []models.ConfigEntry{
		{ID: "c-1", Namespace: "business.shipping", Key: "carrier.fedex.base_rate", Value: "7.99"},
		{ID: "c-2", Namespace: "business.shipping", Key: "carrier.ups.base_rate", Value: "7.49"},
	}
	mockRepo.On("ListByNamespace", ctx, "business.shipping", "", "").Return(entries, nil)

	results, err := svc.ExportNamespace(ctx, "business.shipping", "", "")

	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

// === Helper Tests ===

func TestToConfigResponse_Secret(t *testing.T) {
	entry := &models.ConfigEntry{
		ID: "c-1", Namespace: "global", Key: "db.password",
		Value: "my-password", IsSecret: true,
	}

	resp := toConfigResponse(entry)
	assert.Equal(t, "********", resp.Value)
}

func TestToConfigResponse_Normal(t *testing.T) {
	entry := &models.ConfigEntry{
		ID: "c-1", Namespace: "global", Key: "page_size",
		Value: "20", ValueType: "number", Version: 3,
	}

	resp := toConfigResponse(entry)
	assert.Equal(t, "20", resp.Value)
	assert.Equal(t, 3, resp.Version)
}

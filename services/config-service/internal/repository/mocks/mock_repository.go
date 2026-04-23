package mocks

import (
	"context"

	"github.com/ecommerce/config-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockConfigRepository struct {
	mock.Mock
}

func (m *MockConfigRepository) Get(ctx context.Context, namespace, key, environment, tenantID string) (*models.ConfigEntry, error) {
	args := m.Called(ctx, namespace, key, environment, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigEntry), args.Error(1)
}

func (m *MockConfigRepository) Set(ctx context.Context, entry *models.ConfigEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockConfigRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockConfigRepository) GetByID(ctx context.Context, id string) (*models.ConfigEntry, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigEntry), args.Error(1)
}

func (m *MockConfigRepository) ListByNamespace(ctx context.Context, namespace, environment, tenantID string) ([]models.ConfigEntry, error) {
	args := m.Called(ctx, namespace, environment, tenantID)
	return args.Get(0).([]models.ConfigEntry), args.Error(1)
}

func (m *MockConfigRepository) ListNamespaces(ctx context.Context) ([]models.NamespaceSummary, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.NamespaceSummary), args.Error(1)
}

func (m *MockConfigRepository) BulkGet(ctx context.Context, keys []models.NamespaceKey, environment, tenantID string) ([]models.ConfigEntry, error) {
	args := m.Called(ctx, keys, environment, tenantID)
	return args.Get(0).([]models.ConfigEntry), args.Error(1)
}

func (m *MockConfigRepository) Search(ctx context.Context, query, namespace, environment string, page, pageSize int) ([]models.ConfigEntry, int64, error) {
	args := m.Called(ctx, query, namespace, environment, page, pageSize)
	return args.Get(0).([]models.ConfigEntry), args.Get(1).(int64), args.Error(2)
}

func (m *MockConfigRepository) RecordAudit(ctx context.Context, log *models.ConfigAuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockConfigRepository) GetAuditLog(ctx context.Context, namespace, key string, page, pageSize int) ([]models.ConfigAuditLog, int64, error) {
	args := m.Called(ctx, namespace, key, page, pageSize)
	return args.Get(0).([]models.ConfigAuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockConfigRepository) GetAuditByConfigID(ctx context.Context, configID string, page, pageSize int) ([]models.ConfigAuditLog, int64, error) {
	args := m.Called(ctx, configID, page, pageSize)
	return args.Get(0).([]models.ConfigAuditLog), args.Get(1).(int64), args.Error(2)
}

// === MockMenuRepository ===

type MockMenuRepository struct {
	mock.Mock
}

func (m *MockMenuRepository) CreateMenu(ctx context.Context, menu *models.Menu) error {
	args := m.Called(ctx, menu)
	return args.Error(0)
}

func (m *MockMenuRepository) GetMenu(ctx context.Context, id string) (*models.Menu, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Menu), args.Error(1)
}

func (m *MockMenuRepository) GetMenuBySlug(ctx context.Context, tenantID, slug string) (*models.Menu, error) {
	args := m.Called(ctx, tenantID, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Menu), args.Error(1)
}

func (m *MockMenuRepository) UpdateMenu(ctx context.Context, menu *models.Menu) error {
	args := m.Called(ctx, menu)
	return args.Error(0)
}

func (m *MockMenuRepository) DeleteMenu(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMenuRepository) ListMenus(ctx context.Context, tenantID string) ([]models.Menu, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]models.Menu), args.Error(1)
}

func (m *MockMenuRepository) ListMenusByLocation(ctx context.Context, tenantID, location string) ([]models.Menu, error) {
	args := m.Called(ctx, tenantID, location)
	return args.Get(0).([]models.Menu), args.Error(1)
}

func (m *MockMenuRepository) CreateMenuItem(ctx context.Context, item *models.MenuItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockMenuRepository) GetMenuItem(ctx context.Context, id string) (*models.MenuItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MenuItem), args.Error(1)
}

func (m *MockMenuRepository) UpdateMenuItem(ctx context.Context, item *models.MenuItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockMenuRepository) DeleteMenuItem(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMenuRepository) GetMenuItems(ctx context.Context, menuID string) ([]models.MenuItem, error) {
	args := m.Called(ctx, menuID)
	return args.Get(0).([]models.MenuItem), args.Error(1)
}

func (m *MockMenuRepository) BulkUpdatePositions(ctx context.Context, items []models.MenuItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

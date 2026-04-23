package service

import (
	"context"
	"fmt"

	"github.com/ecommerce/config-service/internal/models"
	"github.com/ecommerce/config-service/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ConfigService interface {
	// CRUD
	GetConfig(ctx context.Context, namespace, key, environment, tenantID string) (*models.ConfigEntryResponse, error)
	SetConfig(ctx context.Context, req *models.SetConfigRequest) (*models.ConfigEntryResponse, error)
	DeleteConfig(ctx context.Context, id string) error

	// Listing
	ListByNamespace(ctx context.Context, namespace, environment, tenantID string) ([]models.ConfigEntryResponse, error)
	ListNamespaces(ctx context.Context) (*models.NamespaceListResponse, error)
	SearchConfigs(ctx context.Context, query, namespace, environment string, page, pageSize int) ([]models.ConfigEntryResponse, int64, error)

	// Bulk operations
	BulkGet(ctx context.Context, req *models.BulkGetRequest, environment, tenantID string) ([]models.ConfigEntryResponse, error)
	BulkSet(ctx context.Context, req *models.BulkSetRequest) ([]models.ConfigEntryResponse, error)

	// Audit
	GetAuditLog(ctx context.Context, namespace, key string, page, pageSize int) ([]models.ConfigAuditResponse, int64, error)
	GetConfigHistory(ctx context.Context, configID string, page, pageSize int) ([]models.ConfigAuditResponse, int64, error)

	// Export
	ExportNamespace(ctx context.Context, namespace, environment, tenantID string) ([]models.ConfigEntryResponse, error)
}

type configService struct {
	repo   repository.ConfigRepository
	logger *logrus.Logger
}

func NewConfigService(repo repository.ConfigRepository, logger *logrus.Logger) ConfigService {
	return &configService{
		repo:   repo,
		logger: logger,
	}
}

func (s *configService) GetConfig(ctx context.Context, namespace, key, environment, tenantID string) (*models.ConfigEntryResponse, error) {
	if environment == "" {
		environment = "all"
	}

	entry, err := s.repo.Get(ctx, namespace, key, environment, tenantID)
	if err != nil {
		return nil, fmt.Errorf("config not found: %s.%s", namespace, key)
	}

	return toConfigResponse(entry), nil
}

func (s *configService) SetConfig(ctx context.Context, req *models.SetConfigRequest) (*models.ConfigEntryResponse, error) {
	if req.ValueType == "" {
		req.ValueType = models.TypeString
	}
	if req.Environment == "" {
		req.Environment = "all"
	}
	if req.TenantID == "" {
		req.TenantID = ""
	}

	validTypes := map[string]bool{
		models.TypeString: true, models.TypeNumber: true,
		models.TypeBoolean: true, models.TypeJSON: true,
	}
	if !validTypes[req.ValueType] {
		return nil, fmt.Errorf("invalid value_type: must be string, number, boolean, or json")
	}

	// Check if entry already exists
	existing, err := s.repo.Get(ctx, req.Namespace, req.Key, req.Environment, req.TenantID)

	var entry *models.ConfigEntry
	action := "create"
	oldValue := ""

	if err == nil && existing != nil {
		// Update existing
		action = "update"
		oldValue = existing.Value
		existing.Value = req.Value
		existing.ValueType = req.ValueType
		existing.Description = req.Description
		existing.IsSecret = req.IsSecret
		existing.Version++
		existing.UpdatedBy = req.UpdatedBy
		entry = existing
	} else {
		// Create new
		entry = &models.ConfigEntry{
			ID:          uuid.New().String(),
			Namespace:   req.Namespace,
			Key:         req.Key,
			Value:       req.Value,
			ValueType:   req.ValueType,
			Description: req.Description,
			Environment: req.Environment,
			TenantID:    req.TenantID,
			IsSecret:    req.IsSecret,
			Version:     1,
			UpdatedBy:   req.UpdatedBy,
		}
	}

	if err := s.repo.Set(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Record audit
	audit := &models.ConfigAuditLog{
		ID:          uuid.New().String(),
		ConfigID:    entry.ID,
		Namespace:   entry.Namespace,
		Key:         entry.Key,
		OldValue:    oldValue,
		NewValue:    entry.Value,
		Action:      action,
		ChangedBy:   req.UpdatedBy,
		Environment: entry.Environment,
		TenantID:    entry.TenantID,
	}
	s.repo.RecordAudit(ctx, audit)

	return toConfigResponse(entry), nil
}

func (s *configService) DeleteConfig(ctx context.Context, id string) error {
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("config not found")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	// Record audit
	audit := &models.ConfigAuditLog{
		ID:          uuid.New().String(),
		ConfigID:    entry.ID,
		Namespace:   entry.Namespace,
		Key:         entry.Key,
		OldValue:    entry.Value,
		NewValue:    "",
		Action:      "delete",
		ChangedBy:   "system",
		Environment: entry.Environment,
		TenantID:    entry.TenantID,
	}
	s.repo.RecordAudit(ctx, audit)

	return nil
}

func (s *configService) ListByNamespace(ctx context.Context, namespace, environment, tenantID string) ([]models.ConfigEntryResponse, error) {
	entries, err := s.repo.ListByNamespace(ctx, namespace, environment, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	return toConfigResponses(entries), nil
}

func (s *configService) ListNamespaces(ctx context.Context) (*models.NamespaceListResponse, error) {
	summaries, err := s.repo.ListNamespaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	return &models.NamespaceListResponse{Namespaces: summaries}, nil
}

func (s *configService) SearchConfigs(ctx context.Context, query, namespace, environment string, page, pageSize int) ([]models.ConfigEntryResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	entries, total, err := s.repo.Search(ctx, query, namespace, environment, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return toConfigResponses(entries), total, nil
}

func (s *configService) BulkGet(ctx context.Context, req *models.BulkGetRequest, environment, tenantID string) ([]models.ConfigEntryResponse, error) {
	if environment == "" {
		environment = "all"
	}

	entries, err := s.repo.BulkGet(ctx, req.Keys, environment, tenantID)
	if err != nil {
		return nil, err
	}

	return toConfigResponses(entries), nil
}

func (s *configService) BulkSet(ctx context.Context, req *models.BulkSetRequest) ([]models.ConfigEntryResponse, error) {
	var results []models.ConfigEntryResponse

	for i := range req.Entries {
		if req.Entries[i].UpdatedBy == "" {
			req.Entries[i].UpdatedBy = req.UpdatedBy
		}
		resp, err := s.SetConfig(ctx, &req.Entries[i])
		if err != nil {
			s.logger.WithError(err).WithField("key", req.Entries[i].Namespace+"."+req.Entries[i].Key).Warn("Failed to set config in bulk")
			continue
		}
		results = append(results, *resp)
	}

	return results, nil
}

func (s *configService) GetAuditLog(ctx context.Context, namespace, key string, page, pageSize int) ([]models.ConfigAuditResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	logs, total, err := s.repo.GetAuditLog(ctx, namespace, key, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return toAuditResponses(logs), total, nil
}

func (s *configService) GetConfigHistory(ctx context.Context, configID string, page, pageSize int) ([]models.ConfigAuditResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	logs, total, err := s.repo.GetAuditByConfigID(ctx, configID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return toAuditResponses(logs), total, nil
}

func (s *configService) ExportNamespace(ctx context.Context, namespace, environment, tenantID string) ([]models.ConfigEntryResponse, error) {
	return s.ListByNamespace(ctx, namespace, environment, tenantID)
}

// === Helpers ===

func toConfigResponse(entry *models.ConfigEntry) *models.ConfigEntryResponse {
	value := entry.Value
	if entry.IsSecret {
		value = "********"
	}
	return &models.ConfigEntryResponse{
		ID:          entry.ID,
		Namespace:   entry.Namespace,
		Key:         entry.Key,
		Value:       value,
		ValueType:   entry.ValueType,
		Description: entry.Description,
		Environment: entry.Environment,
		TenantID:    entry.TenantID,
		IsSecret:    entry.IsSecret,
		Version:     entry.Version,
		UpdatedAt:   entry.UpdatedAt,
		UpdatedBy:   entry.UpdatedBy,
	}
}

func toConfigResponses(entries []models.ConfigEntry) []models.ConfigEntryResponse {
	responses := make([]models.ConfigEntryResponse, len(entries))
	for i, e := range entries {
		responses[i] = *toConfigResponse(&e)
	}
	return responses
}

func toAuditResponses(logs []models.ConfigAuditLog) []models.ConfigAuditResponse {
	responses := make([]models.ConfigAuditResponse, len(logs))
	for i, l := range logs {
		responses[i] = models.ConfigAuditResponse{
			ID:          l.ID,
			ConfigID:    l.ConfigID,
			Namespace:   l.Namespace,
			Key:         l.Key,
			OldValue:    l.OldValue,
			NewValue:    l.NewValue,
			Action:      l.Action,
			ChangedBy:   l.ChangedBy,
			Environment: l.Environment,
			TenantID:    l.TenantID,
			CreatedAt:   l.CreatedAt,
		}
	}
	return responses
}

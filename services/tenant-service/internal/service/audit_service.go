package service

import (
	"context"

	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/ecommerce/tenant-service/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type AuditService interface {
	CreateAuditLog(ctx context.Context, req *models.CreateAuditLogRequest) error
	GetAuditLogs(ctx context.Context, filters repository.AuditFilters) ([]models.AuditLog, int64, error)
}

type auditService struct {
	repo   repository.AuditRepository
	logger *logrus.Logger
}

func NewAuditService(repo repository.AuditRepository, logger *logrus.Logger) AuditService {
	return &auditService{
		repo:   repo,
		logger: logger,
	}
}

func (s *auditService) CreateAuditLog(ctx context.Context, req *models.CreateAuditLogRequest) error {
	auditLog := &models.AuditLog{
		ID:           uuid.New().String(),
		TenantID:     req.TenantID,
		UserID:       req.UserID,
		Action:       string(req.Action),
		Resource:     string(req.Resource),
		ResourceID:   req.ResourceID,
		Method:       req.Method,
		Path:         req.Path,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		RequestBody:  req.RequestBody,
		ResponseCode: req.ResponseCode,
		OldValue:     repository.ToJSONString(req.OldValue),
		NewValue:     repository.ToJSONString(req.NewValue),
		Metadata:     repository.ToJSONString(req.Metadata),
		ErrorMessage: req.ErrorMessage,
		Duration:     req.Duration,
	}

	if err := s.repo.Create(ctx, auditLog); err != nil {
		s.logger.WithError(err).Error("Failed to create audit log")
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"audit_id":    auditLog.ID,
		"action":      auditLog.Action,
		"resource":    auditLog.Resource,
		"resource_id": auditLog.ResourceID,
		"tenant_id":   auditLog.TenantID,
	}).Info("Audit log created")

	return nil
}

func (s *auditService) GetAuditLogs(ctx context.Context, filters repository.AuditFilters) ([]models.AuditLog, int64, error) {
	return s.repo.List(ctx, filters)
}

package service

import (
	"context"
	"time"

	"github.com/ecommerce/tenant-service/internal/repository"
	"github.com/sirupsen/logrus"
)

// UsageService exposes aggregated tenant usage metrics.
type UsageService interface {
	// GetTenantUsage aggregates audit-log usage per tenant; a non-nil since
	// bounds it to logs created at/after that instant, nil means all-time.
	GetTenantUsage(ctx context.Context, since *time.Time) ([]repository.TenantUsageRow, error)
}

type usageService struct {
	repo   repository.UsageRepository
	logger *logrus.Logger
}

func NewUsageService(repo repository.UsageRepository, logger *logrus.Logger) UsageService {
	return &usageService{
		repo:   repo,
		logger: logger,
	}
}

func (s *usageService) GetTenantUsage(ctx context.Context, since *time.Time) ([]repository.TenantUsageRow, error) {
	rows, err := s.repo.GetTenantUsage(ctx, since)
	if err != nil {
		s.logger.WithError(err).Error("Failed to load tenant usage report")
		return nil, err
	}
	return rows, nil
}

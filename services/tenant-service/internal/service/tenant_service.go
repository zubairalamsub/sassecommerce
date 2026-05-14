package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/ecommerce/tenant-service/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// KafkaPublisher defines the interface for publishing messages to Kafka
type KafkaPublisher interface {
	Publish(ctx context.Context, topic, key string, value []byte) error
}

type TenantService interface {
	CreateTenant(ctx context.Context, req *models.CreateTenantRequest) (*models.TenantResponse, error)
	GetTenant(ctx context.Context, id string) (*models.TenantResponse, error)
	GetTenantBySlug(ctx context.Context, slug string) (*models.TenantResponse, error)
	GetTenantByDomain(ctx context.Context, domain string) (*models.TenantResponse, error)
	ListTenants(ctx context.Context, page, pageSize int) ([]models.TenantResponse, int64, error)
	UpdateTenant(ctx context.Context, id string, req *models.UpdateTenantRequest) (*models.TenantResponse, error)
	DeleteTenant(ctx context.Context, id string) error
	UpdateTenantConfig(ctx context.Context, id string, config *models.TenantConfig) error
}

type tenantService struct {
	repo          repository.TenantRepository
	kafkaProducer KafkaPublisher
	logger        *logrus.Logger
}

func NewTenantService(repo repository.TenantRepository, kafkaProducer KafkaPublisher, logger *logrus.Logger) TenantService {
	return &tenantService{
		repo:          repo,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

func (s *tenantService) CreateTenant(ctx context.Context, req *models.CreateTenantRequest) (*models.TenantResponse, error) {
	// Generate slug from name
	slug := generateSlug(req.Name)

	// Check if slug already exists
	existing, _ := s.repo.GetBySlug(ctx, slug)
	if existing != nil {
		return nil, errors.New("tenant with similar name already exists")
	}

	// Determine limits based on tier
	maxUsers, maxProducts, maxOrders := getTierLimits(models.TenantTier(req.Tier))

	// Determine database strategy based on tier
	dbStrategy := getDatabaseStrategy(models.TenantTier(req.Tier))

	// Create tenant
	tenant := &models.Tenant{
		ID:               uuid.New().String(),
		Name:             req.Name,
		Slug:             slug,
		Email:            req.Email,
		Status:           models.StatusPending,
		Tier:             models.TenantTier(req.Tier),
		MaxUsers:         maxUsers,
		MaxProducts:      maxProducts,
		MaxOrders:        maxOrders,
		DatabaseStrategy: dbStrategy,
		Config: models.TenantConfig{
			General: models.GeneralConfig{
				Timezone:     "Asia/Dhaka",
				Currency:     "BDT",
				Language:     "bn",
				DateFormat:   "DD/MM/YYYY",
				TimeFormat:   "12h",
				ContactEmail: req.Email,
			},
			Branding: models.BrandingConfig{
				PrimaryColor:   "#3b82f6",
				SecondaryColor: "#10b981",
			},
			Features: getDefaultFeatures(models.TenantTier(req.Tier)),
		},
	}

	// Set trial period for paid tiers
	if tenant.Tier != models.TierFree {
		trialEnds := time.Now().Add(14 * 24 * time.Hour) // 14 days trial
		tenant.TrialEndsAt = &trialEnds
	}

	// Save to database
	if err := s.repo.Create(ctx, tenant); err != nil {
		s.logger.WithError(err).Error("Failed to create tenant")
		return nil, err
	}

	// Publish TenantCreated event to Kafka
	event := map[string]interface{}{
		"event_id":   uuid.New().String(),
		"event_type": "TenantCreated",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    "1.0.0",
		"payload": map[string]interface{}{
			"tenant_id": tenant.ID,
			"name":      tenant.Name,
			"slug":      tenant.Slug,
			"tier":      tenant.Tier,
			"status":    tenant.Status,
		},
	}

	if err := s.publishEvent(ctx, event); err != nil {
		s.logger.WithError(err).Warn("Failed to publish tenant created event")
	}

	return toTenantResponse(tenant), nil
}

func (s *tenantService) GetTenant(ctx context.Context, id string) (*models.TenantResponse, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toTenantResponse(tenant), nil
}

func (s *tenantService) GetTenantBySlug(ctx context.Context, slug string) (*models.TenantResponse, error) {
	tenant, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return toTenantResponse(tenant), nil
}

func (s *tenantService) GetTenantByDomain(ctx context.Context, domain string) (*models.TenantResponse, error) {
	tenant, err := s.repo.GetByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	return toTenantResponse(tenant), nil
}

func (s *tenantService) ListTenants(ctx context.Context, page, pageSize int) ([]models.TenantResponse, int64, error) {
	tenants, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.TenantResponse, len(tenants))
	for i, tenant := range tenants {
		responses[i] = *toTenantResponse(&tenant)
	}

	return responses, total, nil
}

func (s *tenantService) UpdateTenant(ctx context.Context, id string, req *models.UpdateTenantRequest) (*models.TenantResponse, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Snapshot the pre-update status so we can emit a security event when
	// the tenant is suspended or reactivated.
	oldStatus := tenant.Status

	// Update fields
	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.Domain != nil {
		tenant.Domain = *req.Domain
	}
	if req.Status != nil {
		tenant.Status = models.TenantStatus(*req.Status)
	}
	if req.Config != nil {
		tenant.Config = *req.Config
	}

	if err := s.repo.Update(ctx, tenant); err != nil {
		s.logger.WithError(err).Error("Failed to update tenant")
		return nil, err
	}

	// Publish TenantUpdated event
	event := map[string]interface{}{
		"event_id":   uuid.New().String(),
		"event_type": "TenantUpdated",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    "1.0.0",
		"payload": map[string]interface{}{
			"tenant_id": tenant.ID,
			"name":      tenant.Name,
			"status":    tenant.Status,
		},
	}

	if err := s.publishEvent(ctx, event); err != nil {
		s.logger.WithError(err).Warn("Failed to publish tenant updated event")
	}

	// Additionally emit a status-specific security event when the tenant
	// gets suspended or reactivated. These are higher-signal than the
	// generic TenantUpdated and let the audit log surface them clearly.
	if tenant.Status != oldStatus {
		var statusEvent string
		switch tenant.Status {
		case models.StatusSuspended:
			statusEvent = "TenantSuspended"
		case models.StatusActive:
			if oldStatus == models.StatusSuspended {
				statusEvent = "TenantReactivated"
			}
		}
		if statusEvent != "" {
			s.publishSecurityEvent(ctx, statusEvent, map[string]interface{}{
				"tenant_id":  tenant.ID,
				"name":       tenant.Name,
				"status":     tenant.Status,
				"old_values": map[string]interface{}{"status": oldStatus},
			})
		}
	}

	return toTenantResponse(tenant), nil
}

func (s *tenantService) DeleteTenant(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Publish TenantDeleted event
	event := map[string]interface{}{
		"event_id":   uuid.New().String(),
		"event_type": "TenantDeleted",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    "1.0.0",
		"payload": map[string]interface{}{
			"tenant_id": id,
		},
	}

	if err := s.publishEvent(ctx, event); err != nil {
		s.logger.WithError(err).Warn("Failed to publish tenant deleted event")
	}

	return nil
}

// UpdateTenantConfig replaces the tenant's configuration. To keep the audit
// log readable (configs are large and noisy), the published event lists only
// the keys whose top-level value changed — not the full before/after blob.
func (s *tenantService) UpdateTenantConfig(ctx context.Context, id string, config *models.TenantConfig) error {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	changedKeys := diffTenantConfigKeys(&tenant.Config, config)

	tenant.Config = *config

	if err := s.repo.Update(ctx, tenant); err != nil {
		return err
	}

	// Only emit an event when something actually changed; otherwise this
	// can flood the audit log on idempotent PATCH retries.
	if len(changedKeys) > 0 {
		s.publishSecurityEvent(ctx, "TenantConfigChanged", map[string]interface{}{
			"tenant_id":    tenant.ID,
			"changed_keys": changedKeys,
		})
	}

	return nil
}

// diffTenantConfigKeys returns the names of the top-level config sections
// whose value changed between two TenantConfig snapshots. We intentionally
// stop at the section granularity (general, branding, features, …) so the
// audit log stays compact even when nested values change.
func diffTenantConfigKeys(oldCfg, newCfg *models.TenantConfig) []string {
	if oldCfg == nil || newCfg == nil {
		return nil
	}
	var changed []string
	oldBytes, _ := json.Marshal(oldCfg)
	newBytes, _ := json.Marshal(newCfg)
	var oldMap, newMap map[string]json.RawMessage
	_ = json.Unmarshal(oldBytes, &oldMap)
	_ = json.Unmarshal(newBytes, &newMap)
	for k, newVal := range newMap {
		oldVal, ok := oldMap[k]
		if !ok || string(oldVal) != string(newVal) {
			changed = append(changed, k)
		}
	}
	for k := range oldMap {
		if _, ok := newMap[k]; !ok {
			changed = append(changed, k)
		}
	}
	return changed
}

// Helper functions

func (s *tenantService) publishEvent(ctx context.Context, event map[string]interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return s.kafkaProducer.Publish(ctx, "tenant-events", string(event["event_id"].(string)), data)
}

// publishSecurityEvent is a thin wrapper that builds the standard envelope
// for security-relevant events (suspend, reactivate, config change) and
// publishes them to tenant-events. Errors are logged but never returned —
// auditing is best-effort and must not block the calling mutation.
func (s *tenantService) publishSecurityEvent(ctx context.Context, eventType string, payload map[string]interface{}) {
	event := map[string]interface{}{
		"event_id":   uuid.New().String(),
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    "1.0.0",
		"payload":    payload,
	}
	if err := s.publishEvent(ctx, event); err != nil {
		s.logger.WithError(err).WithField("event_type", eventType).Warn("Failed to publish tenant security event")
	}
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Add UUID suffix to ensure uniqueness
	return fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])
}

func getTierLimits(tier models.TenantTier) (int, int, int) {
	switch tier {
	case models.TierFree:
		return 5, 100, 1000
	case models.TierStarter:
		return 25, 1000, 10000
	case models.TierProfessional:
		return 100, 10000, 100000
	case models.TierEnterprise:
		return -1, -1, -1 // Unlimited
	default:
		return 5, 100, 1000
	}
}

func getDatabaseStrategy(tier models.TenantTier) string {
	switch tier {
	case models.TierFree, models.TierStarter:
		return "pool" // Shared database
	case models.TierProfessional:
		return "bridge" // Separate schema
	case models.TierEnterprise:
		return "silo" // Dedicated database
	default:
		return "pool"
	}
}

func getDefaultFeatures(tier models.TenantTier) models.FeatureConfig {
	features := models.FeatureConfig{
		GuestCheckout:     true,
		ProductReviews:    true,
		Wishlist:          tier != models.TierFree,
		MultiCurrency:     tier == models.TierProfessional || tier == models.TierEnterprise,
		SocialLogin:       tier == models.TierProfessional || tier == models.TierEnterprise,
		AIRecommendations: tier == models.TierProfessional || tier == models.TierEnterprise,
		LoyaltyProgram:    tier == models.TierProfessional || tier == models.TierEnterprise,
		Subscriptions:     tier == models.TierEnterprise,
		GiftCards:         tier == models.TierEnterprise,
	}
	return features
}

func toTenantResponse(tenant *models.Tenant) *models.TenantResponse {
	return &models.TenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		Slug:      tenant.Slug,
		Domain:    tenant.Domain,
		Email:     tenant.Email,
		Status:    tenant.Status,
		Tier:      tenant.Tier,
		Config:    tenant.Config,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	}
}

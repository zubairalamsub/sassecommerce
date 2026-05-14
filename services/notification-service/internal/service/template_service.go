// Package service - template_service.go provides CRUD and rendering for
// admin-managed notification templates. Templates live in MongoDB and override
// the hardcoded RenderEmailHTML calls in the Kafka consumer: when a tenant has
// an active template for a given (channel, type) pair the consumer uses it;
// otherwise it falls back to the hardcoded copy.
package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/repository"
	deftpl "github.com/ecommerce/notification-service/internal/templates"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TemplateService is the admin-facing API for managing notification templates.
type TemplateService interface {
	List(ctx context.Context, tenantID string) ([]models.NotificationTemplate, error)
	Get(ctx context.Context, id string) (*models.NotificationTemplate, error)
	Create(ctx context.Context, tenantID string, req *models.CreateTemplateRequest) (*models.NotificationTemplate, error)
	Update(ctx context.Context, id string, req *models.UpdateTemplateRequest) (*models.NotificationTemplate, error)
	Delete(ctx context.Context, id string) error
	// Preview renders the template with sample values; returns subject + body.
	Preview(ctx context.Context, id string, sampleVars map[string]interface{}) (*models.RenderedTemplate, error)
	// TestSend renders the template with sample values and dispatches an email
	// to the test address via the email provider.
	TestSend(ctx context.Context, id, email string, sampleVars map[string]interface{}) error
	// InstallDefaults seeds the tenant's template collection with the
	// pre-designed starter pack. When force=true any existing template for the
	// same (tenant_id, type, channel) tuple is overwritten; otherwise it is
	// left untouched and reported as skipped.
	InstallDefaults(ctx context.Context, tenantID string, force bool) (*InstallDefaultsResult, error)
}

// InstallDefaultsResult summarises the outcome of an install-defaults run.
// We return the touched templates so the admin UI can refresh the list
// without re-fetching from the API.
type InstallDefaultsResult struct {
	Created   int                            `json:"created"`
	Updated   int                            `json:"updated"`
	Skipped   int                            `json:"skipped"`
	Templates []models.NotificationTemplate  `json:"templates"`
}

type templateService struct {
	repo      repository.NotificationRepository
	providers map[models.Channel]NotificationProvider
	logger    *logrus.Logger
}

func NewTemplateService(repo repository.NotificationRepository, providers map[models.Channel]NotificationProvider, logger *logrus.Logger) TemplateService {
	return &templateService{repo: repo, providers: providers, logger: logger}
}

// validChannels — keep in sync with models.Channel*.
var validChannels = map[string]struct{}{
	string(models.ChannelEmail): {},
	string(models.ChannelSMS):   {},
	string(models.ChannelPush):  {},
}

func (s *templateService) List(ctx context.Context, tenantID string) ([]models.NotificationTemplate, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}
	return s.repo.ListTemplates(ctx, tenantID)
}

func (s *templateService) Get(ctx context.Context, id string) (*models.NotificationTemplate, error) {
	return s.repo.GetTemplate(ctx, id)
}

func (s *templateService) Create(ctx context.Context, tenantID string, req *models.CreateTemplateRequest) (*models.NotificationTemplate, error) {
	if err := s.validateCreate(tenantID, req); err != nil {
		return nil, err
	}
	t := &models.NotificationTemplate{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Type:            models.NotificationType(strings.TrimSpace(req.Type)),
		Channel:         models.Channel(strings.TrimSpace(req.Channel)),
		Name:            strings.TrimSpace(req.Name),
		SubjectTemplate: req.SubjectTemplate,
		BodyTemplate:    req.BodyTemplate,
		IsActive:        req.IsActive,
	}
	// Validate the templates parse before persisting — saves admins debugging
	// 500 errors at send time.
	if _, err := template.New("subject").Parse(t.SubjectTemplate); err != nil {
		return nil, fmt.Errorf("subject template invalid: %w", err)
	}
	if _, err := template.New("body").Parse(t.BodyTemplate); err != nil {
		return nil, fmt.Errorf("body template invalid: %w", err)
	}
	if err := s.repo.CreateTemplate(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *templateService) Update(ctx context.Context, id string, req *models.UpdateTemplateRequest) (*models.NotificationTemplate, error) {
	existing, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Type != nil {
		existing.Type = models.NotificationType(strings.TrimSpace(*req.Type))
	}
	if req.Channel != nil {
		ch := strings.TrimSpace(*req.Channel)
		if _, ok := validChannels[ch]; !ok {
			return nil, fmt.Errorf("invalid channel: %s", ch)
		}
		existing.Channel = models.Channel(ch)
	}
	if req.Name != nil {
		existing.Name = strings.TrimSpace(*req.Name)
	}
	if req.SubjectTemplate != nil {
		if _, err := template.New("subject").Parse(*req.SubjectTemplate); err != nil {
			return nil, fmt.Errorf("subject template invalid: %w", err)
		}
		existing.SubjectTemplate = *req.SubjectTemplate
	}
	if req.BodyTemplate != nil {
		if _, err := template.New("body").Parse(*req.BodyTemplate); err != nil {
			return nil, fmt.Errorf("body template invalid: %w", err)
		}
		existing.BodyTemplate = *req.BodyTemplate
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if err := s.repo.UpdateTemplate(ctx, id, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *templateService) Delete(ctx context.Context, id string) error {
	return s.repo.DeleteTemplate(ctx, id)
}

func (s *templateService) Preview(ctx context.Context, id string, sampleVars map[string]interface{}) (*models.RenderedTemplate, error) {
	t, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	vars := MergeSampleVars(t.Type, sampleVars)
	subject, body, err := RenderTemplate(t, vars)
	if err != nil {
		return nil, err
	}
	return &models.RenderedTemplate{Subject: subject, Body: body}, nil
}

func (s *templateService) TestSend(ctx context.Context, id, email string, sampleVars map[string]interface{}) error {
	if email == "" {
		return errors.New("email is required")
	}
	t, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return err
	}
	vars := MergeSampleVars(t.Type, sampleVars)
	subject, body, err := RenderTemplate(t, vars)
	if err != nil {
		return err
	}

	provider, ok := s.providers[t.Channel]
	if !ok {
		return fmt.Errorf("no provider configured for channel: %s", t.Channel)
	}
	now := time.Now().UTC()
	notif := &models.Notification{
		ID:        uuid.New().String(),
		TenantID:  t.TenantID,
		UserID:    "admin",
		Channel:   t.Channel,
		Type:      t.Type,
		Status:    models.StatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata: map[string]interface{}{
			"test_send":   true,
			"template_id": t.ID,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	result, err := provider.Send(notif)
	if err != nil {
		return fmt.Errorf("test send failed: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("test send failed: %s", result.Error)
	}
	return nil
}

// InstallDefaults seeds the tenant's notification_templates collection from the
// starter-pack registry. Existing templates are matched by (type, channel) —
// when force=false we skip them, when force=true we replace their content.
//
// The function streams results back via the InstallDefaultsResult so the admin
// UI can toast the user "Installed N, updated M, skipped K". We deliberately
// don't run this inside a transaction: each insert is small, the operation is
// idempotent w.r.t. (type, channel), and a partial failure leaves the tenant
// with a usable subset of templates rather than nothing.
func (s *templateService) InstallDefaults(ctx context.Context, tenantID string, force bool) (*InstallDefaultsResult, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	defaults := deftpl.Defaults()
	result := &InstallDefaultsResult{
		Templates: make([]models.NotificationTemplate, 0, len(defaults)),
	}

	// Pre-fetch every template for this tenant once; matching in-memory is
	// faster than N round-trips and avoids contending on the same index.
	existing, err := s.repo.ListTemplates(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list existing templates: %w", err)
	}
	type key struct {
		Type    models.NotificationType
		Channel models.Channel
	}
	byKey := make(map[key]*models.NotificationTemplate, len(existing))
	for i := range existing {
		k := key{Type: existing[i].Type, Channel: existing[i].Channel}
		byKey[k] = &existing[i]
	}

	now := time.Now().UTC()
	for _, def := range defaults {
		k := key{Type: def.Type, Channel: def.Channel}
		if dup, found := byKey[k]; found {
			if !force {
				result.Skipped++
				continue
			}
			// Overwrite: keep ID and CreatedAt so audit history is preserved;
			// updated_at is bumped by UpdateTemplate.
			dup.Name = def.Name
			dup.SubjectTemplate = def.SubjectTemplate
			dup.BodyTemplate = def.BodyTemplate
			dup.IsActive = true
			if err := s.repo.UpdateTemplate(ctx, dup.ID, dup); err != nil {
				s.logger.WithError(err).WithField("type", def.Type).Warn("Failed to update default template")
				continue
			}
			result.Updated++
			result.Templates = append(result.Templates, *dup)
			continue
		}

		t := &models.NotificationTemplate{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			Type:            def.Type,
			Channel:         def.Channel,
			Name:            def.Name,
			SubjectTemplate: def.SubjectTemplate,
			BodyTemplate:    def.BodyTemplate,
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := s.repo.CreateTemplate(ctx, t); err != nil {
			s.logger.WithError(err).WithField("type", def.Type).Warn("Failed to install default template")
			continue
		}
		result.Created++
		result.Templates = append(result.Templates, *t)
	}

	return result, nil
}

func (s *templateService) validateCreate(tenantID string, req *models.CreateTemplateRequest) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(req.Type) == "" {
		return errors.New("type is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	ch := strings.TrimSpace(req.Channel)
	if _, ok := validChannels[ch]; !ok {
		return fmt.Errorf("invalid channel: %s (must be email/sms/push)", ch)
	}
	if strings.TrimSpace(req.BodyTemplate) == "" {
		return errors.New("body_template is required")
	}
	return nil
}

// RenderTemplate substitutes vars into the subject + body of a template.
// Exported so the Kafka consumer can reuse the same rendering path.
//
// We deliberately use text/template (not html/template) because admins author
// the body field as raw HTML — escaping there would double-encode their copy.
func RenderTemplate(t *models.NotificationTemplate, vars map[string]interface{}) (string, string, error) {
	subject, err := renderOne("subject", t.SubjectTemplate, vars)
	if err != nil {
		return "", "", fmt.Errorf("subject: %w", err)
	}
	body, err := renderOne("body", t.BodyTemplate, vars)
	if err != nil {
		return "", "", fmt.Errorf("body: %w", err)
	}
	return subject, body, nil
}

func renderOne(name, tmpl string, vars map[string]interface{}) (string, error) {
	if tmpl == "" {
		return "", nil
	}
	parsed, err := template.New(name).Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := parsed.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// MergeSampleVars layers sensible defaults under any explicit sample vars an
// admin provided in the preview request. Defaults vary by NotificationType so
// the preview matches what the consumer will inject at runtime.
func MergeSampleVars(notifType models.NotificationType, override map[string]interface{}) map[string]interface{} {
	base := map[string]interface{}{
		"TenantName":      "Saajan Store",
		"BrandColor":      "#006A4E",
		"FrontendBaseURL": "https://shop.example.com",
		"UserName":        "Sample Customer",
		"CustomerName":    "Sample Customer",
	}

	switch notifType {
	case models.TypeEmailVerification:
		base["VerifyURL"] = "https://shop.example.com/verify-email?token=sample-token"
	case models.TypePasswordReset:
		base["ResetURL"] = "https://shop.example.com/reset-password?token=sample-token"
	case models.TypeOrderConfirmation, models.TypeReceipt:
		base["OrderID"] = "ORD-12345"
		base["Total"] = "৳1,250.00"
		base["PaymentMethod"] = "bKash"
		base["Items"] = []map[string]interface{}{
			{"Name": "T-shirt (M)", "Quantity": 2, "Price": "৳400.00", "Subtotal": "৳800.00"},
			{"Name": "Sample Mug", "Quantity": 1, "Price": "৳450.00", "Subtotal": "৳450.00"},
		}
	case models.TypeOrderShipped:
		base["OrderID"] = "ORD-12345"
		base["TrackingNumber"] = "TRK-987654"
		base["Carrier"] = "Pathao Courier"
	case models.TypePaymentConfirmed:
		base["OrderID"] = "ORD-12345"
		base["Total"] = "৳1,250.00"
	case models.TypeStockAlert:
		base["ProductName"] = "Sample T-shirt"
		base["SKU"] = "SKU-ABC-001"
		base["CurrentQuantity"] = 5
	}

	for k, v := range override {
		base[k] = v
	}
	return base
}

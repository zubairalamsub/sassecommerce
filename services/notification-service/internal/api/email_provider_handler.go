package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/repository"
	"github.com/ecommerce/notification-service/internal/service"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// EmailProviderHandler exposes the admin surface for configuring email
// delivery: which vendors a tenant sends through, in what order, and their
// credentials.
//
// Secrets are write-only. Nothing here ever returns a stored credential —
// responses report only whether one is set plus a hint derived from the
// ciphertext — so a compromised admin session cannot be used to read the
// tenant's vendor keys back out.
type EmailProviderHandler struct {
	repo     repository.EmailProviderRepository
	resolver *service.EmailProviderResolver
	logger   *logrus.Logger
}

func NewEmailProviderHandler(
	repo repository.EmailProviderRepository,
	resolver *service.EmailProviderResolver,
	logger *logrus.Logger,
) *EmailProviderHandler {
	return &EmailProviderHandler{repo: repo, resolver: resolver, logger: logger}
}

// scopeFor resolves which configuration scope the request addresses.
//
// The platform default is addressable only by a super_admin, via
// ?scope=platform. Everyone else operates on their own tenant, taken from the
// verified JWT — never from a query or path parameter, so a tenant admin
// cannot reach another tenant's credentials or the platform's by asking.
func (h *EmailProviderHandler) scopeFor(c *gin.Context) (string, bool) {
	role, _ := c.Get("role")
	isSuperAdmin := role == "super_admin"

	// The platform branch is checked before any tenant requirement, because a
	// super_admin legitimately has no tenant: its JWT carries an empty
	// tenant_id. Requiring a tenant first would 401 the only role allowed to
	// configure the platform default, making that scope unreachable.
	if c.Query("scope") == "platform" {
		if !isSuperAdmin {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: "forbidden", Message: "only a super_admin may configure the platform default",
			})
			return "", false
		}
		return models.PlatformScope, true
	}

	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		// A tenant-less super_admin asking for tenant scope is a usable
		// request aimed at the wrong place, so say so rather than returning a
		// bare 401 that looks like an auth failure.
		if isSuperAdmin {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "tenant_required",
				Message: "a super_admin has no tenant of its own; use ?scope=platform to configure the platform default",
			})
			return "", false
		}
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return "", false
	}
	return tenantID, true
}

// ListProviders returns the effective chain for the caller's scope, plus the
// catalogue of provider keys the UI can offer.
//
// A tenant with no configuration of its own sees the platform default marked
// inherited, so the UI can render it read-only and make the precedence
// visible rather than leaving an operator guessing why their mail works.
func (h *EmailProviderHandler) ListProviders(c *gin.Context) {
	scope, ok := h.scopeFor(c)
	if !ok {
		return
	}

	configs, err := h.repo.List(c.Request.Context(), scope)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list email provider configs")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed_to_list_providers"})
		return
	}

	inherited := false
	if len(configs) == 0 && scope != models.PlatformScope {
		platform, err := h.repo.List(c.Request.Context(), models.PlatformScope)
		if err == nil && len(platform) > 0 {
			configs = platform
			inherited = true
		}
	}

	out := make([]*models.EmailProviderConfigResponse, 0, len(configs))
	for i := range configs {
		out = append(out, configs[i].ToResponse(inherited))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      out,
		"scope":     scope,
		"inherited": inherited,
		"available": service.EmailProviderNames(),
		// Surfaced so an operator can see at a glance whether the deployment
		// is actually encrypting credentials or only base64-encoding them.
		"encrypted_at_rest": h.repo.UsingEncryption(),
	})
}

// UpsertProvider creates or updates one provider in the caller's scope.
func (h *EmailProviderHandler) UpsertProvider(c *gin.Context) {
	scope, ok := h.scopeFor(c)
	if !ok {
		return
	}

	var req models.UpsertEmailProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if !knownProvider(provider) {
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "unknown_provider",
			Message: "provider must be one of: " + strings.Join(service.EmailProviderNames(), ", "),
		})
		return
	}

	existing, err := h.repo.Get(c.Request.Context(), scope, provider)
	if err != nil && !errors.Is(err, repository.ErrProviderConfigNotFound) {
		h.logger.WithError(err).Error("Failed to load existing provider config")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed_to_save_provider"})
		return
	}

	cfg := &models.EmailProviderConfig{TenantID: scope, Provider: provider}
	if existing != nil {
		cfg = existing
	}

	// Only overwrite what the request actually carried, so a partial update
	// (toggle Enabled, change priority) does not blank the rest.
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		cfg.Priority = *req.Priority
	}
	if req.Port != nil {
		cfg.Port = *req.Port
	}
	if req.Host != "" {
		cfg.Host = req.Host
	}
	if req.Username != "" {
		cfg.Username = req.Username
	}
	if req.FromEmail != "" {
		cfg.FromEmail = req.FromEmail
	}
	if req.FromName != "" {
		cfg.FromName = req.FromName
	}

	// An omitted secret keeps the stored one. cfg.Secret is left exactly as
	// loaded rather than blanked, so this does not depend on the repository
	// treating an empty value as "skip this field" — the existing ciphertext
	// is simply written back unchanged.
	if req.Secret != "" {
		sealed, err := h.repo.EncryptSecret(req.Secret)
		if err != nil {
			h.logger.WithError(err).Error("Failed to encrypt provider secret")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed_to_save_provider"})
			return
		}
		cfg.Secret = sealed
	}

	if err := h.repo.Upsert(c.Request.Context(), cfg); err != nil {
		h.logger.WithError(err).Error("Failed to save provider config")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed_to_save_provider"})
		return
	}

	// Drop the cached chain so the change takes effect on the next send rather
	// than whenever the TTL happens to lapse.
	if h.resolver != nil {
		h.resolver.Invalidate(scope)
	}

	saved, err := h.repo.Get(c.Request.Context(), scope, provider)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Provider saved"})
		return
	}

	// Never log the credential, but do record that it changed — rotating a
	// vendor key is exactly the kind of event worth an audit trail.
	h.logger.WithFields(logrus.Fields{
		"scope":          scope,
		"provider":       provider,
		"secret_changed": req.Secret != "",
	}).Info("Email provider configuration saved")

	c.JSON(http.StatusOK, saved.ToResponse(false))
}

// DeleteProvider removes a provider from the caller's scope.
func (h *EmailProviderHandler) DeleteProvider(c *gin.Context) {
	scope, ok := h.scopeFor(c)
	if !ok {
		return
	}

	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	if err := h.repo.Delete(c.Request.Context(), scope, provider); err != nil {
		if errors.Is(err, repository.ErrProviderConfigNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "provider_not_configured"})
			return
		}
		h.logger.WithError(err).Error("Failed to delete provider config")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed_to_delete_provider"})
		return
	}

	if h.resolver != nil {
		h.resolver.Invalidate(scope)
	}
	c.JSON(http.StatusOK, SuccessResponse{Message: "Provider removed"})
}

// TestProvider sends a probe message through one configured provider so an
// operator can verify credentials without waiting for real traffic.
//
// It deliberately targets a single provider rather than the chain: a chain
// would fall through to a working fallback and report success, which is
// exactly the wrong answer when you are trying to find out whether the
// credentials you just pasted are correct.
func (h *EmailProviderHandler) TestProvider(c *gin.Context) {
	scope, ok := h.scopeFor(c)
	if !ok {
		return
	}

	var req models.TestEmailProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	cfg, err := h.repo.Get(c.Request.Context(), scope, provider)
	if err != nil {
		if errors.Is(err, repository.ErrProviderConfigNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "provider_not_configured"})
			return
		}
		h.logger.WithError(err).Error("Failed to load provider for test")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "test_failed"})
		return
	}

	secret, err := h.repo.DecryptSecret(cfg.Secret)
	if err != nil {
		h.logger.WithError(err).Error("Failed to decrypt provider secret for test")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "test_failed",
			Message: "stored credential could not be decrypted; re-enter it",
		})
		return
	}

	live, err := service.BuildProviderFromConfig(cfg, secret, h.logger)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: "misconfigured", Message: err.Error()})
		return
	}

	result, err := live.Send(&models.Notification{
		TenantID:  scope,
		Channel:   models.ChannelEmail,
		Type:      models.TypeCustom,
		Recipient: req.To,
		Subject:   "Test email from your store",
		Body:      "<p>This is a test message confirming your email provider is configured correctly.</p>",
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "test_failed", Message: err.Error()})
		return
	}
	if result == nil || !result.Success {
		message := "provider rejected the message"
		if result != nil && result.Error != "" {
			message = result.Error
		}
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "test_failed", Message: message})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"provider":   result.ProviderName,
		"message_id": result.MessageID,
		"sent_to":    req.To,
	})
}

func knownProvider(name string) bool {
	for _, known := range service.EmailProviderNames() {
		if known == name {
			return true
		}
	}
	return false
}

// RegisterEmailProviderRoutes wires the admin endpoints. Every route is behind
// the router's auth middleware plus an admin role gate: these endpoints write
// vendor credentials, so a customer token must never reach them.
func RegisterEmailProviderRoutes(router *gin.Engine, h *EmailProviderHandler) {
	group := router.Group("/api/v1/email-providers")
	group.Use(sharedmiddleware.RequireRole("admin", "super_admin"))
	{
		group.GET("", h.ListProviders)
		group.PUT("", h.UpsertProvider)
		group.POST("/test", h.TestProvider)
		group.DELETE("/:provider", h.DeleteProvider)
	}
}

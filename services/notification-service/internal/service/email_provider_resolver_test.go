package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
)

// stubProviderRepo is an in-memory EmailProviderRepository.
type stubProviderRepo struct {
	byScope   map[string][]models.EmailProviderConfig
	listErr   error
	decErr    error
	listCalls int
}

func newStubProviderRepo() *stubProviderRepo {
	return &stubProviderRepo{byScope: map[string][]models.EmailProviderConfig{}}
}

func (r *stubProviderRepo) List(ctx context.Context, tenantID string) ([]models.EmailProviderConfig, error) {
	return r.byScope[tenantID], r.listErr
}

func (r *stubProviderRepo) ListEnabled(ctx context.Context, tenantID string) ([]models.EmailProviderConfig, error) {
	r.listCalls++
	if r.listErr != nil {
		return nil, r.listErr
	}
	out := make([]models.EmailProviderConfig, 0)
	for _, c := range r.byScope[tenantID] {
		if c.Enabled {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *stubProviderRepo) Get(ctx context.Context, tenantID, provider string) (*models.EmailProviderConfig, error) {
	for i, c := range r.byScope[tenantID] {
		if c.Provider == provider {
			return &r.byScope[tenantID][i], nil
		}
	}
	return nil, errors.New("not found")
}

func (r *stubProviderRepo) Upsert(ctx context.Context, cfg *models.EmailProviderConfig) error {
	r.byScope[cfg.TenantID] = append(r.byScope[cfg.TenantID], *cfg)
	return nil
}

func (r *stubProviderRepo) Delete(ctx context.Context, tenantID, provider string) error { return nil }
func (r *stubProviderRepo) EncryptSecret(p string) (string, error)                      { return "enc:" + p, nil }
func (r *stubProviderRepo) DecryptSecret(s string) (string, error) {
	if r.decErr != nil {
		return "", r.decErr
	}
	return s, nil
}
func (r *stubProviderRepo) UsingEncryption() bool { return true }

func smtpConfig(provider string, priority int, enabled bool) models.EmailProviderConfig {
	return models.EmailProviderConfig{
		TenantID: "tenant-1",
		Provider: provider,
		Enabled:  enabled,
		Priority: priority,
		Username: "user",
		Secret:   "secret",
	}
}

// A tenant with its own providers must use exactly those — not a mixture with
// the platform's. "I configured Mailjet, why did that go out via the
// platform's SendGrid" is a worse failure than one provider being down.
func TestResolverPrefersTenantOverPlatform(t *testing.T) {
	repo := newStubProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{smtpConfig("mailjet", 1, true)}
	platform := smtpConfig("brevo", 1, true)
	platform.TenantID = models.PlatformScope
	repo.byScope[models.PlatformScope] = []models.EmailProviderConfig{platform}

	resolver := NewEmailProviderResolver(repo, nil, quietLog())
	resolver.SetTTL(0)

	chain := resolver.Resolve(context.Background(), "tenant-1")

	if chain == nil {
		t.Fatal("chain = nil, want the tenant's own provider")
	}
	named, ok := chain.(interface{ Name() string })
	if !ok {
		t.Fatalf("chain = %T, want a single named provider", chain)
	}
	if named.Name() != "mailjet" {
		t.Errorf("provider = %q, want mailjet (the tenant's own)", named.Name())
	}
}

// A tenant with nothing configured inherits the platform default.
func TestResolverFallsBackToPlatform(t *testing.T) {
	repo := newStubProviderRepo()
	platform := smtpConfig("brevo", 1, true)
	platform.TenantID = models.PlatformScope
	repo.byScope[models.PlatformScope] = []models.EmailProviderConfig{platform}

	resolver := NewEmailProviderResolver(repo, nil, quietLog())
	resolver.SetTTL(0)

	chain := resolver.Resolve(context.Background(), "tenant-with-nothing")

	named, ok := chain.(interface{ Name() string })
	if !ok {
		t.Fatalf("chain = %T, want the platform provider", chain)
	}
	if named.Name() != "brevo" {
		t.Errorf("provider = %q, want the platform's brevo", named.Name())
	}
}

// With nothing in the database at all, the environment chain still carries
// mail — a Mongo problem must not stop delivery.
func TestResolverFallsBackToEnvironmentChain(t *testing.T) {
	repo := newStubProviderRepo()
	envChain := &scriptedProvider{name: "env-sendgrid", mode: "succeed"}

	resolver := NewEmailProviderResolver(repo, envChain, quietLog())
	resolver.SetTTL(0)

	if got := resolver.Resolve(context.Background(), "tenant-1"); got != envChain {
		t.Errorf("chain = %#v, want the environment fallback", got)
	}
}

// A database error must degrade to the fallback rather than dropping mail.
func TestResolverSurvivesRepositoryFailure(t *testing.T) {
	repo := newStubProviderRepo()
	repo.listErr = errors.New("mongo down")
	envChain := &scriptedProvider{name: "env", mode: "succeed"}

	resolver := NewEmailProviderResolver(repo, envChain, quietLog())
	resolver.SetTTL(0)

	if got := resolver.Resolve(context.Background(), "tenant-1"); got != envChain {
		t.Errorf("chain = %#v, want the environment fallback on a repo error", got)
	}
}

func TestResolverReturnsNilWhenNothingIsConfiguredAnywhere(t *testing.T) {
	resolver := NewEmailProviderResolver(newStubProviderRepo(), nil, quietLog())
	resolver.SetTTL(0)

	if got := resolver.Resolve(context.Background(), "tenant-1"); got != nil {
		t.Errorf("chain = %#v, want nil so the caller can decide", got)
	}
}

// Disabled rows are configuration an operator has switched off; they must not
// end up in the chain.
func TestResolverIgnoresDisabledProviders(t *testing.T) {
	repo := newStubProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{
		smtpConfig("mailjet", 1, false),
		smtpConfig("brevo", 2, true),
	}

	resolver := NewEmailProviderResolver(repo, nil, quietLog())
	resolver.SetTTL(0)

	named := resolver.Resolve(context.Background(), "tenant-1").(interface{ Name() string })
	if named.Name() != "brevo" {
		t.Errorf("provider = %q, want only the enabled brevo", named.Name())
	}
}

// Priority is the chain order, so it decides which vendor carries traffic.
func TestResolverOrdersByPriority(t *testing.T) {
	repo := newStubProviderRepo()
	// ListEnabled preserves insertion order here; the real repository sorts by
	// priority in Mongo. Inserted in priority order to mirror that.
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{
		smtpConfig("brevo", 1, true),
		smtpConfig("mailjet", 2, true),
	}

	resolver := NewEmailProviderResolver(repo, nil, quietLog())
	resolver.SetTTL(0)

	chain, ok := resolver.Resolve(context.Background(), "tenant-1").(*FailoverProvider)
	if !ok {
		t.Fatal("expected a failover chain for two providers")
	}
	if chain.Len() != 2 {
		t.Fatalf("chain length = %d, want 2", chain.Len())
	}
	if got := providerLabel(chain.providers[0], 0); got != "brevo" {
		t.Errorf("first in chain = %q, want brevo (priority 1)", got)
	}
}

// A row whose credentials cannot build a provider is skipped, not fatal — one
// broken entry must not take out the rest of the chain.
func TestResolverSkipsMisconfiguredRows(t *testing.T) {
	repo := newStubProviderRepo()
	broken := smtpConfig("mailjet", 1, true)
	broken.Username = "" // SMTP needs a username
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{broken, smtpConfig("brevo", 2, true)}

	resolver := NewEmailProviderResolver(repo, nil, quietLog())
	resolver.SetTTL(0)

	named, ok := resolver.Resolve(context.Background(), "tenant-1").(interface{ Name() string })
	if !ok {
		t.Fatalf("expected the surviving provider, got %T", resolver.Resolve(context.Background(), "tenant-1"))
	}
	if named.Name() != "brevo" {
		t.Errorf("provider = %q, want brevo after skipping the broken row", named.Name())
	}
}

// A credential that will not decrypt (key rotated, corrupt row) is skipped
// rather than crashing the send path.
func TestResolverSkipsUndecryptableSecret(t *testing.T) {
	repo := newStubProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{smtpConfig("mailjet", 1, true)}
	repo.decErr = errors.New("cipher: message authentication failed")
	envChain := &scriptedProvider{name: "env", mode: "succeed"}

	resolver := NewEmailProviderResolver(repo, envChain, quietLog())
	resolver.SetTTL(0)

	if got := resolver.Resolve(context.Background(), "tenant-1"); got != envChain {
		t.Errorf("chain = %#v, want the fallback when the secret cannot be decrypted", got)
	}
}

// Caching exists because one send can fan out; it must actually cache.
func TestResolverCachesWithinTTL(t *testing.T) {
	repo := newStubProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{smtpConfig("mailjet", 1, true)}

	resolver := NewEmailProviderResolver(repo, nil, quietLog())
	resolver.SetTTL(time.Minute)

	resolver.Resolve(context.Background(), "tenant-1")
	callsAfterFirst := repo.listCalls
	resolver.Resolve(context.Background(), "tenant-1")

	if repo.listCalls != callsAfterFirst {
		t.Errorf("repo hit %d times on the second resolve, want the cached chain", repo.listCalls-callsAfterFirst)
	}
}

// The UI's whole promise is that a saved change takes effect now, so an
// invalidation has to re-read.
func TestResolverInvalidateForcesReload(t *testing.T) {
	repo := newStubProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{smtpConfig("mailjet", 1, true)}

	resolver := NewEmailProviderResolver(repo, nil, quietLog())
	resolver.SetTTL(time.Minute)

	resolver.Resolve(context.Background(), "tenant-1")
	before := repo.listCalls
	resolver.Invalidate("tenant-1")
	resolver.Resolve(context.Background(), "tenant-1")

	if repo.listCalls <= before {
		t.Error("Invalidate did not force a re-read")
	}
}

// The platform default underpins every tenant that has not overridden it, so
// changing it must clear all cached chains, not just its own.
func TestInvalidatingPlatformClearsEveryTenant(t *testing.T) {
	repo := newStubProviderRepo()
	platform := smtpConfig("brevo", 1, true)
	platform.TenantID = models.PlatformScope
	repo.byScope[models.PlatformScope] = []models.EmailProviderConfig{platform}

	resolver := NewEmailProviderResolver(repo, nil, quietLog())
	resolver.SetTTL(time.Minute)

	resolver.Resolve(context.Background(), "tenant-a")
	resolver.Resolve(context.Background(), "tenant-b")
	before := repo.listCalls

	resolver.Invalidate(models.PlatformScope)
	resolver.Resolve(context.Background(), "tenant-a")
	resolver.Resolve(context.Background(), "tenant-b")

	if repo.listCalls <= before+1 {
		t.Errorf("repo hit %d more times, want both tenants to re-read", repo.listCalls-before)
	}
}

// ------------------------------------------------ BuildProviderFromConfig

func TestBuildProviderFromConfigUsesPresetCoordinates(t *testing.T) {
	cfg := &models.EmailProviderConfig{Provider: "mailjet", Username: "api-key"}

	provider, err := BuildProviderFromConfig(cfg, "api-secret", quietLog())
	if err != nil {
		t.Fatalf("BuildProviderFromConfig = %v", err)
	}

	smtpProvider, ok := provider.(*SMTPEmailProvider)
	if !ok {
		t.Fatalf("provider = %T, want *SMTPEmailProvider", provider)
	}
	if smtpProvider.cfg.Host != "in-v3.mailjet.com" || smtpProvider.cfg.Port != 587 {
		t.Errorf("relay = %s:%d, want mailjet's preset", smtpProvider.cfg.Host, smtpProvider.cfg.Port)
	}
	// Same table as the environment path, so UI-configured and env-configured
	// vendors resolve identically.
	if smtpProvider.cfg.Password != "api-secret" {
		t.Error("decrypted secret did not reach the provider")
	}
}

func TestBuildProviderFromConfigValidation(t *testing.T) {
	cases := []struct {
		name   string
		cfg    *models.EmailProviderConfig
		secret string
	}{
		{name: "unknown provider", cfg: &models.EmailProviderConfig{Provider: "carrierpigeon"}, secret: "s"},
		{name: "smtp without host", cfg: &models.EmailProviderConfig{Provider: "smtp", Username: "u"}, secret: "s"},
		{name: "smtp without username", cfg: &models.EmailProviderConfig{Provider: "mailjet"}, secret: "s"},
		{name: "smtp without secret", cfg: &models.EmailProviderConfig{Provider: "mailjet", Username: "u"}, secret: ""},
		{name: "sendgrid without key", cfg: &models.EmailProviderConfig{Provider: "sendgrid"}, secret: ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := BuildProviderFromConfig(c.cfg, c.secret, quietLog()); err == nil {
				t.Error("expected an error for an incomplete config")
			}
		})
	}
}

func TestBuildProviderFromConfigSimulatedNeedsNothing(t *testing.T) {
	provider, err := BuildProviderFromConfig(&models.EmailProviderConfig{Provider: "simulated"}, "", quietLog())
	if err != nil {
		t.Fatalf("BuildProviderFromConfig(simulated) = %v", err)
	}
	if provider == nil {
		t.Fatal("provider = nil")
	}
}

func TestBuildProviderFromConfigPort465ImplicitTLS(t *testing.T) {
	cfg := &models.EmailProviderConfig{Provider: "smtp", Host: "relay.test", Port: 465, Username: "u"}

	provider, err := BuildProviderFromConfig(cfg, "s", quietLog())
	if err != nil {
		t.Fatalf("BuildProviderFromConfig = %v", err)
	}
	if !provider.(*SMTPEmailProvider).cfg.ImplicitTLS {
		t.Error("port 465 should dial implicit TLS")
	}
}

package service

import (
	"context"
	"sync"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/repository"
	"github.com/sirupsen/logrus"
)

// EmailProviderResolver decides which provider chain a given tenant's mail
// goes through, in this precedence:
//
//  1. the tenant's own enabled configs, if it has any
//  2. otherwise the platform default (models.PlatformScope)
//  3. otherwise the startup chain from EMAIL_PROVIDERS
//
// Precedence is all-or-nothing at each level rather than merged per provider:
// a tenant that configures its own relay should send through exactly that, not
// through a surprise mixture of its own relay and the platform's. "I set up
// Mailjet, why did that go out via the platform's SendGrid" is a worse failure
// than "my one provider is down".
//
// Resolution hits Mongo, and one send can fan out to several notifications, so
// results are cached briefly. The TTL is short because the whole point of the
// UI is that a credential change takes effect without a restart.
type EmailProviderResolver struct {
	repo     repository.EmailProviderRepository
	fallback NotificationProvider
	logger   *logrus.Logger
	ttl      time.Duration

	mu    sync.RWMutex
	cache map[string]cachedChain
}

type cachedChain struct {
	provider NotificationProvider
	expires  time.Time
	// scope records where the chain came from, for logging.
	scope string
}

const defaultResolverTTL = 30 * time.Second

// NewEmailProviderResolver builds a resolver. fallback is the chain built from
// the environment at startup and may be nil.
func NewEmailProviderResolver(
	repo repository.EmailProviderRepository,
	fallback NotificationProvider,
	logger *logrus.Logger,
) *EmailProviderResolver {
	return &EmailProviderResolver{
		repo:     repo,
		fallback: fallback,
		logger:   logger,
		ttl:      defaultResolverTTL,
		cache:    make(map[string]cachedChain),
	}
}

// SetTTL overrides the cache lifetime. Zero disables caching, which tests and
// a "reload now" admin action both want.
func (r *EmailProviderResolver) SetTTL(ttl time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ttl = ttl
	r.cache = make(map[string]cachedChain)
}

// Invalidate drops a tenant's cached chain so the next send re-reads the
// database. Called after any write through the admin API, which is what makes
// a change in the UI take effect immediately rather than within the TTL.
func (r *EmailProviderResolver) Invalidate(tenantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tenantID == models.PlatformScope {
		// The platform default underpins every tenant that has not overridden
		// it, so there is no way to know which cached chains derived from it.
		r.cache = make(map[string]cachedChain)
		return
	}
	delete(r.cache, tenantID)
}

// Resolve returns the chain for a tenant, or nil if nothing is configured
// anywhere.
func (r *EmailProviderResolver) Resolve(ctx context.Context, tenantID string) NotificationProvider {
	if cached, ok := r.cached(tenantID); ok {
		return cached
	}

	provider, scope := r.build(ctx, tenantID)

	r.mu.Lock()
	if r.ttl > 0 {
		r.cache[tenantID] = cachedChain{provider: provider, expires: time.Now().Add(r.ttl), scope: scope}
	}
	r.mu.Unlock()

	return provider
}

func (r *EmailProviderResolver) cached(tenantID string) (NotificationProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.cache[tenantID]
	if !ok || time.Now().After(entry.expires) {
		return nil, false
	}
	return entry.provider, true
}

// build does the actual resolution without touching the cache.
func (r *EmailProviderResolver) build(ctx context.Context, tenantID string) (NotificationProvider, string) {
	if r.repo != nil && tenantID != "" {
		if chain := r.chainFor(ctx, tenantID); chain != nil {
			return chain, "tenant"
		}
	}
	if r.repo != nil {
		if chain := r.chainFor(ctx, models.PlatformScope); chain != nil {
			return chain, "platform"
		}
	}
	// A database problem must not stop mail going out; the environment chain
	// is the last line.
	return r.fallback, "environment"
}

// chainFor loads one scope's enabled configs and turns them into a chain.
func (r *EmailProviderResolver) chainFor(ctx context.Context, scope string) NotificationProvider {
	configs, err := r.repo.ListEnabled(ctx, scope)
	if err != nil {
		r.logger.WithError(err).WithField("scope", scope).
			Warn("Failed to load email provider configs; falling through")
		return nil
	}
	if len(configs) == 0 {
		return nil
	}

	providers := make([]NotificationProvider, 0, len(configs))
	for i := range configs {
		cfg := configs[i]
		provider, err := r.providerFrom(&cfg)
		if err != nil {
			r.logger.WithError(err).WithFields(logrus.Fields{
				"scope":    scope,
				"provider": cfg.Provider,
			}).Warn("Skipping misconfigured email provider")
			continue
		}
		providers = append(providers, provider)
	}
	if len(providers) == 0 {
		return nil
	}
	return NewFailoverProvider(models.ChannelEmail, r.logger, providers...)
}

// providerFrom turns a stored config into a live provider, decrypting the
// credential at the last possible moment.
func (r *EmailProviderResolver) providerFrom(cfg *models.EmailProviderConfig) (NotificationProvider, error) {
	secret, err := r.repo.DecryptSecret(cfg.Secret)
	if err != nil {
		return nil, err
	}
	return BuildProviderFromConfig(cfg, secret, r.logger)
}

package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ErrProviderConfigNotFound is returned when no config exists for a
// (tenant, provider) pair.
var ErrProviderConfigNotFound = errors.New("email provider config not found")

// EmailProviderRepository stores the per-tenant and platform-wide email
// provider chains. Every method takes a tenantID and scopes on it, so one
// tenant can never read or overwrite another's credentials; the platform
// default lives under models.PlatformScope.
type EmailProviderRepository interface {
	// List returns every config for a scope, ordered by priority.
	List(ctx context.Context, tenantID string) ([]models.EmailProviderConfig, error)
	// ListEnabled returns only the enabled configs for a scope, ordered by
	// priority — this is what the resolver builds a chain from.
	ListEnabled(ctx context.Context, tenantID string) ([]models.EmailProviderConfig, error)
	// Get fetches one config by provider key within a scope.
	Get(ctx context.Context, tenantID, provider string) (*models.EmailProviderConfig, error)
	// Upsert creates or updates the config for a (tenant, provider) pair.
	Upsert(ctx context.Context, cfg *models.EmailProviderConfig) error
	// Delete removes one config from a scope.
	Delete(ctx context.Context, tenantID, provider string) error

	// Secrets are sealed and opened here so the plaintext never lives in a
	// struct that another layer might marshal.
	EncryptSecret(plaintext string) (string, error)
	DecryptSecret(stored string) (string, error)
	// UsingEncryption reports whether secrets are genuinely encrypted rather
	// than base64-encoded, so startup can warn.
	UsingEncryption() bool
}

type emailProviderRepository struct {
	col    *mongo.Collection
	secret *SecretBox
}

// NewEmailProviderRepository wires the repository. encryptionKey may be empty
// for local development, in which case secrets are only base64-encoded.
func NewEmailProviderRepository(db *mongo.Database, encryptionKey []byte) (EmailProviderRepository, error) {
	box, err := NewSecretBox(encryptionKey)
	if err != nil {
		return nil, err
	}
	r := &emailProviderRepository{
		col:    db.Collection("email_provider_configs"),
		secret: box,
	}
	r.ensureIndexes(context.Background())
	return r, nil
}

// ensureIndexes enforces one config per (tenant, provider) so an upsert cannot
// silently create a duplicate chain entry. Failure is non-fatal: an existing
// deployment with a conflicting row should not stop the service booting.
func (r *emailProviderRepository) ensureIndexes(ctx context.Context) {
	_, _ = r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "tenant_id", Value: 1}, {Key: "provider", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_tenant_provider"),
	})
}

func (r *emailProviderRepository) List(ctx context.Context, tenantID string) ([]models.EmailProviderConfig, error) {
	return r.find(ctx, bson.M{"tenant_id": tenantID})
}

func (r *emailProviderRepository) ListEnabled(ctx context.Context, tenantID string) ([]models.EmailProviderConfig, error) {
	return r.find(ctx, bson.M{"tenant_id": tenantID, "enabled": true})
}

func (r *emailProviderRepository) find(ctx context.Context, filter bson.M) ([]models.EmailProviderConfig, error) {
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: 1}, {Key: "provider", Value: 1}})
	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	configs := make([]models.EmailProviderConfig, 0)
	if err := cursor.All(ctx, &configs); err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *emailProviderRepository) Get(ctx context.Context, tenantID, provider string) (*models.EmailProviderConfig, error) {
	var cfg models.EmailProviderConfig
	err := r.col.FindOne(ctx, bson.M{"tenant_id": tenantID, "provider": provider}).Decode(&cfg)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrProviderConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *emailProviderRepository) Upsert(ctx context.Context, cfg *models.EmailProviderConfig) error {
	if cfg.TenantID == "" || cfg.Provider == "" {
		return errors.New("tenant_id and provider are required")
	}
	now := time.Now().UTC()
	cfg.UpdatedAt = now

	set := bson.M{
		"provider":   cfg.Provider,
		"tenant_id":  cfg.TenantID,
		"enabled":    cfg.Enabled,
		"priority":   cfg.Priority,
		"host":       cfg.Host,
		"port":       cfg.Port,
		"username":   cfg.Username,
		"from_email": cfg.FromEmail,
		"from_name":  cfg.FromName,
		"updated_at": now,
	}
	// An empty secret means "leave the stored credential alone", so the field
	// is only written when a new one was supplied. This is what lets the UI
	// edit a port or toggle Enabled without re-entering the key.
	if cfg.Secret != "" {
		set["secret_enc"] = cfg.Secret
	}

	update := bson.M{
		"$set": set,
		"$setOnInsert": bson.M{
			"_id":        firstNonEmptyID(cfg.ID),
			"created_at": now,
		},
	}

	_, err := r.col.UpdateOne(ctx,
		bson.M{"tenant_id": cfg.TenantID, "provider": cfg.Provider},
		update,
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *emailProviderRepository) Delete(ctx context.Context, tenantID, provider string) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"tenant_id": tenantID, "provider": provider})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrProviderConfigNotFound
	}
	return nil
}

func (r *emailProviderRepository) EncryptSecret(plaintext string) (string, error) {
	return r.secret.Encrypt(plaintext)
}

func (r *emailProviderRepository) DecryptSecret(stored string) (string, error) {
	return r.secret.Decrypt(stored)
}

func (r *emailProviderRepository) UsingEncryption() bool {
	return !r.secret.NoEncryption()
}

func firstNonEmptyID(id string) string {
	if id != "" {
		return id
	}
	return uuid.New().String()
}

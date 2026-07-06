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

type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByID(ctx context.Context, tenantID, id string) (*models.Notification, error)
	GetByUserID(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.Notification, int64, error)
	Update(ctx context.Context, notification *models.Notification) error
	MarkAsRead(ctx context.Context, tenantID, id string) error

	// User preferences
	GetPreference(ctx context.Context, tenantID, userID string) (*models.UserPreference, error)
	UpsertPreference(ctx context.Context, pref *models.UserPreference) error

	// Templates
	ListTemplates(ctx context.Context, tenantID string) ([]models.NotificationTemplate, error)
	GetTemplate(ctx context.Context, tenantID, id string) (*models.NotificationTemplate, error)
	GetTemplateByType(ctx context.Context, tenantID, channel, notifType string) (*models.NotificationTemplate, error)
	CreateTemplate(ctx context.Context, t *models.NotificationTemplate) error
	UpdateTemplate(ctx context.Context, tenantID, id string, t *models.NotificationTemplate) error
	DeleteTemplate(ctx context.Context, tenantID, id string) error
}

type notificationRepository struct {
	notifications *mongo.Collection
	preferences   *mongo.Collection
	templates     *mongo.Collection
}

func NewNotificationRepository(db *mongo.Database) NotificationRepository {
	return &notificationRepository{
		notifications: db.Collection("notifications"),
		preferences:   db.Collection("user_preferences"),
		templates:     db.Collection("notification_templates"),
	}
}

func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	notification.CreatedAt = now
	notification.UpdatedAt = now

	_, err := r.notifications.InsertOne(ctx, notification)
	return err
}

func (r *notificationRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Notification, error) {
	var notification models.Notification
	err := r.notifications.FindOne(ctx, bson.M{"_id": id, "tenant_id": tenantID}).Decode(&notification)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("notification not found")
		}
		return nil, err
	}
	return &notification, nil
}

func (r *notificationRepository) GetByUserID(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.Notification, int64, error) {
	filter := bson.M{
		"tenant_id": tenantID,
		"user_id":   userID,
	}

	total, err := r.notifications.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	offset := int64((page - 1) * pageSize)
	opts := options.Find().
		SetSkip(offset).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.notifications.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *notificationRepository) Update(ctx context.Context, notification *models.Notification) error {
	notification.UpdatedAt = time.Now().UTC()
	_, err := r.notifications.ReplaceOne(ctx, bson.M{"_id": notification.ID}, notification)
	return err
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, tenantID, id string) error {
	now := time.Now().UTC()
	res, err := r.notifications.UpdateOne(
		ctx,
		bson.M{"_id": id, "tenant_id": tenantID},
		bson.M{
			"$set": bson.M{
				"status":     models.StatusRead,
				"read_at":    now,
				"updated_at": now,
			},
		},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("notification not found")
	}
	return nil
}

func (r *notificationRepository) GetPreference(ctx context.Context, tenantID, userID string) (*models.UserPreference, error) {
	var pref models.UserPreference
	err := r.preferences.FindOne(ctx, bson.M{
		"tenant_id": tenantID,
		"user_id":   userID,
	}).Decode(&pref)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &pref, nil
}

func (r *notificationRepository) UpsertPreference(ctx context.Context, pref *models.UserPreference) error {
	now := time.Now().UTC()
	pref.UpdatedAt = now

	filter := bson.M{
		"tenant_id": pref.TenantID,
		"user_id":   pref.UserID,
	}

	update := bson.M{
		"$set": pref,
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.preferences.UpdateOne(ctx, filter, update, opts)
	return err
}

// === Templates ===

func (r *notificationRepository) ListTemplates(ctx context.Context, tenantID string) ([]models.NotificationTemplate, error) {
	filter := bson.M{"tenant_id": tenantID}
	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})

	cursor, err := r.templates.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []models.NotificationTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, err
	}
	if templates == nil {
		templates = []models.NotificationTemplate{}
	}
	return templates, nil
}

func (r *notificationRepository) GetTemplate(ctx context.Context, tenantID, id string) (*models.NotificationTemplate, error) {
	var t models.NotificationTemplate
	err := r.templates.FindOne(ctx, bson.M{"_id": id, "tenant_id": tenantID}).Decode(&t)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("template not found")
		}
		return nil, err
	}
	return &t, nil
}

// GetTemplateByType returns the active template for a tenant+channel+type tuple.
// Returns nil (no error) when no template is configured — callers must treat
// that as "fall back to the hardcoded renderer".
func (r *notificationRepository) GetTemplateByType(ctx context.Context, tenantID, channel, notifType string) (*models.NotificationTemplate, error) {
	var t models.NotificationTemplate
	err := r.templates.FindOne(ctx, bson.M{
		"tenant_id": tenantID,
		"channel":   channel,
		"type":      notifType,
		"is_active": true,
	}).Decode(&t)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *notificationRepository) CreateTemplate(ctx context.Context, t *models.NotificationTemplate) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	_, err := r.templates.InsertOne(ctx, t)
	return err
}

func (r *notificationRepository) UpdateTemplate(ctx context.Context, tenantID, id string, t *models.NotificationTemplate) error {
	t.UpdatedAt = time.Now().UTC()
	set := bson.M{
		"updated_at": t.UpdatedAt,
	}
	if t.Type != "" {
		set["type"] = t.Type
	}
	if t.Channel != "" {
		set["channel"] = t.Channel
	}
	if t.Name != "" {
		set["name"] = t.Name
	}
	// Subject can legitimately be empty (push/sms with no subject); always set.
	set["subject_template"] = t.SubjectTemplate
	if t.BodyTemplate != "" {
		set["body_template"] = t.BodyTemplate
	}
	set["is_active"] = t.IsActive

	res, err := r.templates.UpdateOne(ctx, bson.M{"_id": id, "tenant_id": tenantID}, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("template not found")
	}
	return nil
}

func (r *notificationRepository) DeleteTemplate(ctx context.Context, tenantID, id string) error {
	res, err := r.templates.DeleteOne(ctx, bson.M{"_id": id, "tenant_id": tenantID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("template not found")
	}
	return nil
}

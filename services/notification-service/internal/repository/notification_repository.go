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
	GetByID(ctx context.Context, id string) (*models.Notification, error)
	GetByUserID(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.Notification, int64, error)
	Update(ctx context.Context, notification *models.Notification) error
	MarkAsRead(ctx context.Context, id string) error

	// User preferences
	GetPreference(ctx context.Context, tenantID, userID string) (*models.UserPreference, error)
	UpsertPreference(ctx context.Context, pref *models.UserPreference) error
}

type notificationRepository struct {
	notifications *mongo.Collection
	preferences   *mongo.Collection
}

func NewNotificationRepository(db *mongo.Database) NotificationRepository {
	return &notificationRepository{
		notifications: db.Collection("notifications"),
		preferences:   db.Collection("user_preferences"),
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

func (r *notificationRepository) GetByID(ctx context.Context, id string) (*models.Notification, error) {
	var notification models.Notification
	err := r.notifications.FindOne(ctx, bson.M{"_id": id}).Decode(&notification)
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

func (r *notificationRepository) MarkAsRead(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.notifications.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"status":     models.StatusRead,
				"read_at":    now,
				"updated_at": now,
			},
		},
	)
	return err
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

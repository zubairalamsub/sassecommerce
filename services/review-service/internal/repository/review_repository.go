package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ecommerce/review-service/internal/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ReviewRepository interface {
	Create(ctx context.Context, review *models.Review) error
	GetByID(ctx context.Context, id string) (*models.Review, error)
	GetByProductID(ctx context.Context, tenantID, productID string, page, pageSize int) ([]models.Review, int64, error)
	GetByUserID(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.Review, int64, error)
	Update(ctx context.Context, review *models.Review) error
	Delete(ctx context.Context, id string) error
	GetProductSummary(ctx context.Context, tenantID, productID string) (*models.ReviewSummary, error)
	HasUserReviewed(ctx context.Context, tenantID, productID, userID string) (bool, error)
	AddHelpfulVote(ctx context.Context, id, voterID string, helpful bool) error
}

type reviewRepository struct {
	reviews *mongo.Collection
}

func NewReviewRepository(db *mongo.Database) ReviewRepository {
	return &reviewRepository{
		reviews: db.Collection("reviews"),
	}
}

func (r *reviewRepository) Create(ctx context.Context, review *models.Review) error {
	if review.ID == "" {
		review.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	review.CreatedAt = now
	review.UpdatedAt = now

	_, err := r.reviews.InsertOne(ctx, review)
	return err
}

func (r *reviewRepository) GetByID(ctx context.Context, id string) (*models.Review, error) {
	var review models.Review
	err := r.reviews.FindOne(ctx, bson.M{
		"_id":        id,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&review)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("review not found")
		}
		return nil, err
	}
	return &review, nil
}

func (r *reviewRepository) GetByProductID(ctx context.Context, tenantID, productID string, page, pageSize int) ([]models.Review, int64, error) {
	filter := bson.M{
		"tenant_id":  tenantID,
		"product_id": productID,
		"status":     models.StatusApproved,
		"deleted_at": bson.M{"$exists": false},
	}

	total, err := r.reviews.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	offset := int64((page - 1) * pageSize)
	opts := options.Find().
		SetSkip(offset).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.reviews.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var reviews []models.Review
	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *reviewRepository) GetByUserID(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.Review, int64, error) {
	filter := bson.M{
		"tenant_id":  tenantID,
		"user_id":    userID,
		"deleted_at": bson.M{"$exists": false},
	}

	total, err := r.reviews.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	offset := int64((page - 1) * pageSize)
	opts := options.Find().
		SetSkip(offset).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.reviews.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var reviews []models.Review
	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *reviewRepository) Update(ctx context.Context, review *models.Review) error {
	review.UpdatedAt = time.Now().UTC()
	_, err := r.reviews.ReplaceOne(ctx, bson.M{"_id": review.ID}, review)
	return err
}

func (r *reviewRepository) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.reviews.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"deleted_at": now, "updated_at": now}},
	)
	return err
}

func (r *reviewRepository) GetProductSummary(ctx context.Context, tenantID, productID string) (*models.ReviewSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"tenant_id":  tenantID,
			"product_id": productID,
			"status":     models.StatusApproved,
			"deleted_at": bson.M{"$exists": false},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":            "$product_id",
			"average_rating": bson.M{"$avg": "$rating"},
			"total_reviews":  bson.M{"$sum": 1},
			"ratings":        bson.M{"$push": "$rating"},
		}}},
	}

	cursor, err := r.reviews.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		ProductID     string    `bson:"_id"`
		AverageRating float64   `bson:"average_rating"`
		TotalReviews  int       `bson:"total_reviews"`
		Ratings       []int     `bson:"ratings"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &models.ReviewSummary{
			ProductID:     productID,
			TenantID:      tenantID,
			AverageRating: 0,
			TotalReviews:  0,
			Distribution:  map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0},
		}, nil
	}

	result := results[0]
	distribution := map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0}
	for _, r := range result.Ratings {
		key := string(rune('0' + r))
		distribution[key]++
	}

	// Round average to 1 decimal
	avgRounded := float64(int(result.AverageRating*10)) / 10

	return &models.ReviewSummary{
		ProductID:     productID,
		TenantID:      tenantID,
		AverageRating: avgRounded,
		TotalReviews:  result.TotalReviews,
		Distribution:  distribution,
	}, nil
}

func (r *reviewRepository) HasUserReviewed(ctx context.Context, tenantID, productID, userID string) (bool, error) {
	count, err := r.reviews.CountDocuments(ctx, bson.M{
		"tenant_id":  tenantID,
		"product_id": productID,
		"user_id":    userID,
		"deleted_at": bson.M{"$exists": false},
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *reviewRepository) AddHelpfulVote(ctx context.Context, id, voterID string, helpful bool) error {
	field := "helpful_count"
	if !helpful {
		field = "unhelpful_count"
	}

	_, err := r.reviews.UpdateOne(
		ctx,
		bson.M{
			"_id":              id,
			"helpful_voters":   bson.M{"$ne": voterID},
		},
		bson.M{
			"$inc":      bson.M{field: 1},
			"$addToSet": bson.M{"helpful_voters": voterID},
			"$set":      bson.M{"updated_at": time.Now().UTC()},
		},
	)
	return err
}

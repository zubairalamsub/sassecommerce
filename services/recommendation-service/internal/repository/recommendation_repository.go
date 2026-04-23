package repository

import (
	"context"

	"github.com/ecommerce/recommendation-service/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RecommendationRepository interface {
	// User interactions
	RecordInteraction(ctx context.Context, interaction *models.UserInteraction) error
	GetUserInteractions(ctx context.Context, tenantID, userID string, limit int) ([]models.UserInteraction, error)
	GetProductInteractors(ctx context.Context, tenantID, productID string) ([]string, error)

	// Product similarity
	UpsertSimilarity(ctx context.Context, similarity *models.ProductSimilarity) error
	GetSimilarProducts(ctx context.Context, tenantID, productID string, limit int) ([]models.ProductSimilarity, error)
	DeleteSimilaritiesByTenant(ctx context.Context, tenantID string) error

	// User-based recommendations (collaborative filtering)
	GetUserRecommendations(ctx context.Context, tenantID, userID string, limit int) ([]models.ProductRecommendation, error)

	// Co-purchase data for training
	GetCoPurchasePairs(ctx context.Context, tenantID string) ([]CoPurchasePair, error)
	GetDistinctProducts(ctx context.Context, tenantID string) ([]string, error)

	// Training jobs
	CreateTrainingJob(ctx context.Context, job *models.TrainingJob) error
	UpdateTrainingJob(ctx context.Context, job *models.TrainingJob) error
	GetTrainingJob(ctx context.Context, id string) (*models.TrainingJob, error)
}

type CoPurchasePair struct {
	ProductA string
	ProductB string
	Count    int
}

type recommendationRepository struct {
	db *gorm.DB
}

func NewRecommendationRepository(db *gorm.DB) RecommendationRepository {
	return &recommendationRepository{db: db}
}

func (r *recommendationRepository) RecordInteraction(ctx context.Context, interaction *models.UserInteraction) error {
	return r.db.WithContext(ctx).Create(interaction).Error
}

func (r *recommendationRepository) GetUserInteractions(ctx context.Context, tenantID, userID string, limit int) ([]models.UserInteraction, error) {
	var interactions []models.UserInteraction
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&interactions).Error
	return interactions, err
}

func (r *recommendationRepository) GetProductInteractors(ctx context.Context, tenantID, productID string) ([]string, error) {
	var userIDs []string
	err := r.db.WithContext(ctx).Model(&models.UserInteraction{}).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Distinct("user_id").
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *recommendationRepository) UpsertSimilarity(ctx context.Context, similarity *models.ProductSimilarity) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"score", "reason", "updated_at"}),
		}).
		Create(similarity).Error
}

func (r *recommendationRepository) GetSimilarProducts(ctx context.Context, tenantID, productID string, limit int) ([]models.ProductSimilarity, error) {
	var similarities []models.ProductSimilarity
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Order("score DESC").
		Limit(limit).
		Find(&similarities).Error
	return similarities, err
}

func (r *recommendationRepository) DeleteSimilaritiesByTenant(ctx context.Context, tenantID string) error {
	return r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Delete(&models.ProductSimilarity{}).Error
}

func (r *recommendationRepository) GetUserRecommendations(ctx context.Context, tenantID, userID string, limit int) ([]models.ProductRecommendation, error) {
	// Collaborative filtering: find products purchased by users who purchased similar products
	// 1. Get products this user interacted with
	// 2. Find other users who interacted with same products
	// 3. Get products those users interacted with that this user hasn't seen
	var recommendations []models.ProductRecommendation

	subQuery := r.db.WithContext(ctx).Model(&models.UserInteraction{}).
		Select("product_id").
		Where("tenant_id = ? AND user_id = ?", tenantID, userID)

	err := r.db.WithContext(ctx).Model(&models.UserInteraction{}).
		Select("product_id, SUM(score) as score").
		Where("tenant_id = ? AND user_id IN (?) AND product_id NOT IN (?)",
			tenantID,
			r.db.Model(&models.UserInteraction{}).Select("DISTINCT user_id").
				Where("tenant_id = ? AND product_id IN (?)", tenantID, subQuery),
			subQuery,
		).
		Group("product_id").
		Order("score DESC").
		Limit(limit).
		Find(&recommendations).Error

	if err != nil {
		return nil, err
	}

	for i := range recommendations {
		recommendations[i].Reason = "collaborative_filtering"
	}

	return recommendations, nil
}

func (r *recommendationRepository) GetCoPurchasePairs(ctx context.Context, tenantID string) ([]CoPurchasePair, error) {
	var pairs []CoPurchasePair

	err := r.db.WithContext(ctx).Raw(`
		SELECT a.product_id as product_a, b.product_id as product_b, COUNT(*) as count
		FROM user_interactions a
		JOIN user_interactions b ON a.user_id = b.user_id AND a.tenant_id = b.tenant_id
		WHERE a.tenant_id = ? AND a.type = 'purchase' AND b.type = 'purchase'
		AND a.product_id < b.product_id
		GROUP BY a.product_id, b.product_id
		HAVING COUNT(*) >= 2
		ORDER BY count DESC
	`, tenantID).Scan(&pairs).Error

	return pairs, err
}

func (r *recommendationRepository) GetDistinctProducts(ctx context.Context, tenantID string) ([]string, error) {
	var productIDs []string
	err := r.db.WithContext(ctx).Model(&models.UserInteraction{}).
		Where("tenant_id = ?", tenantID).
		Distinct("product_id").
		Pluck("product_id", &productIDs).Error
	return productIDs, err
}

func (r *recommendationRepository) CreateTrainingJob(ctx context.Context, job *models.TrainingJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *recommendationRepository) UpdateTrainingJob(ctx context.Context, job *models.TrainingJob) error {
	return r.db.WithContext(ctx).Save(job).Error
}

func (r *recommendationRepository) GetTrainingJob(ctx context.Context, id string) (*models.TrainingJob, error) {
	var job models.TrainingJob
	if err := r.db.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

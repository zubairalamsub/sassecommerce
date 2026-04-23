package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ecommerce/recommendation-service/internal/models"
	"github.com/ecommerce/recommendation-service/internal/repository"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type RecommendationService interface {
	// Recommendations
	GetUserRecommendations(ctx context.Context, tenantID, userID string, limit int) (*models.RecommendationResponse, error)
	GetProductRecommendations(ctx context.Context, tenantID, productID string, limit int) (*models.RecommendationResponse, error)

	// Training
	TrainModel(ctx context.Context, tenantID string) (*models.TrainingJobResponse, error)
	GetTrainingJob(ctx context.Context, id string) (*models.TrainingJobResponse, error)

	// Event ingestion
	RecordInteraction(ctx context.Context, tenantID, userID, productID, interactionType string) error
}

type recommendationService struct {
	repo   repository.RecommendationRepository
	writer *kafka.Writer
	logger *logrus.Logger
}

func NewRecommendationService(repo repository.RecommendationRepository, writer *kafka.Writer, logger *logrus.Logger) RecommendationService {
	return &recommendationService{
		repo:   repo,
		writer: writer,
		logger: logger,
	}
}

func (s *recommendationService) GetUserRecommendations(ctx context.Context, tenantID, userID string, limit int) (*models.RecommendationResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Try collaborative filtering first
	recommendations, err := s.repo.GetUserRecommendations(ctx, tenantID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get user recommendations: %w", err)
	}

	strategy := "collaborative_filtering"

	// If not enough collaborative recommendations, fall back to content-based
	if len(recommendations) < limit {
		contentRecs, err := s.getContentBasedRecommendations(ctx, tenantID, userID, limit-len(recommendations))
		if err == nil && len(contentRecs) > 0 {
			recommendations = append(recommendations, contentRecs...)
			if strategy == "collaborative_filtering" && len(contentRecs) > 0 {
				strategy = "hybrid"
			}
		}
	}

	// If still empty, return popular items
	if len(recommendations) == 0 {
		strategy = "popular"
		popularRecs, err := s.getPopularProducts(ctx, tenantID, limit)
		if err == nil {
			recommendations = popularRecs
		}
	}

	return &models.RecommendationResponse{
		UserID:          userID,
		Recommendations: recommendations,
		Strategy:        strategy,
		GeneratedAt:     time.Now().UTC(),
	}, nil
}

func (s *recommendationService) GetProductRecommendations(ctx context.Context, tenantID, productID string, limit int) (*models.RecommendationResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	similarities, err := s.repo.GetSimilarProducts(ctx, tenantID, productID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get product recommendations: %w", err)
	}

	recommendations := make([]models.ProductRecommendation, len(similarities))
	for i, sim := range similarities {
		recommendations[i] = models.ProductRecommendation{
			ProductID: sim.SimilarID,
			Score:     sim.Score,
			Reason:    sim.Reason,
		}
	}

	strategy := "content_based"
	if len(recommendations) == 0 {
		strategy = "popular"
		popularRecs, err := s.getPopularProducts(ctx, tenantID, limit)
		if err == nil {
			recommendations = popularRecs
		}
	}

	return &models.RecommendationResponse{
		ProductID:       productID,
		Recommendations: recommendations,
		Strategy:        strategy,
		GeneratedAt:     time.Now().UTC(),
	}, nil
}

func (s *recommendationService) TrainModel(ctx context.Context, tenantID string) (*models.TrainingJobResponse, error) {
	job := &models.TrainingJob{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Status:   "running",
	}

	if err := s.repo.CreateTrainingJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create training job: %w", err)
	}

	// Run training inline
	s.runTraining(ctx, job)

	return toTrainingJobResponse(job), nil
}

func (s *recommendationService) runTraining(ctx context.Context, job *models.TrainingJob) {
	// Clear existing similarities for this tenant
	if err := s.repo.DeleteSimilaritiesByTenant(ctx, job.TenantID); err != nil {
		s.logger.WithError(err).Error("Failed to clear old similarities")
		job.Status = "failed"
		job.Error = err.Error()
		s.repo.UpdateTrainingJob(ctx, job)
		return
	}

	// Get co-purchase pairs
	pairs, err := s.repo.GetCoPurchasePairs(ctx, job.TenantID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get co-purchase pairs")
		job.Status = "failed"
		job.Error = err.Error()
		s.repo.UpdateTrainingJob(ctx, job)
		return
	}

	// Get distinct products count
	products, err := s.repo.GetDistinctProducts(ctx, job.TenantID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get distinct products")
		job.Status = "failed"
		job.Error = err.Error()
		s.repo.UpdateTrainingJob(ctx, job)
		return
	}

	job.ItemsProcessed = len(products)
	similCount := 0

	// Generate similarities from co-purchase data
	for _, pair := range pairs {
		score := normalizeScore(pair.Count)

		// Create bidirectional similarity
		sim1 := &models.ProductSimilarity{
			ID:        uuid.New().String(),
			TenantID:  job.TenantID,
			ProductID: pair.ProductA,
			SimilarID: pair.ProductB,
			Score:     score,
			Reason:    "co_purchase",
		}
		sim2 := &models.ProductSimilarity{
			ID:        uuid.New().String(),
			TenantID:  job.TenantID,
			ProductID: pair.ProductB,
			SimilarID: pair.ProductA,
			Score:     score,
			Reason:    "co_purchase",
		}

		if err := s.repo.UpsertSimilarity(ctx, sim1); err != nil {
			s.logger.WithError(err).Warn("Failed to upsert similarity")
			continue
		}
		if err := s.repo.UpsertSimilarity(ctx, sim2); err != nil {
			s.logger.WithError(err).Warn("Failed to upsert similarity")
			continue
		}
		similCount += 2
	}

	job.SimilaritiesGenerated = similCount
	job.Status = "completed"
	now := time.Now().UTC()
	job.CompletedAt = &now

	s.repo.UpdateTrainingJob(ctx, job)

	s.logger.WithFields(logrus.Fields{
		"tenant_id":     job.TenantID,
		"products":      job.ItemsProcessed,
		"similarities":  job.SimilaritiesGenerated,
	}).Info("Training completed")
}

func (s *recommendationService) GetTrainingJob(ctx context.Context, id string) (*models.TrainingJobResponse, error) {
	job, err := s.repo.GetTrainingJob(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("training job not found")
	}
	return toTrainingJobResponse(job), nil
}

func (s *recommendationService) RecordInteraction(ctx context.Context, tenantID, userID, productID, interactionType string) error {
	weight, ok := models.InteractionWeights[interactionType]
	if !ok {
		weight = 1.0
	}

	interaction := &models.UserInteraction{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		UserID:    userID,
		ProductID: productID,
		Type:      interactionType,
		Score:     weight,
	}

	if err := s.repo.RecordInteraction(ctx, interaction); err != nil {
		return fmt.Errorf("failed to record interaction: %w", err)
	}

	return nil
}

// === Helper methods ===

func (s *recommendationService) getContentBasedRecommendations(ctx context.Context, tenantID, userID string, limit int) ([]models.ProductRecommendation, error) {
	// Get user's recent interactions
	interactions, err := s.repo.GetUserInteractions(ctx, tenantID, userID, 20)
	if err != nil || len(interactions) == 0 {
		return nil, err
	}

	// Collect similar products for items the user interacted with
	seen := make(map[string]bool)
	var recommendations []models.ProductRecommendation

	for _, interaction := range interactions {
		seen[interaction.ProductID] = true
	}

	for _, interaction := range interactions {
		similarities, err := s.repo.GetSimilarProducts(ctx, tenantID, interaction.ProductID, 5)
		if err != nil {
			continue
		}
		for _, sim := range similarities {
			if seen[sim.SimilarID] {
				continue
			}
			seen[sim.SimilarID] = true
			recommendations = append(recommendations, models.ProductRecommendation{
				ProductID: sim.SimilarID,
				Score:     sim.Score * interaction.Score,
				Reason:    "content_based",
			})
			if len(recommendations) >= limit {
				return recommendations, nil
			}
		}
	}

	return recommendations, nil
}

func (s *recommendationService) getPopularProducts(ctx context.Context, tenantID string, limit int) ([]models.ProductRecommendation, error) {
	// Get most interacted products as a fallback
	var results []struct {
		ProductID string
		Score     float64
	}

	interactions, err := s.repo.GetUserInteractions(ctx, tenantID, "", limit*2)
	if err != nil {
		return nil, err
	}

	scoreMap := make(map[string]float64)
	for _, i := range interactions {
		scoreMap[i.ProductID] += i.Score
	}

	for pid, score := range scoreMap {
		results = append(results, struct {
			ProductID string
			Score     float64
		}{pid, score})
	}

	recommendations := make([]models.ProductRecommendation, 0, limit)
	for i, r := range results {
		if i >= limit {
			break
		}
		recommendations = append(recommendations, models.ProductRecommendation{
			ProductID: r.ProductID,
			Score:     r.Score,
			Reason:    "popular",
		})
	}

	return recommendations, nil
}

func normalizeScore(count int) float64 {
	// Normalize co-purchase count to a 0-1 score
	// Using a simple log-based normalization
	if count <= 0 {
		return 0
	}
	score := float64(count) / 10.0
	if score > 1.0 {
		score = 1.0
	}
	return score
}

func toTrainingJobResponse(job *models.TrainingJob) *models.TrainingJobResponse {
	return &models.TrainingJobResponse{
		ID:                    job.ID,
		TenantID:              job.TenantID,
		Status:                job.Status,
		ItemsProcessed:        job.ItemsProcessed,
		SimilaritiesGenerated: job.SimilaritiesGenerated,
		StartedAt:             job.StartedAt,
		CompletedAt:           job.CompletedAt,
		Error:                 job.Error,
	}
}

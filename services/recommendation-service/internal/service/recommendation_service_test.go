package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ecommerce/recommendation-service/internal/models"
	"github.com/ecommerce/recommendation-service/internal/repository"
	repoMocks "github.com/ecommerce/recommendation-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (*recommendationService, *repoMocks.MockRecommendationRepository) {
	mockRepo := new(repoMocks.MockRecommendationRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &recommendationService{
		repo:   mockRepo,
		writer: nil,
		logger: logger,
	}

	return svc, mockRepo
}

// === GetUserRecommendations Tests ===

func TestGetUserRecommendations_CollaborativeFiltering(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	// Collaborative returns enough to fill limit
	recs := make([]models.ProductRecommendation, 10)
	for i := 0; i < 10; i++ {
		recs[i] = models.ProductRecommendation{ProductID: fmt.Sprintf("p-%d", i), Score: float64(10 - i), Reason: "collaborative_filtering"}
	}
	mockRepo.On("GetUserRecommendations", ctx, "tenant-1", "user-1", 10).Return(recs, nil)

	result, err := svc.GetUserRecommendations(ctx, "tenant-1", "user-1", 10)

	assert.NoError(t, err)
	assert.Equal(t, "user-1", result.UserID)
	assert.Equal(t, "collaborative_filtering", result.Strategy)
	assert.Len(t, result.Recommendations, 10)
}

func TestGetUserRecommendations_FallbackToContentBased(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	// Collaborative returns partial results
	collabRecs := []models.ProductRecommendation{
		{ProductID: "p-1", Score: 10, Reason: "collaborative_filtering"},
	}
	mockRepo.On("GetUserRecommendations", ctx, "tenant-1", "user-1", 5).Return(collabRecs, nil)

	// Content-based path: get user interactions then similar products
	interactions := []models.UserInteraction{
		{ProductID: "p-5", Score: 5.0},
	}
	mockRepo.On("GetUserInteractions", ctx, "tenant-1", "user-1", 20).Return(interactions, nil)

	similarities := []models.ProductSimilarity{
		{SimilarID: "p-6", Score: 0.8, Reason: "co_purchase"},
	}
	mockRepo.On("GetSimilarProducts", ctx, "tenant-1", "p-5", 5).Return(similarities, nil)

	result, err := svc.GetUserRecommendations(ctx, "tenant-1", "user-1", 5)

	assert.NoError(t, err)
	assert.Equal(t, "hybrid", result.Strategy)
	assert.Len(t, result.Recommendations, 2)
}

func TestGetUserRecommendations_FallbackToPopular(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	// Collaborative returns empty
	mockRepo.On("GetUserRecommendations", ctx, "tenant-1", "user-new", 10).Return([]models.ProductRecommendation{}, nil)

	// Content-based: no interactions
	mockRepo.On("GetUserInteractions", ctx, "tenant-1", "user-new", 20).Return([]models.UserInteraction{}, nil)

	// Popular fallback
	popularInteractions := []models.UserInteraction{
		{ProductID: "p-popular", Score: 5.0},
	}
	mockRepo.On("GetUserInteractions", ctx, "tenant-1", "", 20).Return(popularInteractions, nil)

	result, err := svc.GetUserRecommendations(ctx, "tenant-1", "user-new", 10)

	assert.NoError(t, err)
	assert.Equal(t, "popular", result.Strategy)
}

func TestGetUserRecommendations_DefaultLimit(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetUserRecommendations", ctx, "tenant-1", "user-1", 10).Return([]models.ProductRecommendation{}, nil)
	mockRepo.On("GetUserInteractions", ctx, "tenant-1", "user-1", 20).Return([]models.UserInteraction{}, nil)
	mockRepo.On("GetUserInteractions", ctx, "tenant-1", "", 20).Return([]models.UserInteraction{}, nil)

	result, err := svc.GetUserRecommendations(ctx, "tenant-1", "user-1", 0)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserRecommendations_LimitCapped(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetUserRecommendations", ctx, "tenant-1", "user-1", 50).Return([]models.ProductRecommendation{}, nil)
	mockRepo.On("GetUserInteractions", ctx, "tenant-1", "user-1", 20).Return([]models.UserInteraction{}, nil)
	mockRepo.On("GetUserInteractions", ctx, "tenant-1", "", 100).Return([]models.UserInteraction{}, nil)

	result, err := svc.GetUserRecommendations(ctx, "tenant-1", "user-1", 100)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserRecommendations_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetUserRecommendations", ctx, "tenant-1", "user-1", 10).Return(nil, errors.New("db error"))

	result, err := svc.GetUserRecommendations(ctx, "tenant-1", "user-1", 10)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetProductRecommendations Tests ===

func TestGetProductRecommendations_WithSimilarities(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	similarities := []models.ProductSimilarity{
		{SimilarID: "p-2", Score: 0.9, Reason: "co_purchase"},
		{SimilarID: "p-3", Score: 0.7, Reason: "co_purchase"},
	}
	mockRepo.On("GetSimilarProducts", ctx, "tenant-1", "p-1", 10).Return(similarities, nil)

	result, err := svc.GetProductRecommendations(ctx, "tenant-1", "p-1", 10)

	assert.NoError(t, err)
	assert.Equal(t, "p-1", result.ProductID)
	assert.Equal(t, "content_based", result.Strategy)
	assert.Len(t, result.Recommendations, 2)
	assert.Equal(t, "p-2", result.Recommendations[0].ProductID)
}

func TestGetProductRecommendations_FallbackToPopular(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	// No similarities
	mockRepo.On("GetSimilarProducts", ctx, "tenant-1", "p-new", 10).Return([]models.ProductSimilarity{}, nil)

	// Popular fallback
	interactions := []models.UserInteraction{
		{ProductID: "p-popular", Score: 5.0},
	}
	mockRepo.On("GetUserInteractions", ctx, "tenant-1", "", 20).Return(interactions, nil)

	result, err := svc.GetProductRecommendations(ctx, "tenant-1", "p-new", 10)

	assert.NoError(t, err)
	assert.Equal(t, "popular", result.Strategy)
}

func TestGetProductRecommendations_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetSimilarProducts", ctx, "tenant-1", "p-1", 10).Return([]models.ProductSimilarity{}, errors.New("db error"))

	result, err := svc.GetProductRecommendations(ctx, "tenant-1", "p-1", 10)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === TrainModel Tests ===

func TestTrainModel_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreateTrainingJob", ctx, mock.AnythingOfType("*models.TrainingJob")).Return(nil)
	mockRepo.On("DeleteSimilaritiesByTenant", ctx, "tenant-1").Return(nil)

	pairs := []repository.CoPurchasePair{
		{ProductA: "p-1", ProductB: "p-2", Count: 5},
	}
	mockRepo.On("GetCoPurchasePairs", ctx, "tenant-1").Return(pairs, nil)
	mockRepo.On("GetDistinctProducts", ctx, "tenant-1").Return([]string{"p-1", "p-2", "p-3"}, nil)
	mockRepo.On("UpsertSimilarity", ctx, mock.AnythingOfType("*models.ProductSimilarity")).Return(nil)
	mockRepo.On("UpdateTrainingJob", ctx, mock.AnythingOfType("*models.TrainingJob")).Return(nil)

	result, err := svc.TrainModel(ctx, "tenant-1")

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, 3, result.ItemsProcessed)
	assert.Equal(t, 2, result.SimilaritiesGenerated) // 1 pair = 2 bidirectional
}

func TestTrainModel_NoPairs(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreateTrainingJob", ctx, mock.AnythingOfType("*models.TrainingJob")).Return(nil)
	mockRepo.On("DeleteSimilaritiesByTenant", ctx, "tenant-1").Return(nil)
	mockRepo.On("GetCoPurchasePairs", ctx, "tenant-1").Return([]repository.CoPurchasePair{}, nil)
	mockRepo.On("GetDistinctProducts", ctx, "tenant-1").Return([]string{}, nil)
	mockRepo.On("UpdateTrainingJob", ctx, mock.AnythingOfType("*models.TrainingJob")).Return(nil)

	result, err := svc.TrainModel(ctx, "tenant-1")

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, 0, result.SimilaritiesGenerated)
}

func TestTrainModel_CreateJobFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreateTrainingJob", ctx, mock.AnythingOfType("*models.TrainingJob")).Return(errors.New("db error"))

	result, err := svc.TrainModel(ctx, "tenant-1")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTrainModel_DeleteFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreateTrainingJob", ctx, mock.AnythingOfType("*models.TrainingJob")).Return(nil)
	mockRepo.On("DeleteSimilaritiesByTenant", ctx, "tenant-1").Return(errors.New("delete error"))
	mockRepo.On("UpdateTrainingJob", ctx, mock.AnythingOfType("*models.TrainingJob")).Return(nil)

	result, err := svc.TrainModel(ctx, "tenant-1")

	assert.NoError(t, err) // Job was created, training failed
	assert.Equal(t, "failed", result.Status)
}

// === GetTrainingJob Tests ===

func TestGetTrainingJob_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	job := &models.TrainingJob{
		ID:       "job-1",
		TenantID: "tenant-1",
		Status:   "completed",
		ItemsProcessed: 100,
		SimilaritiesGenerated: 50,
	}
	mockRepo.On("GetTrainingJob", ctx, "job-1").Return(job, nil)

	result, err := svc.GetTrainingJob(ctx, "job-1")

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, 100, result.ItemsProcessed)
}

func TestGetTrainingJob_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetTrainingJob", ctx, "bad").Return(nil, errors.New("record not found"))

	result, err := svc.GetTrainingJob(ctx, "bad")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// === RecordInteraction Tests ===

func TestRecordInteraction_Purchase(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordInteraction", ctx, mock.AnythingOfType("*models.UserInteraction")).Return(nil)

	err := svc.RecordInteraction(ctx, "tenant-1", "user-1", "product-1", "purchase")

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "RecordInteraction", ctx, mock.MatchedBy(func(i *models.UserInteraction) bool {
		return i.Score == 5.0 && i.Type == "purchase"
	}))
}

func TestRecordInteraction_View(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordInteraction", ctx, mock.AnythingOfType("*models.UserInteraction")).Return(nil)

	err := svc.RecordInteraction(ctx, "tenant-1", "user-1", "product-1", "view")

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "RecordInteraction", ctx, mock.MatchedBy(func(i *models.UserInteraction) bool {
		return i.Score == 1.0 && i.Type == "view"
	}))
}

func TestRecordInteraction_Cart(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordInteraction", ctx, mock.AnythingOfType("*models.UserInteraction")).Return(nil)

	err := svc.RecordInteraction(ctx, "tenant-1", "user-1", "product-1", "cart")

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "RecordInteraction", ctx, mock.MatchedBy(func(i *models.UserInteraction) bool {
		return i.Score == 3.0 && i.Type == "cart"
	}))
}

func TestRecordInteraction_UnknownType(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordInteraction", ctx, mock.AnythingOfType("*models.UserInteraction")).Return(nil)

	err := svc.RecordInteraction(ctx, "tenant-1", "user-1", "product-1", "unknown")

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "RecordInteraction", ctx, mock.MatchedBy(func(i *models.UserInteraction) bool {
		return i.Score == 1.0 // default weight
	}))
}

func TestRecordInteraction_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordInteraction", ctx, mock.AnythingOfType("*models.UserInteraction")).Return(errors.New("db error"))

	err := svc.RecordInteraction(ctx, "tenant-1", "user-1", "product-1", "view")

	assert.Error(t, err)
}

// === normalizeScore Tests ===

func TestNormalizeScore(t *testing.T) {
	assert.Equal(t, 0.0, normalizeScore(0))
	assert.Equal(t, 0.2, normalizeScore(2))
	assert.Equal(t, 0.5, normalizeScore(5))
	assert.Equal(t, 1.0, normalizeScore(10))
	assert.Equal(t, 1.0, normalizeScore(20)) // capped at 1.0
}

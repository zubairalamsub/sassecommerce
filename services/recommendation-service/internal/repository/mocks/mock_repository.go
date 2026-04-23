package mocks

import (
	"context"

	"github.com/ecommerce/recommendation-service/internal/models"
	"github.com/ecommerce/recommendation-service/internal/repository"
	"github.com/stretchr/testify/mock"
)

type MockRecommendationRepository struct {
	mock.Mock
}

func (m *MockRecommendationRepository) RecordInteraction(ctx context.Context, interaction *models.UserInteraction) error {
	args := m.Called(ctx, interaction)
	return args.Error(0)
}

func (m *MockRecommendationRepository) GetUserInteractions(ctx context.Context, tenantID, userID string, limit int) ([]models.UserInteraction, error) {
	args := m.Called(ctx, tenantID, userID, limit)
	return args.Get(0).([]models.UserInteraction), args.Error(1)
}

func (m *MockRecommendationRepository) GetProductInteractors(ctx context.Context, tenantID, productID string) ([]string, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRecommendationRepository) UpsertSimilarity(ctx context.Context, similarity *models.ProductSimilarity) error {
	args := m.Called(ctx, similarity)
	return args.Error(0)
}

func (m *MockRecommendationRepository) GetSimilarProducts(ctx context.Context, tenantID, productID string, limit int) ([]models.ProductSimilarity, error) {
	args := m.Called(ctx, tenantID, productID, limit)
	return args.Get(0).([]models.ProductSimilarity), args.Error(1)
}

func (m *MockRecommendationRepository) DeleteSimilaritiesByTenant(ctx context.Context, tenantID string) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

func (m *MockRecommendationRepository) GetUserRecommendations(ctx context.Context, tenantID, userID string, limit int) ([]models.ProductRecommendation, error) {
	args := m.Called(ctx, tenantID, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ProductRecommendation), args.Error(1)
}

func (m *MockRecommendationRepository) GetCoPurchasePairs(ctx context.Context, tenantID string) ([]repository.CoPurchasePair, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]repository.CoPurchasePair), args.Error(1)
}

func (m *MockRecommendationRepository) GetDistinctProducts(ctx context.Context, tenantID string) ([]string, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRecommendationRepository) CreateTrainingJob(ctx context.Context, job *models.TrainingJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockRecommendationRepository) UpdateTrainingJob(ctx context.Context, job *models.TrainingJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockRecommendationRepository) GetTrainingJob(ctx context.Context, id string) (*models.TrainingJob, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TrainingJob), args.Error(1)
}

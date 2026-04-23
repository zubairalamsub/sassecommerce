package mocks

import (
	"context"

	"github.com/ecommerce/review-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockReviewRepository struct {
	mock.Mock
}

func (m *MockReviewRepository) Create(ctx context.Context, review *models.Review) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

func (m *MockReviewRepository) GetByID(ctx context.Context, id string) (*models.Review, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Review), args.Error(1)
}

func (m *MockReviewRepository) GetByProductID(ctx context.Context, tenantID, productID string, page, pageSize int) ([]models.Review, int64, error) {
	args := m.Called(ctx, tenantID, productID, page, pageSize)
	return args.Get(0).([]models.Review), args.Get(1).(int64), args.Error(2)
}

func (m *MockReviewRepository) GetByUserID(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.Review, int64, error) {
	args := m.Called(ctx, tenantID, userID, page, pageSize)
	return args.Get(0).([]models.Review), args.Get(1).(int64), args.Error(2)
}

func (m *MockReviewRepository) Update(ctx context.Context, review *models.Review) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

func (m *MockReviewRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReviewRepository) GetProductSummary(ctx context.Context, tenantID, productID string) (*models.ReviewSummary, error) {
	args := m.Called(ctx, tenantID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReviewSummary), args.Error(1)
}

func (m *MockReviewRepository) HasUserReviewed(ctx context.Context, tenantID, productID, userID string) (bool, error) {
	args := m.Called(ctx, tenantID, productID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockReviewRepository) AddHelpfulVote(ctx context.Context, id, voterID string, helpful bool) error {
	args := m.Called(ctx, id, voterID, helpful)
	return args.Error(0)
}

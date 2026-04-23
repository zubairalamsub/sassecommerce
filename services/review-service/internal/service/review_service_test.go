package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/review-service/internal/models"
	repoMocks "github.com/ecommerce/review-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (*reviewService, *repoMocks.MockReviewRepository) {
	mockRepo := new(repoMocks.MockReviewRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &reviewService{
		repo:   mockRepo,
		writer: nil, // no Kafka in tests
		logger: logger,
	}

	return svc, mockRepo
}

func createTestReview() *models.Review {
	now := time.Now().UTC()
	return &models.Review{
		ID:               "review-1",
		TenantID:         "tenant-1",
		ProductID:        "product-1",
		UserID:           "user-1",
		OrderID:          "order-1",
		Rating:           5,
		Title:            "Great product!",
		Comment:          "I love this product. Highly recommended.",
		Status:           models.StatusApproved,
		VerifiedPurchase: true,
		HelpfulCount:     3,
		UnhelpfulCount:   1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// === CreateReview Tests ===

func TestCreateReview_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("HasUserReviewed", ctx, "tenant-1", "product-1", "user-1").Return(false, nil)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Review")).Return(nil)

	req := &models.CreateReviewRequest{
		TenantID:  "tenant-1",
		ProductID: "product-1",
		UserID:    "user-1",
		OrderID:   "order-1",
		Rating:    5,
		Title:     "Great product!",
		Comment:   "Highly recommended.",
	}

	result, err := svc.CreateReview(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.Rating)
	assert.Equal(t, "Great product!", result.Title)
	assert.True(t, result.VerifiedPurchase)
	assert.Equal(t, models.StatusApproved, result.Status)
}

func TestCreateReview_NoOrderID_NotVerified(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("HasUserReviewed", ctx, "tenant-1", "product-1", "user-1").Return(false, nil)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Review")).Return(nil)

	req := &models.CreateReviewRequest{
		TenantID:  "tenant-1",
		ProductID: "product-1",
		UserID:    "user-1",
		Rating:    4,
		Title:     "Good",
		Comment:   "Nice product.",
	}

	result, err := svc.CreateReview(ctx, req)

	assert.NoError(t, err)
	assert.False(t, result.VerifiedPurchase)
}

func TestCreateReview_AlreadyReviewed(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("HasUserReviewed", ctx, "tenant-1", "product-1", "user-1").Return(true, nil)

	req := &models.CreateReviewRequest{
		TenantID:  "tenant-1",
		ProductID: "product-1",
		UserID:    "user-1",
		Rating:    5,
		Title:     "Test",
		Comment:   "Test",
	}

	result, err := svc.CreateReview(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already reviewed")
}

func TestCreateReview_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("HasUserReviewed", ctx, "tenant-1", "product-1", "user-1").Return(false, nil)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Review")).Return(errors.New("db error"))

	req := &models.CreateReviewRequest{
		TenantID:  "tenant-1",
		ProductID: "product-1",
		UserID:    "user-1",
		Rating:    5,
		Title:     "Test",
		Comment:   "Test",
	}

	result, err := svc.CreateReview(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetReview Tests ===

func TestGetReview_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)

	result, err := svc.GetReview(ctx, "review-1")

	assert.NoError(t, err)
	assert.Equal(t, "review-1", result.ID)
	assert.Equal(t, 5, result.Rating)
}

func TestGetReview_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.New("review not found"))

	result, err := svc.GetReview(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetProductReviews Tests ===

func TestGetProductReviews_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	reviews := []models.Review{*createTestReview()}
	mockRepo.On("GetByProductID", ctx, "tenant-1", "product-1", 1, 20).Return(reviews, int64(1), nil)

	results, total, err := svc.GetProductReviews(ctx, "tenant-1", "product-1", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
}

func TestGetProductReviews_Empty(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByProductID", ctx, "tenant-1", "product-1", 1, 20).Return([]models.Review{}, int64(0), nil)

	results, total, err := svc.GetProductReviews(ctx, "tenant-1", "product-1", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, results, 0)
}

// === GetUserReviews Tests ===

func TestGetUserReviews_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	reviews := []models.Review{*createTestReview()}
	mockRepo.On("GetByUserID", ctx, "tenant-1", "user-1", 1, 20).Return(reviews, int64(1), nil)

	results, total, err := svc.GetUserReviews(ctx, "tenant-1", "user-1", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
}

// === UpdateReview Tests ===

func TestUpdateReview_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Review")).Return(nil)

	newRating := 4
	req := &models.UpdateReviewRequest{
		Rating:  &newRating,
		Title:   "Updated title",
		Comment: "Updated comment",
	}

	result, err := svc.UpdateReview(ctx, "review-1", "user-1", req)

	assert.NoError(t, err)
	assert.Equal(t, 4, result.Rating)
	assert.Equal(t, "Updated title", result.Title)
}

func TestUpdateReview_Unauthorized(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)

	req := &models.UpdateReviewRequest{Title: "Hacked"}

	result, err := svc.UpdateReview(ctx, "review-1", "other-user", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestUpdateReview_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.New("review not found"))

	req := &models.UpdateReviewRequest{Title: "Test"}

	result, err := svc.UpdateReview(ctx, "nonexistent", "user-1", req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === DeleteReview Tests ===

func TestDeleteReview_ByOwner(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)
	mockRepo.On("Delete", ctx, "review-1").Return(nil)

	err := svc.DeleteReview(ctx, "review-1", "user-1")

	assert.NoError(t, err)
}

func TestDeleteReview_ByAdmin(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)
	mockRepo.On("Delete", ctx, "review-1").Return(nil)

	err := svc.DeleteReview(ctx, "review-1", "") // empty userID = admin

	assert.NoError(t, err)
}

func TestDeleteReview_Unauthorized(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)

	err := svc.DeleteReview(ctx, "review-1", "other-user")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestDeleteReview_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.New("review not found"))

	err := svc.DeleteReview(ctx, "nonexistent", "user-1")

	assert.Error(t, err)
}

// === ModerateReview Tests ===

func TestModerateReview_Approve(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	review.Status = models.StatusPending
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Review")).Return(nil)

	req := &models.ModerateReviewRequest{Status: "approved"}

	result, err := svc.ModerateReview(ctx, "review-1", req)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusApproved, result.Status)
}

func TestModerateReview_Reject(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Review")).Return(nil)

	req := &models.ModerateReviewRequest{
		Status:       "rejected",
		RejectReason: "Inappropriate content",
	}

	result, err := svc.ModerateReview(ctx, "review-1", req)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusRejected, result.Status)
}

func TestModerateReview_InvalidStatus(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)

	req := &models.ModerateReviewRequest{Status: "invalid"}

	result, err := svc.ModerateReview(ctx, "review-1", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid moderation status")
}

// === AddHelpfulVote Tests ===

func TestAddHelpfulVote_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)
	mockRepo.On("AddHelpfulVote", ctx, "review-1", "voter-1", true).Return(nil)

	req := &models.HelpfulVoteRequest{
		UserID:  "voter-1",
		Helpful: true,
	}

	err := svc.AddHelpfulVote(ctx, "review-1", req)

	assert.NoError(t, err)
}

func TestAddHelpfulVote_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.New("review not found"))

	req := &models.HelpfulVoteRequest{UserID: "voter-1", Helpful: true}

	err := svc.AddHelpfulVote(ctx, "nonexistent", req)

	assert.Error(t, err)
}

// === RespondToReview Tests ===

func TestRespondToReview_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	review := createTestReview()
	mockRepo.On("GetByID", ctx, "review-1").Return(review, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Review")).Return(nil)

	req := &models.SellerResponseRequest{
		Response: "Thank you for your feedback!",
	}

	result, err := svc.RespondToReview(ctx, "review-1", req)

	assert.NoError(t, err)
	assert.Equal(t, "Thank you for your feedback!", result.SellerResponse)
	assert.NotNil(t, result.SellerRespondedAt)
}

func TestRespondToReview_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.New("review not found"))

	req := &models.SellerResponseRequest{Response: "Thanks!"}

	result, err := svc.RespondToReview(ctx, "nonexistent", req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetProductSummary Tests ===

func TestGetProductSummary_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	summary := &models.ReviewSummary{
		ProductID:     "product-1",
		TenantID:      "tenant-1",
		AverageRating: 4.5,
		TotalReviews:  10,
		Distribution:  map[string]int{"1": 0, "2": 1, "3": 1, "4": 3, "5": 5},
	}
	mockRepo.On("GetProductSummary", ctx, "tenant-1", "product-1").Return(summary, nil)

	result, err := svc.GetProductSummary(ctx, "tenant-1", "product-1")

	assert.NoError(t, err)
	assert.Equal(t, 4.5, result.AverageRating)
	assert.Equal(t, 10, result.TotalReviews)
	assert.Equal(t, 5, result.Distribution["5"])
}

func TestGetProductSummary_NoReviews(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	summary := &models.ReviewSummary{
		ProductID:     "product-1",
		TenantID:      "tenant-1",
		AverageRating: 0,
		TotalReviews:  0,
		Distribution:  map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0},
	}
	mockRepo.On("GetProductSummary", ctx, "tenant-1", "product-1").Return(summary, nil)

	result, err := svc.GetProductSummary(ctx, "tenant-1", "product-1")

	assert.NoError(t, err)
	assert.Equal(t, 0.0, result.AverageRating)
	assert.Equal(t, 0, result.TotalReviews)
}

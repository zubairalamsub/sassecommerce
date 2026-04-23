package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ecommerce/review-service/internal/models"
	"github.com/ecommerce/review-service/internal/repository"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type ReviewService interface {
	CreateReview(ctx context.Context, req *models.CreateReviewRequest) (*models.ReviewResponse, error)
	GetReview(ctx context.Context, id string) (*models.ReviewResponse, error)
	GetProductReviews(ctx context.Context, tenantID, productID string, page, pageSize int) ([]models.ReviewResponse, int64, error)
	GetUserReviews(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.ReviewResponse, int64, error)
	UpdateReview(ctx context.Context, id, userID string, req *models.UpdateReviewRequest) (*models.ReviewResponse, error)
	DeleteReview(ctx context.Context, id, userID string) error
	ModerateReview(ctx context.Context, id string, req *models.ModerateReviewRequest) (*models.ReviewResponse, error)
	AddHelpfulVote(ctx context.Context, id string, req *models.HelpfulVoteRequest) error
	RespondToReview(ctx context.Context, id string, req *models.SellerResponseRequest) (*models.ReviewResponse, error)
	GetProductSummary(ctx context.Context, tenantID, productID string) (*models.ReviewSummaryResponse, error)
}

type reviewService struct {
	repo   repository.ReviewRepository
	writer *kafka.Writer
	logger *logrus.Logger
}

func NewReviewService(repo repository.ReviewRepository, writer *kafka.Writer, logger *logrus.Logger) ReviewService {
	return &reviewService{
		repo:   repo,
		writer: writer,
		logger: logger,
	}
}

func (s *reviewService) CreateReview(ctx context.Context, req *models.CreateReviewRequest) (*models.ReviewResponse, error) {
	// Check if user already reviewed this product
	hasReviewed, err := s.repo.HasUserReviewed(ctx, req.TenantID, req.ProductID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing review: %w", err)
	}
	if hasReviewed {
		return nil, errors.New("user has already reviewed this product")
	}

	review := &models.Review{
		ID:               uuid.New().String(),
		TenantID:         req.TenantID,
		ProductID:        req.ProductID,
		UserID:           req.UserID,
		OrderID:          req.OrderID,
		Rating:           req.Rating,
		Title:            req.Title,
		Comment:          req.Comment,
		Images:           req.Images,
		Status:           models.StatusApproved, // auto-approve for now
		VerifiedPurchase: req.OrderID != "",
	}

	if err := s.repo.Create(ctx, review); err != nil {
		s.logger.WithError(err).Error("Failed to create review")
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	s.publishEvent(ctx, "ReviewCreated", review)

	s.logger.WithFields(logrus.Fields{
		"review_id":  review.ID,
		"product_id": review.ProductID,
		"rating":     review.Rating,
	}).Info("Review created")

	return toReviewResponse(review), nil
}

func (s *reviewService) GetReview(ctx context.Context, id string) (*models.ReviewResponse, error) {
	review, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toReviewResponse(review), nil
}

func (s *reviewService) GetProductReviews(ctx context.Context, tenantID, productID string, page, pageSize int) ([]models.ReviewResponse, int64, error) {
	reviews, total, err := s.repo.GetByProductID(ctx, tenantID, productID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.ReviewResponse, len(reviews))
	for i, r := range reviews {
		responses[i] = *toReviewResponse(&r)
	}

	return responses, total, nil
}

func (s *reviewService) GetUserReviews(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.ReviewResponse, int64, error) {
	reviews, total, err := s.repo.GetByUserID(ctx, tenantID, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.ReviewResponse, len(reviews))
	for i, r := range reviews {
		responses[i] = *toReviewResponse(&r)
	}

	return responses, total, nil
}

func (s *reviewService) UpdateReview(ctx context.Context, id, userID string, req *models.UpdateReviewRequest) (*models.ReviewResponse, error) {
	review, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if review.UserID != userID {
		return nil, errors.New("unauthorized: you can only update your own reviews")
	}

	if req.Rating != nil {
		review.Rating = *req.Rating
	}
	if req.Title != "" {
		review.Title = req.Title
	}
	if req.Comment != "" {
		review.Comment = req.Comment
	}
	if req.Images != nil {
		review.Images = req.Images
	}

	if err := s.repo.Update(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	s.publishEvent(ctx, "ReviewUpdated", review)

	return toReviewResponse(review), nil
}

func (s *reviewService) DeleteReview(ctx context.Context, id, userID string) error {
	review, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Allow deletion by the review owner or admin (userID empty = admin)
	if userID != "" && review.UserID != userID {
		return errors.New("unauthorized: you can only delete your own reviews")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete review: %w", err)
	}

	s.publishEvent(ctx, "ReviewDeleted", review)

	s.logger.WithField("review_id", id).Info("Review deleted")
	return nil
}

func (s *reviewService) ModerateReview(ctx context.Context, id string, req *models.ModerateReviewRequest) (*models.ReviewResponse, error) {
	review, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	status := models.ReviewStatus(req.Status)
	if status != models.StatusApproved && status != models.StatusRejected && status != models.StatusFlagged {
		return nil, fmt.Errorf("invalid moderation status: %s", req.Status)
	}

	review.Status = status
	if status == models.StatusRejected {
		review.RejectReason = req.RejectReason
	}

	if err := s.repo.Update(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to moderate review: %w", err)
	}

	return toReviewResponse(review), nil
}

func (s *reviewService) AddHelpfulVote(ctx context.Context, id string, req *models.HelpfulVoteRequest) error {
	// Verify review exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.AddHelpfulVote(ctx, id, req.UserID, req.Helpful)
}

func (s *reviewService) RespondToReview(ctx context.Context, id string, req *models.SellerResponseRequest) (*models.ReviewResponse, error) {
	review, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	review.SellerResponse = req.Response
	review.SellerRespondedAt = &now

	if err := s.repo.Update(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to save seller response: %w", err)
	}

	return toReviewResponse(review), nil
}

func (s *reviewService) GetProductSummary(ctx context.Context, tenantID, productID string) (*models.ReviewSummaryResponse, error) {
	summary, err := s.repo.GetProductSummary(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	return &models.ReviewSummaryResponse{
		ProductID:     summary.ProductID,
		AverageRating: summary.AverageRating,
		TotalReviews:  summary.TotalReviews,
		Distribution:  summary.Distribution,
	}, nil
}

func (s *reviewService) publishEvent(ctx context.Context, eventType string, review *models.Review) {
	if s.writer == nil {
		return
	}

	event := map[string]interface{}{
		"event_id":   uuid.New().String(),
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    "1.0.0",
		"payload": map[string]interface{}{
			"review_id":  review.ID,
			"tenant_id":  review.TenantID,
			"product_id": review.ProductID,
			"user_id":    review.UserID,
			"rating":     review.Rating,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to marshal review event")
		return
	}

	msg := kafka.Message{
		Topic: "review-events",
		Key:   []byte(review.ID),
		Value: data,
	}

	if err := s.writer.WriteMessages(ctx, msg); err != nil {
		s.logger.WithError(err).Warn("Failed to publish review event")
	}
}

func toReviewResponse(r *models.Review) *models.ReviewResponse {
	return &models.ReviewResponse{
		ID:                r.ID,
		TenantID:          r.TenantID,
		ProductID:         r.ProductID,
		UserID:            r.UserID,
		OrderID:           r.OrderID,
		Rating:            r.Rating,
		Title:             r.Title,
		Comment:           r.Comment,
		Images:            r.Images,
		Status:            r.Status,
		HelpfulCount:      r.HelpfulCount,
		UnhelpfulCount:    r.UnhelpfulCount,
		VerifiedPurchase:  r.VerifiedPurchase,
		SellerResponse:    r.SellerResponse,
		SellerRespondedAt: r.SellerRespondedAt,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

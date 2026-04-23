package models

import (
	"time"
)

// ReviewStatus represents the moderation status of a review
type ReviewStatus string

const (
	StatusPending  ReviewStatus = "pending"
	StatusApproved ReviewStatus = "approved"
	StatusRejected ReviewStatus = "rejected"
	StatusFlagged  ReviewStatus = "flagged"
)

// Review represents a product review
type Review struct {
	ID        string       `bson:"_id,omitempty" json:"id"`
	TenantID  string       `bson:"tenant_id" json:"tenant_id"`
	ProductID string       `bson:"product_id" json:"product_id"`
	UserID    string       `bson:"user_id" json:"user_id"`
	OrderID   string       `bson:"order_id,omitempty" json:"order_id,omitempty"`

	// Review content
	Rating  int    `bson:"rating" json:"rating"` // 1-5
	Title   string `bson:"title" json:"title"`
	Comment string `bson:"comment" json:"comment"`

	// Media
	Images []string `bson:"images,omitempty" json:"images,omitempty"`

	// Moderation
	Status       ReviewStatus `bson:"status" json:"status"`
	RejectReason string       `bson:"reject_reason,omitempty" json:"reject_reason,omitempty"`

	// Helpfulness
	HelpfulCount   int      `bson:"helpful_count" json:"helpful_count"`
	UnhelpfulCount int      `bson:"unhelpful_count" json:"unhelpful_count"`
	HelpfulVoters  []string `bson:"helpful_voters,omitempty" json:"-"`

	// Verified purchase
	VerifiedPurchase bool `bson:"verified_purchase" json:"verified_purchase"`

	// Seller response
	SellerResponse   string     `bson:"seller_response,omitempty" json:"seller_response,omitempty"`
	SellerRespondedAt *time.Time `bson:"seller_responded_at,omitempty" json:"seller_responded_at,omitempty"`

	// Timestamps
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// ReviewSummary holds aggregated rating data for a product
type ReviewSummary struct {
	ProductID    string             `bson:"_id" json:"product_id"`
	TenantID     string             `bson:"tenant_id" json:"tenant_id"`
	AverageRating float64           `bson:"average_rating" json:"average_rating"`
	TotalReviews int               `bson:"total_reviews" json:"total_reviews"`
	Distribution map[string]int    `bson:"distribution" json:"distribution"` // "1":5, "2":3, etc.
}

// === Request DTOs ===

type CreateReviewRequest struct {
	TenantID  string   `json:"tenant_id" binding:"required"`
	ProductID string   `json:"product_id" binding:"required"`
	UserID    string   `json:"user_id" binding:"required"`
	OrderID   string   `json:"order_id,omitempty"`
	Rating    int      `json:"rating" binding:"required,min=1,max=5"`
	Title     string   `json:"title" binding:"required"`
	Comment   string   `json:"comment" binding:"required"`
	Images    []string `json:"images,omitempty"`
}

type UpdateReviewRequest struct {
	Rating  *int     `json:"rating,omitempty" binding:"omitempty,min=1,max=5"`
	Title   string   `json:"title,omitempty"`
	Comment string   `json:"comment,omitempty"`
	Images  []string `json:"images,omitempty"`
}

type ModerateReviewRequest struct {
	Status       string `json:"status" binding:"required"`
	RejectReason string `json:"reject_reason,omitempty"`
}

type SellerResponseRequest struct {
	Response string `json:"response" binding:"required"`
}

type HelpfulVoteRequest struct {
	UserID  string `json:"user_id" binding:"required"`
	Helpful bool   `json:"helpful"`
}

// === Response DTOs ===

type ReviewResponse struct {
	ID               string       `json:"id"`
	TenantID         string       `json:"tenant_id"`
	ProductID        string       `json:"product_id"`
	UserID           string       `json:"user_id"`
	OrderID          string       `json:"order_id,omitempty"`
	Rating           int          `json:"rating"`
	Title            string       `json:"title"`
	Comment          string       `json:"comment"`
	Images           []string     `json:"images,omitempty"`
	Status           ReviewStatus `json:"status"`
	HelpfulCount     int          `json:"helpful_count"`
	UnhelpfulCount   int          `json:"unhelpful_count"`
	VerifiedPurchase bool         `json:"verified_purchase"`
	SellerResponse   string       `json:"seller_response,omitempty"`
	SellerRespondedAt *time.Time  `json:"seller_responded_at,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type ReviewSummaryResponse struct {
	ProductID     string          `json:"product_id"`
	AverageRating float64        `json:"average_rating"`
	TotalReviews  int            `json:"total_reviews"`
	Distribution  map[string]int `json:"distribution"`
}

// EventEnvelope is the Kafka event wire format
type EventEnvelope struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	AggregateID   string                 `json:"aggregate_id,omitempty"`
	AggregateType string                 `json:"aggregate_type,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Version       string                 `json:"version,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	Data          map[string]interface{} `json:"data,omitempty"`
}

func (e *EventEnvelope) GetPayload() map[string]interface{} {
	if e.Payload != nil {
		return e.Payload
	}
	return e.Data
}

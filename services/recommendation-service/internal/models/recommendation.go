package models

import (
	"encoding/json"
	"time"
)

// === Interaction Types ===

const (
	InteractionView     = "view"
	InteractionCart      = "cart"
	InteractionPurchase  = "purchase"
	InteractionWishlist  = "wishlist"
)

// Interaction weights for scoring
var InteractionWeights = map[string]float64{
	InteractionView:     1.0,
	InteractionCart:      3.0,
	InteractionPurchase:  5.0,
	InteractionWishlist:  2.0,
}

// === Database Models ===

type UserInteraction struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID  string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);index:idx_user_product"`
	ProductID string    `json:"product_id" gorm:"type:varchar(36);index:idx_user_product"`
	Type      string    `json:"type" gorm:"type:varchar(20)"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type ProductSimilarity struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID     string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	ProductID    string    `json:"product_id" gorm:"type:varchar(36);index:idx_similarity"`
	SimilarID    string    `json:"similar_id" gorm:"type:varchar(36);index:idx_similarity"`
	Score        float64   `json:"score"`
	Reason       string    `json:"reason" gorm:"type:varchar(50)"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type TrainingJob struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID    string     `json:"tenant_id" gorm:"type:varchar(36);index"`
	Status      string     `json:"status" gorm:"type:varchar(20);default:'pending'"`
	ItemsProcessed int    `json:"items_processed"`
	SimilaritiesGenerated int `json:"similarities_generated"`
	StartedAt   time.Time  `json:"started_at" gorm:"autoCreateTime"`
	CompletedAt *time.Time `json:"completed_at"`
	Error       string     `json:"error" gorm:"type:text"`
}

// === Response DTOs ===

type RecommendationResponse struct {
	UserID          string              `json:"user_id,omitempty"`
	ProductID       string              `json:"product_id,omitempty"`
	Recommendations []ProductRecommendation `json:"recommendations"`
	Strategy        string              `json:"strategy"`
	GeneratedAt     time.Time           `json:"generated_at"`
}

type ProductRecommendation struct {
	ProductID string  `json:"product_id"`
	Score     float64 `json:"score"`
	Reason    string  `json:"reason"`
}

type TrainingJobResponse struct {
	ID                    string     `json:"id"`
	TenantID              string     `json:"tenant_id"`
	Status                string     `json:"status"`
	ItemsProcessed        int        `json:"items_processed"`
	SimilaritiesGenerated int        `json:"similarities_generated"`
	StartedAt             time.Time  `json:"started_at"`
	CompletedAt           *time.Time `json:"completed_at,omitempty"`
	Error                 string     `json:"error,omitempty"`
}

// === Request DTOs ===

type TrainRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
}

// === Kafka Event Envelope ===

type EventEnvelope struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string           `json:"source"`
	Payload  json.RawMessage  `json:"payload,omitempty"`
	Data     json.RawMessage  `json:"data,omitempty"`
}

func (e *EventEnvelope) GetPayload() map[string]interface{} {
	raw := e.Payload
	if len(raw) == 0 {
		raw = e.Data
	}
	if len(raw) == 0 {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}

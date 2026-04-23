package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecommerce/cart-service/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	cartKeyPrefix = "cart:"
	cartTTL       = 7 * 24 * time.Hour // 7 days
)

// CartRepository defines the interface for cart data access
type CartRepository interface {
	GetCart(ctx context.Context, tenantID, userID string) (*models.Cart, error)
	SaveCart(ctx context.Context, cart *models.Cart) error
	DeleteCart(ctx context.Context, tenantID, userID string) error
	GetCartsByProduct(ctx context.Context, productID string) ([]string, error)
	AddProductCartMapping(ctx context.Context, productID, cartKey string) error
	RemoveProductCartMapping(ctx context.Context, productID, cartKey string) error
}

type redisCartRepository struct {
	client *redis.Client
}

// NewCartRepository creates a new Redis-backed cart repository
func NewCartRepository(client *redis.Client) CartRepository {
	return &redisCartRepository{client: client}
}

func cartKey(tenantID, userID string) string {
	return fmt.Sprintf("%s%s:%s", cartKeyPrefix, tenantID, userID)
}

func productCartSetKey(productID string) string {
	return fmt.Sprintf("product_carts:%s", productID)
}

func (r *redisCartRepository) GetCart(ctx context.Context, tenantID, userID string) (*models.Cart, error) {
	key := cartKey(tenantID, userID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		// Return empty cart
		return &models.Cart{
			TenantID:  tenantID,
			UserID:    userID,
			Items:     []models.CartItem{},
			UpdatedAt: time.Now().UTC(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	var cart models.Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cart: %w", err)
	}

	return &cart, nil
}

func (r *redisCartRepository) SaveCart(ctx context.Context, cart *models.Cart) error {
	key := cartKey(cart.TenantID, cart.UserID)
	cart.UpdatedAt = time.Now().UTC()

	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("failed to marshal cart: %w", err)
	}

	if err := r.client.Set(ctx, key, data, cartTTL).Err(); err != nil {
		return fmt.Errorf("failed to save cart: %w", err)
	}

	return nil
}

func (r *redisCartRepository) DeleteCart(ctx context.Context, tenantID, userID string) error {
	key := cartKey(tenantID, userID)

	// Get the cart first to clean up product mappings
	cart, err := r.GetCart(ctx, tenantID, userID)
	if err == nil && len(cart.Items) > 0 {
		for _, item := range cart.Items {
			r.RemoveProductCartMapping(ctx, item.ProductID, key)
		}
	}

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}

	return nil
}

func (r *redisCartRepository) GetCartsByProduct(ctx context.Context, productID string) ([]string, error) {
	key := productCartSetKey(productID)
	members, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get carts by product: %w", err)
	}
	return members, nil
}

func (r *redisCartRepository) AddProductCartMapping(ctx context.Context, productID, cartKey string) error {
	key := productCartSetKey(productID)
	return r.client.SAdd(ctx, key, cartKey).Err()
}

func (r *redisCartRepository) RemoveProductCartMapping(ctx context.Context, productID, cartKey string) error {
	key := productCartSetKey(productID)
	return r.client.SRem(ctx, key, cartKey).Err()
}

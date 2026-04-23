package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecommerce/cart-service/internal/models"
	"github.com/ecommerce/cart-service/internal/repository"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// CartService defines the interface for cart business logic
type CartService interface {
	AddItem(ctx context.Context, req *models.AddItemRequest) (*models.CartResponse, error)
	GetCart(ctx context.Context, tenantID, userID string) (*models.CartResponse, error)
	UpdateItem(ctx context.Context, tenantID, userID, itemID string, req *models.UpdateItemRequest) (*models.CartResponse, error)
	RemoveItem(ctx context.Context, tenantID, userID, itemID string) (*models.CartResponse, error)
	ClearCart(ctx context.Context, tenantID, userID string) error
	UpdateProductPrice(ctx context.Context, productID string, newPrice float64) error
	RemoveProduct(ctx context.Context, productID string) error
}

type cartService struct {
	repo   repository.CartRepository
	writer *kafka.Writer
	logger *logrus.Logger
}

// NewCartService creates a new CartService instance
func NewCartService(repo repository.CartRepository, writer *kafka.Writer, logger *logrus.Logger) CartService {
	return &cartService{
		repo:   repo,
		writer: writer,
		logger: logger,
	}
}

func (s *cartService) AddItem(ctx context.Context, req *models.AddItemRequest) (*models.CartResponse, error) {
	cart, err := s.repo.GetCart(ctx, req.TenantID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// Check if item with the same product already exists
	found := false
	for i, item := range cart.Items {
		if item.ProductID == req.ProductID {
			cart.Items[i].Quantity += req.Quantity
			cart.Items[i].Price = req.Price // update price to latest
			cart.Items[i].Name = req.Name
			found = true
			break
		}
	}

	if !found {
		newItem := models.CartItem{
			ID:        uuid.New().String(),
			ProductID: req.ProductID,
			Name:      req.Name,
			Price:     req.Price,
			Quantity:  req.Quantity,
			ImageURL:  req.ImageURL,
			AddedAt:   time.Now().UTC().Format(time.RFC3339),
		}
		cart.Items = append(cart.Items, newItem)
	}

	if err := s.repo.SaveCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to save cart: %w", err)
	}

	// Track product-cart mapping
	cartKey := fmt.Sprintf("cart:%s:%s", req.TenantID, req.UserID)
	s.repo.AddProductCartMapping(ctx, req.ProductID, cartKey)

	s.publishCartUpdated(cart)

	return toCartResponse(cart), nil
}

func (s *cartService) GetCart(ctx context.Context, tenantID, userID string) (*models.CartResponse, error) {
	cart, err := s.repo.GetCart(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	return toCartResponse(cart), nil
}

func (s *cartService) UpdateItem(ctx context.Context, tenantID, userID, itemID string, req *models.UpdateItemRequest) (*models.CartResponse, error) {
	cart, err := s.repo.GetCart(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	found := false
	for i, item := range cart.Items {
		if item.ID == itemID {
			cart.Items[i].Quantity = req.Quantity
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("item not found in cart")
	}

	if err := s.repo.SaveCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to save cart: %w", err)
	}

	s.publishCartUpdated(cart)

	return toCartResponse(cart), nil
}

func (s *cartService) RemoveItem(ctx context.Context, tenantID, userID, itemID string) (*models.CartResponse, error) {
	cart, err := s.repo.GetCart(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	found := false
	var removedProductID string
	newItems := make([]models.CartItem, 0)
	for _, item := range cart.Items {
		if item.ID == itemID {
			found = true
			removedProductID = item.ProductID
			continue
		}
		newItems = append(newItems, item)
	}

	if !found {
		return nil, fmt.Errorf("item not found in cart")
	}

	cart.Items = newItems

	if err := s.repo.SaveCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to save cart: %w", err)
	}

	// Clean up product-cart mapping if no more items with that product
	if removedProductID != "" {
		hasProduct := false
		for _, item := range cart.Items {
			if item.ProductID == removedProductID {
				hasProduct = true
				break
			}
		}
		if !hasProduct {
			cartKey := fmt.Sprintf("cart:%s:%s", tenantID, userID)
			s.repo.RemoveProductCartMapping(ctx, removedProductID, cartKey)
		}
	}

	s.publishCartUpdated(cart)

	return toCartResponse(cart), nil
}

func (s *cartService) ClearCart(ctx context.Context, tenantID, userID string) error {
	if err := s.repo.DeleteCart(ctx, tenantID, userID); err != nil {
		return fmt.Errorf("failed to clear cart: %w", err)
	}

	return nil
}

func (s *cartService) UpdateProductPrice(ctx context.Context, productID string, newPrice float64) error {
	cartKeys, err := s.repo.GetCartsByProduct(ctx, productID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get carts by product")
		return err
	}

	for _, key := range cartKeys {
		tenantID, userID := parseCartKey(key)
		if tenantID == "" || userID == "" {
			continue
		}

		cart, err := s.repo.GetCart(ctx, tenantID, userID)
		if err != nil {
			s.logger.WithError(err).WithField("cart_key", key).Error("Failed to get cart for price update")
			continue
		}

		updated := false
		for i, item := range cart.Items {
			if item.ProductID == productID {
				cart.Items[i].Price = newPrice
				updated = true
			}
		}

		if updated {
			if err := s.repo.SaveCart(ctx, cart); err != nil {
				s.logger.WithError(err).WithField("cart_key", key).Error("Failed to save cart after price update")
			}
		}
	}

	return nil
}

func (s *cartService) RemoveProduct(ctx context.Context, productID string) error {
	cartKeys, err := s.repo.GetCartsByProduct(ctx, productID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get carts by product")
		return err
	}

	for _, key := range cartKeys {
		tenantID, userID := parseCartKey(key)
		if tenantID == "" || userID == "" {
			continue
		}

		cart, err := s.repo.GetCart(ctx, tenantID, userID)
		if err != nil {
			s.logger.WithError(err).WithField("cart_key", key).Error("Failed to get cart for product removal")
			continue
		}

		newItems := make([]models.CartItem, 0)
		for _, item := range cart.Items {
			if item.ProductID != productID {
				newItems = append(newItems, item)
			}
		}

		cart.Items = newItems

		if err := s.repo.SaveCart(ctx, cart); err != nil {
			s.logger.WithError(err).WithField("cart_key", key).Error("Failed to save cart after product removal")
		}
	}

	// Clean up the product-cart mapping set
	s.repo.RemoveProductCartMapping(ctx, productID, "")

	return nil
}

func (s *cartService) publishCartUpdated(cart *models.Cart) {
	if s.writer == nil {
		return
	}

	event := models.CartEvent{
		EventID:   uuid.New().String(),
		EventType: "CartUpdated",
		Timestamp: time.Now().UTC(),
		Payload: models.CartUpdatedPayload{
			TenantID:    cart.TenantID,
			UserID:      cart.UserID,
			TotalItems:  cart.TotalItems(),
			TotalAmount: cart.TotalAmount(),
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal cart event")
		return
	}

	err = s.writer.WriteMessages(context.Background(), kafka.Message{
		Topic: "cart-events",
		Key:   []byte(fmt.Sprintf("%s:%s", cart.TenantID, cart.UserID)),
		Value: data,
	})
	if err != nil {
		s.logger.WithError(err).Error("Failed to publish cart event")
	}
}

// parseCartKey extracts tenantID and userID from a cart key like "cart:tenant-1:user-1"
func parseCartKey(key string) (string, string) {
	// key format: "cart:tenantID:userID"
	if len(key) < 6 {
		return "", ""
	}
	// Remove "cart:" prefix
	rest := key[5:]
	for i, c := range rest {
		if c == ':' {
			return rest[:i], rest[i+1:]
		}
	}
	return "", ""
}

func toCartResponse(cart *models.Cart) *models.CartResponse {
	items := make([]models.CartItemResponse, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = models.CartItemResponse{
			ID:        item.ID,
			ProductID: item.ProductID,
			Name:      item.Name,
			Price:     item.Price,
			Quantity:  item.Quantity,
			Subtotal:  item.Price * float64(item.Quantity),
			ImageURL:  item.ImageURL,
			AddedAt:   item.AddedAt,
		}
	}

	return &models.CartResponse{
		TenantID:    cart.TenantID,
		UserID:      cart.UserID,
		Items:       items,
		TotalItems:  cart.TotalItems(),
		TotalAmount: cart.TotalAmount(),
		UpdatedAt:   cart.UpdatedAt,
	}
}

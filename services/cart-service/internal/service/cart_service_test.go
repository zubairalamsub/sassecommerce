package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/cart-service/internal/models"
	repoMocks "github.com/ecommerce/cart-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (*cartService, *repoMocks.MockCartRepository) {
	mockRepo := new(repoMocks.MockCartRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &cartService{
		repo:   mockRepo,
		writer: nil, // no Kafka in tests
		logger: logger,
	}

	return svc, mockRepo
}

func createEmptyCart() *models.Cart {
	return &models.Cart{
		TenantID:  "tenant-1",
		UserID:    "user-1",
		Items:     []models.CartItem{},
		UpdatedAt: time.Now().UTC(),
	}
}

func createCartWithItems() *models.Cart {
	return &models.Cart{
		TenantID: "tenant-1",
		UserID:   "user-1",
		Items: []models.CartItem{
			{
				ID:        "item-1",
				ProductID: "product-1",
				Name:      "Widget A",
				Price:     29.99,
				Quantity:  2,
				AddedAt:   time.Now().UTC().Format(time.RFC3339),
			},
			{
				ID:        "item-2",
				ProductID: "product-2",
				Name:      "Widget B",
				Price:     49.99,
				Quantity:  1,
				AddedAt:   time.Now().UTC().Format(time.RFC3339),
			},
		},
		UpdatedAt: time.Now().UTC(),
	}
}

// === AddItem Tests ===

func TestAddItem_Success_NewItem(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(createEmptyCart(), nil)
	mockRepo.On("SaveCart", ctx, mock.AnythingOfType("*models.Cart")).Return(nil)
	mockRepo.On("AddProductCartMapping", ctx, "product-1", "cart:tenant-1:user-1").Return(nil)

	req := &models.AddItemRequest{
		TenantID:  "tenant-1",
		UserID:    "user-1",
		ProductID: "product-1",
		Name:      "Widget A",
		Price:     29.99,
		Quantity:  2,
	}

	result, err := svc.AddItem(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Items))
	assert.Equal(t, "product-1", result.Items[0].ProductID)
	assert.Equal(t, 2, result.Items[0].Quantity)
	assert.Equal(t, 59.98, result.TotalAmount)
	assert.Equal(t, 2, result.TotalItems)
}

func TestAddItem_Success_ExistingProduct(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	cart := createCartWithItems()
	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(cart, nil)
	mockRepo.On("SaveCart", ctx, mock.AnythingOfType("*models.Cart")).Return(nil)
	mockRepo.On("AddProductCartMapping", ctx, "product-1", "cart:tenant-1:user-1").Return(nil)

	req := &models.AddItemRequest{
		TenantID:  "tenant-1",
		UserID:    "user-1",
		ProductID: "product-1",
		Name:      "Widget A",
		Price:     29.99,
		Quantity:  3,
	}

	result, err := svc.AddItem(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Items)) // still 2 distinct items
	// original had 2, adding 3 more = 5
	assert.Equal(t, 5, result.Items[0].Quantity)
}

func TestAddItem_GetCartFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(nil, errors.New("redis error"))

	req := &models.AddItemRequest{
		TenantID:  "tenant-1",
		UserID:    "user-1",
		ProductID: "product-1",
		Name:      "Widget",
		Price:     10.0,
		Quantity:  1,
	}

	result, err := svc.AddItem(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get cart")
}

func TestAddItem_SaveCartFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(createEmptyCart(), nil)
	mockRepo.On("SaveCart", ctx, mock.AnythingOfType("*models.Cart")).Return(errors.New("redis error"))

	req := &models.AddItemRequest{
		TenantID:  "tenant-1",
		UserID:    "user-1",
		ProductID: "product-1",
		Name:      "Widget",
		Price:     10.0,
		Quantity:  1,
	}

	result, err := svc.AddItem(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save cart")
}

// === GetCart Tests ===

func TestGetCart_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	cart := createCartWithItems()
	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(cart, nil)

	result, err := svc.GetCart(ctx, "tenant-1", "user-1")

	assert.NoError(t, err)
	assert.Equal(t, "tenant-1", result.TenantID)
	assert.Equal(t, "user-1", result.UserID)
	assert.Equal(t, 2, len(result.Items))
	assert.Equal(t, 3, result.TotalItems)                       // 2 + 1
	assert.InDelta(t, 109.97, result.TotalAmount, 0.01) // 29.99*2 + 49.99
}

func TestGetCart_Empty(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(createEmptyCart(), nil)

	result, err := svc.GetCart(ctx, "tenant-1", "user-1")

	assert.NoError(t, err)
	assert.Equal(t, 0, len(result.Items))
	assert.Equal(t, 0, result.TotalItems)
	assert.Equal(t, 0.0, result.TotalAmount)
}

func TestGetCart_Failure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(nil, errors.New("redis error"))

	result, err := svc.GetCart(ctx, "tenant-1", "user-1")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === UpdateItem Tests ===

func TestUpdateItem_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	cart := createCartWithItems()
	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(cart, nil)
	mockRepo.On("SaveCart", ctx, mock.AnythingOfType("*models.Cart")).Return(nil)

	req := &models.UpdateItemRequest{Quantity: 5}

	result, err := svc.UpdateItem(ctx, "tenant-1", "user-1", "item-1", req)

	assert.NoError(t, err)
	assert.Equal(t, 5, result.Items[0].Quantity)
}

func TestUpdateItem_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	cart := createCartWithItems()
	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(cart, nil)

	req := &models.UpdateItemRequest{Quantity: 5}

	result, err := svc.UpdateItem(ctx, "tenant-1", "user-1", "nonexistent", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "item not found")
}

func TestUpdateItem_GetCartFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(nil, errors.New("redis error"))

	req := &models.UpdateItemRequest{Quantity: 5}

	result, err := svc.UpdateItem(ctx, "tenant-1", "user-1", "item-1", req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUpdateItem_SaveFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	cart := createCartWithItems()
	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(cart, nil)
	mockRepo.On("SaveCart", ctx, mock.AnythingOfType("*models.Cart")).Return(errors.New("redis error"))

	req := &models.UpdateItemRequest{Quantity: 5}

	result, err := svc.UpdateItem(ctx, "tenant-1", "user-1", "item-1", req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === RemoveItem Tests ===

func TestRemoveItem_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	cart := createCartWithItems()
	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(cart, nil)
	mockRepo.On("SaveCart", ctx, mock.AnythingOfType("*models.Cart")).Return(nil)
	mockRepo.On("RemoveProductCartMapping", ctx, "product-1", "cart:tenant-1:user-1").Return(nil)

	result, err := svc.RemoveItem(ctx, "tenant-1", "user-1", "item-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Items))
	assert.Equal(t, "product-2", result.Items[0].ProductID)
}

func TestRemoveItem_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	cart := createCartWithItems()
	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(cart, nil)

	result, err := svc.RemoveItem(ctx, "tenant-1", "user-1", "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "item not found")
}

func TestRemoveItem_GetCartFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(nil, errors.New("redis error"))

	result, err := svc.RemoveItem(ctx, "tenant-1", "user-1", "item-1")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === ClearCart Tests ===

func TestClearCart_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("DeleteCart", ctx, "tenant-1", "user-1").Return(nil)

	err := svc.ClearCart(ctx, "tenant-1", "user-1")

	assert.NoError(t, err)
}

func TestClearCart_Failure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("DeleteCart", ctx, "tenant-1", "user-1").Return(errors.New("redis error"))

	err := svc.ClearCart(ctx, "tenant-1", "user-1")

	assert.Error(t, err)
}

// === UpdateProductPrice Tests ===

func TestUpdateProductPrice_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCartsByProduct", ctx, "product-1").Return([]string{"cart:tenant-1:user-1"}, nil)
	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(createCartWithItems(), nil)
	mockRepo.On("SaveCart", ctx, mock.AnythingOfType("*models.Cart")).Return(nil)

	err := svc.UpdateProductPrice(ctx, "product-1", 19.99)

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "SaveCart", ctx, mock.AnythingOfType("*models.Cart"))
}

func TestUpdateProductPrice_NoCarts(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCartsByProduct", ctx, "product-1").Return([]string{}, nil)

	err := svc.UpdateProductPrice(ctx, "product-1", 19.99)

	assert.NoError(t, err)
}

func TestUpdateProductPrice_GetCartsFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCartsByProduct", ctx, "product-1").Return([]string{}, errors.New("redis error"))

	err := svc.UpdateProductPrice(ctx, "product-1", 19.99)

	assert.Error(t, err)
}

// === RemoveProduct Tests ===

func TestRemoveProduct_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCartsByProduct", ctx, "product-1").Return([]string{"cart:tenant-1:user-1"}, nil)
	mockRepo.On("GetCart", ctx, "tenant-1", "user-1").Return(createCartWithItems(), nil)
	mockRepo.On("SaveCart", ctx, mock.AnythingOfType("*models.Cart")).Return(nil)
	mockRepo.On("RemoveProductCartMapping", ctx, "product-1", "").Return(nil)

	err := svc.RemoveProduct(ctx, "product-1")

	assert.NoError(t, err)
}

func TestRemoveProduct_NoCarts(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCartsByProduct", ctx, "product-1").Return([]string{}, nil)
	mockRepo.On("RemoveProductCartMapping", ctx, "product-1", "").Return(nil)

	err := svc.RemoveProduct(ctx, "product-1")

	assert.NoError(t, err)
}

func TestRemoveProduct_GetCartsFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCartsByProduct", ctx, "product-1").Return([]string{}, errors.New("redis error"))

	err := svc.RemoveProduct(ctx, "product-1")

	assert.Error(t, err)
}

// === parseCartKey Tests ===

func TestParseCartKey_Valid(t *testing.T) {
	tenantID, userID := parseCartKey("cart:tenant-1:user-1")
	assert.Equal(t, "tenant-1", tenantID)
	assert.Equal(t, "user-1", userID)
}

func TestParseCartKey_Invalid(t *testing.T) {
	tenantID, userID := parseCartKey("bad")
	assert.Equal(t, "", tenantID)
	assert.Equal(t, "", userID)
}

func TestParseCartKey_TooShort(t *testing.T) {
	tenantID, userID := parseCartKey("abc")
	assert.Equal(t, "", tenantID)
	assert.Equal(t, "", userID)
}

// === toCartResponse Tests ===

func TestToCartResponse(t *testing.T) {
	cart := createCartWithItems()
	resp := toCartResponse(cart)

	assert.Equal(t, "tenant-1", resp.TenantID)
	assert.Equal(t, "user-1", resp.UserID)
	assert.Equal(t, 2, len(resp.Items))
	assert.Equal(t, 3, resp.TotalItems)
	assert.InDelta(t, 109.97, resp.TotalAmount, 0.01)

	// Check subtotals
	assert.InDelta(t, 59.98, resp.Items[0].Subtotal, 0.01) // 29.99 * 2
	assert.InDelta(t, 49.99, resp.Items[1].Subtotal, 0.01) // 49.99 * 1
}

func TestToCartResponse_EmptyCart(t *testing.T) {
	cart := createEmptyCart()
	resp := toCartResponse(cart)

	assert.Equal(t, 0, len(resp.Items))
	assert.Equal(t, 0, resp.TotalItems)
	assert.Equal(t, 0.0, resp.TotalAmount)
}

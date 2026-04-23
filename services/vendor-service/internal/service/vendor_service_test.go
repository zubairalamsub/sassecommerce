package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/vendor-service/internal/models"
	repoMocks "github.com/ecommerce/vendor-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (*vendorService, *repoMocks.MockVendorRepository) {
	mockRepo := new(repoMocks.MockVendorRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &vendorService{
		repo:   mockRepo,
		writer: nil,
		logger: logger,
	}

	return svc, mockRepo
}

func createTestVendor() *models.Vendor {
	return &models.Vendor{
		ID:             "vendor-1",
		TenantID:       "tenant-1",
		Name:           "Acme Corp",
		Email:          "vendor@acme.com",
		Phone:          "+1234567890",
		Description:    "Quality widgets",
		Status:         models.StatusApproved,
		CommissionRate: 10,
		TotalRevenue:   5000,
		TotalOrders:    50,
		TotalProducts:  20,
		Rating:         4.5,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// === RegisterVendor Tests ===

func TestRegisterVendor_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByEmail", ctx, "new@vendor.com").Return(nil, errors.New("vendor not found"))
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Vendor")).Return(nil)

	req := &models.RegisterVendorRequest{
		TenantID: "tenant-1",
		Name:     "New Vendor",
		Email:    "new@vendor.com",
		Phone:    "+1234567890",
		Country:  "US",
	}

	result, err := svc.RegisterVendor(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "New Vendor", result.Name)
	assert.Equal(t, models.StatusPending, result.Status)
	assert.Equal(t, 10.0, result.CommissionRate)
}

func TestRegisterVendor_CustomCommission(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByEmail", ctx, "new@vendor.com").Return(nil, errors.New("not found"))
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Vendor")).Return(nil)

	req := &models.RegisterVendorRequest{
		TenantID:       "tenant-1",
		Name:           "Premium Vendor",
		Email:          "new@vendor.com",
		CommissionRate: 5,
	}

	result, err := svc.RegisterVendor(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, 5.0, result.CommissionRate)
}

func TestRegisterVendor_DuplicateEmail(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	existing := createTestVendor()
	mockRepo.On("GetByEmail", ctx, "vendor@acme.com").Return(existing, nil)

	req := &models.RegisterVendorRequest{
		TenantID: "tenant-1",
		Name:     "Duplicate",
		Email:    "vendor@acme.com",
	}

	result, err := svc.RegisterVendor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRegisterVendor_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByEmail", ctx, "new@vendor.com").Return(nil, errors.New("not found"))
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Vendor")).Return(errors.New("db error"))

	req := &models.RegisterVendorRequest{
		TenantID: "tenant-1",
		Name:     "Vendor",
		Email:    "new@vendor.com",
	}

	result, err := svc.RegisterVendor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetVendor Tests ===

func TestGetVendor_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	vendor := createTestVendor()
	mockRepo.On("GetByID", ctx, "vendor-1").Return(vendor, nil)

	result, err := svc.GetVendor(ctx, "vendor-1")

	assert.NoError(t, err)
	assert.Equal(t, "Acme Corp", result.Name)
	assert.Equal(t, 4.5, result.Rating)
}

func TestGetVendor_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "bad").Return(nil, errors.New("vendor not found"))

	result, err := svc.GetVendor(ctx, "bad")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === ListVendors Tests ===

func TestListVendors_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	vendors := []models.Vendor{*createTestVendor()}
	mockRepo.On("List", ctx, "tenant-1", "", 1, 20).Return(vendors, int64(1), nil)

	results, total, err := svc.ListVendors(ctx, "tenant-1", "", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
}

func TestListVendors_WithStatusFilter(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("List", ctx, "tenant-1", "approved", 1, 20).Return([]models.Vendor{}, int64(0), nil)

	results, total, err := svc.ListVendors(ctx, "tenant-1", "approved", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, results, 0)
}

// === UpdateVendor Tests ===

func TestUpdateVendor_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	vendor := createTestVendor()
	mockRepo.On("GetByID", ctx, "vendor-1").Return(vendor, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Vendor")).Return(nil)

	req := &models.UpdateVendorRequest{Name: "Updated Name", City: "New York"}

	result, err := svc.UpdateVendor(ctx, "vendor-1", req)

	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", result.Name)
	assert.Equal(t, "New York", result.City)
}

func TestUpdateVendor_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "bad").Return(nil, errors.New("vendor not found"))

	req := &models.UpdateVendorRequest{Name: "Test"}

	result, err := svc.UpdateVendor(ctx, "bad", req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === UpdateVendorStatus Tests ===

func TestUpdateVendorStatus_Approve(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	vendor := createTestVendor()
	vendor.Status = models.StatusPending
	mockRepo.On("GetByID", ctx, "vendor-1").Return(vendor, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Vendor")).Return(nil)

	req := &models.UpdateVendorStatusRequest{Status: models.StatusApproved}

	result, err := svc.UpdateVendorStatus(ctx, "vendor-1", req)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusApproved, result.Status)
	assert.NotNil(t, result.ApprovedAt)
}

func TestUpdateVendorStatus_Suspend(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	vendor := createTestVendor()
	vendor.Status = models.StatusApproved
	mockRepo.On("GetByID", ctx, "vendor-1").Return(vendor, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Vendor")).Return(nil)

	req := &models.UpdateVendorStatusRequest{Status: models.StatusSuspended, Reason: "Policy violation"}

	result, err := svc.UpdateVendorStatus(ctx, "vendor-1", req)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusSuspended, result.Status)
}

func TestUpdateVendorStatus_InvalidTransition(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	vendor := createTestVendor()
	vendor.Status = models.StatusPending
	mockRepo.On("GetByID", ctx, "vendor-1").Return(vendor, nil)

	req := &models.UpdateVendorStatusRequest{Status: models.StatusSuspended}

	result, err := svc.UpdateVendorStatus(ctx, "vendor-1", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid status transition")
}

func TestUpdateVendorStatus_Reject(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	vendor := createTestVendor()
	vendor.Status = models.StatusPending
	mockRepo.On("GetByID", ctx, "vendor-1").Return(vendor, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Vendor")).Return(nil)

	req := &models.UpdateVendorStatusRequest{Status: models.StatusRejected, Reason: "Incomplete docs"}

	result, err := svc.UpdateVendorStatus(ctx, "vendor-1", req)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusRejected, result.Status)
}

func TestUpdateVendorStatus_Reactivate(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	vendor := createTestVendor()
	vendor.Status = models.StatusSuspended
	mockRepo.On("GetByID", ctx, "vendor-1").Return(vendor, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Vendor")).Return(nil)

	req := &models.UpdateVendorStatusRequest{Status: models.StatusApproved}

	result, err := svc.UpdateVendorStatus(ctx, "vendor-1", req)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusApproved, result.Status)
}

// === GetVendorOrders Tests ===

func TestGetVendorOrders_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	orders := []models.VendorOrder{
		{ID: "vo-1", VendorID: "vendor-1", OrderID: "order-1", Amount: 100, Commission: 10, NetAmount: 90},
	}
	mockRepo.On("GetOrdersByVendor", ctx, "vendor-1", 1, 20).Return(orders, int64(1), nil)

	results, total, err := svc.GetVendorOrders(ctx, "vendor-1", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, 90.0, results[0].NetAmount)
}

// === GetVendorAnalytics Tests ===

func TestGetVendorAnalytics_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	analytics := &models.VendorAnalyticsResponse{
		VendorID:       "vendor-1",
		TotalRevenue:   5000,
		TotalOrders:    50,
		TotalProducts:  20,
		CommissionPaid: 500,
		NetEarnings:    4500,
		Rating:         4.5,
	}
	mockRepo.On("GetVendorAnalytics", ctx, "vendor-1").Return(analytics, nil)

	result, err := svc.GetVendorAnalytics(ctx, "vendor-1")

	assert.NoError(t, err)
	assert.Equal(t, 5000.0, result.TotalRevenue)
	assert.Equal(t, 4500.0, result.NetEarnings)
}

func TestGetVendorAnalytics_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetVendorAnalytics", ctx, "bad").Return(nil, errors.New("vendor not found"))

	result, err := svc.GetVendorAnalytics(ctx, "bad")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === RecordOrder Tests ===

func TestRecordOrder_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	vendor := createTestVendor()
	mockRepo.On("GetByID", ctx, "vendor-1").Return(vendor, nil)
	mockRepo.On("CreateOrder", ctx, mock.AnythingOfType("*models.VendorOrder")).Return(nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Vendor")).Return(nil)

	err := svc.RecordOrder(ctx, "vendor-1", "tenant-1", "order-1", 200)

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "CreateOrder", ctx, mock.AnythingOfType("*models.VendorOrder"))
}

func TestRecordOrder_VendorNotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "bad").Return(nil, errors.New("vendor not found"))

	err := svc.RecordOrder(ctx, "bad", "tenant-1", "order-1", 200)

	assert.Error(t, err)
}

// === isValidStatusTransition Tests ===

func TestIsValidStatusTransition(t *testing.T) {
	assert.True(t, isValidStatusTransition(models.StatusPending, models.StatusApproved))
	assert.True(t, isValidStatusTransition(models.StatusPending, models.StatusRejected))
	assert.True(t, isValidStatusTransition(models.StatusApproved, models.StatusSuspended))
	assert.True(t, isValidStatusTransition(models.StatusSuspended, models.StatusApproved))
	assert.True(t, isValidStatusTransition(models.StatusRejected, models.StatusPending))

	assert.False(t, isValidStatusTransition(models.StatusPending, models.StatusSuspended))
	assert.False(t, isValidStatusTransition(models.StatusApproved, models.StatusRejected))
	assert.False(t, isValidStatusTransition(models.StatusRejected, models.StatusApproved))
}

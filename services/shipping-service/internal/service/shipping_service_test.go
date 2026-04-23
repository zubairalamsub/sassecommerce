package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/shipping-service/internal/models"
	repoMocks "github.com/ecommerce/shipping-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockCarrierService is an inline mock to avoid import cycles
type mockCarrierService struct {
	mock.Mock
}

func (m *mockCarrierService) CreateLabel(carrier, serviceType string, shipment *models.Shipment) (*LabelResult, error) {
	args := m.Called(carrier, serviceType, shipment)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LabelResult), args.Error(1)
}

func (m *mockCarrierService) CalculateRates(req *models.CalculateRateRequest) ([]models.CarrierRateResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.CarrierRateResponse), args.Error(1)
}

func (m *mockCarrierService) GetTrackingInfo(carrier, trackingNumber string) (*TrackingResult, error) {
	args := m.Called(carrier, trackingNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TrackingResult), args.Error(1)
}

func newTestService() (*shippingService, *repoMocks.MockShipmentRepository, *mockCarrierService) {
	mockRepo := new(repoMocks.MockShipmentRepository)
	mockCarrier := new(mockCarrierService)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &shippingService{
		repo:    mockRepo,
		carrier: mockCarrier,
		logger:  logger,
	}

	return svc, mockRepo, mockCarrier
}

func createTestShipmentRequest() *models.CreateShipmentRequest {
	return &models.CreateShipmentRequest{
		TenantID:    "tenant-1",
		OrderID:     "order-1",
		Carrier:     "pathao",
		ServiceType: "standard",
		WeightOz:    2.0,
		LengthIn:    30.0,
		WidthIn:     20.0,
		HeightIn:    15.0,
		FromAddress: models.AddressRequest{
			Name:       "Warehouse",
			Street:     "Plot 5, BSCIC Industrial Area",
			City:       "Gazipur",
			State:      "Dhaka",
			PostalCode: "1700",
			Country:    "BD",
		},
		ToAddress: models.AddressRequest{
			Name:       "Rahim Uddin",
			Street:     "House 12, Road 5, Dhanmondi",
			City:       "Dhaka",
			State:      "Dhaka",
			PostalCode: "1205",
			Country:    "BD",
		},
		Items: []models.ShipmentItemRequest{
			{
				ProductID: "prod-1",
				SKU:       "SKU-001",
				Name:      "Test Product",
				Quantity:  2,
				WeightOz:  16.0,
			},
		},
		InsuredValue:      100.00,
		SignatureRequired: true,
	}
}

func createTestShipment() *models.Shipment {
	now := time.Now().UTC()
	estimated := now.AddDate(0, 0, 7)
	return &models.Shipment{
		ID:                "shipment-1",
		TenantID:          "tenant-1",
		OrderID:           "order-1",
		Carrier:           "pathao",
		TrackingNumber:    "PA1234567890",
		ServiceType:       "standard",
		LabelURL:          "https://labels.simulated.dev/pathao/PA1234567890.pdf",
		Status:            models.StatusLabelCreated,
		WeightOz:          2.0,
		ShippingCost:      80.0,
		Currency:          "BDT",
		FromName:          "Warehouse",
		FromStreet:        "Plot 5, BSCIC Industrial Area",
		FromCity:          "Gazipur",
		FromState:         "Dhaka",
		FromPostalCode:    "1700",
		FromCountry:       "BD",
		ToName:            "Rahim Uddin",
		ToStreet:          "House 12, Road 5, Dhanmondi",
		ToCity:            "Dhaka",
		ToState:           "Dhaka",
		ToPostalCode:      "1205",
		ToCountry:         "BD",
		EstimatedDelivery: &estimated,
		SignatureRequired: true,
		InsuredValue:      100.00,
		CreatedAt:         now,
		UpdatedAt:         now,
		Items: []models.ShipmentItem{
			{
				ID:        "item-1",
				ProductID: "prod-1",
				SKU:       "SKU-001",
				Name:      "Test Product",
				Quantity:  2,
				WeightOz:  16.0,
			},
		},
		Events: []models.ShipmentEvent{
			{
				ID:          "event-1",
				ShipmentID:  "shipment-1",
				Status:      "label_created",
				Description: "Shipping label created",
				OccurredAt:  now,
			},
		},
	}
}

// === CreateShipment Tests ===

func TestCreateShipment_Success(t *testing.T) {
	svc, mockRepo, mockCarrier := newTestService()
	ctx := context.Background()
	req := createTestShipmentRequest()

	estimated := time.Now().UTC().AddDate(0, 0, 7)
	mockCarrier.On("CreateLabel", "pathao", "standard", mock.AnythingOfType("*models.Shipment")).Return(&LabelResult{
		TrackingNumber:    "PA1234567890",
		LabelURL:          "https://labels.simulated.dev/pathao/PA1234567890.pdf",
		ShippingCost:      11.99,
		EstimatedDelivery: estimated,
	}, nil)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Shipment")).Return(nil)
	mockRepo.On("CreateEvent", ctx, mock.AnythingOfType("*models.ShipmentEvent")).Return(nil)

	result, err := svc.CreateShipment(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "PA1234567890", result.TrackingNumber)
	assert.Equal(t, "pathao", result.Carrier)
	assert.Equal(t, models.StatusLabelCreated, result.Status)
	assert.Equal(t, 11.99, result.ShippingCost)
	assert.Equal(t, "Warehouse", result.FromAddress.Name)
	assert.Equal(t, "Rahim Uddin", result.ToAddress.Name)
	assert.True(t, result.SignatureRequired)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "prod-1", result.Items[0].ProductID)

	mockCarrier.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCreateShipment_DefaultServiceType(t *testing.T) {
	svc, mockRepo, mockCarrier := newTestService()
	ctx := context.Background()
	req := createTestShipmentRequest()
	req.ServiceType = "" // empty => defaults to "standard"

	estimated := time.Now().UTC().AddDate(0, 0, 7)
	mockCarrier.On("CreateLabel", "pathao", "standard", mock.AnythingOfType("*models.Shipment")).Return(&LabelResult{
		TrackingNumber:    "PA1234567890",
		LabelURL:          "https://labels.simulated.dev/pathao/PA1234567890.pdf",
		ShippingCost:      7.99,
		EstimatedDelivery: estimated,
	}, nil)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Shipment")).Return(nil)
	mockRepo.On("CreateEvent", ctx, mock.AnythingOfType("*models.ShipmentEvent")).Return(nil)

	result, err := svc.CreateShipment(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "standard", result.ServiceType)
}

func TestCreateShipment_CarrierFailure(t *testing.T) {
	svc, _, mockCarrier := newTestService()
	ctx := context.Background()
	req := createTestShipmentRequest()

	mockCarrier.On("CreateLabel", "pathao", "standard", mock.AnythingOfType("*models.Shipment")).
		Return(nil, errors.New("carrier service unavailable"))

	result, err := svc.CreateShipment(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create shipping label")
}

func TestCreateShipment_RepositoryFailure(t *testing.T) {
	svc, mockRepo, mockCarrier := newTestService()
	ctx := context.Background()
	req := createTestShipmentRequest()

	estimated := time.Now().UTC().AddDate(0, 0, 7)
	mockCarrier.On("CreateLabel", "pathao", "standard", mock.AnythingOfType("*models.Shipment")).Return(&LabelResult{
		TrackingNumber:    "PA1234567890",
		LabelURL:          "https://labels.simulated.dev/pathao/PA1234567890.pdf",
		ShippingCost:      7.99,
		EstimatedDelivery: estimated,
	}, nil)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Shipment")).Return(errors.New("database error"))

	result, err := svc.CreateShipment(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create shipment")
}

// === GetShipment Tests ===

func TestGetShipment_Success(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()
	shipment := createTestShipment()

	mockRepo.On("GetByIDWithDetails", ctx, "shipment-1").Return(shipment, nil)

	result, err := svc.GetShipment(ctx, "shipment-1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "shipment-1", result.ID)
	assert.Equal(t, "pathao", result.Carrier)
	assert.Len(t, result.Items, 1)
	assert.Len(t, result.Events, 1)
}

func TestGetShipment_NotFound(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByIDWithDetails", ctx, "nonexistent").Return(nil, errors.New("shipment not found"))

	result, err := svc.GetShipment(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetShipmentByTracking_Success(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()
	shipment := createTestShipment()

	mockRepo.On("GetByTrackingNumber", ctx, "PA1234567890").Return(shipment, nil)

	result, err := svc.GetShipmentByTracking(ctx, "PA1234567890")

	assert.NoError(t, err)
	assert.Equal(t, "PA1234567890", result.TrackingNumber)
}

func TestGetShipmentByTracking_NotFound(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByTrackingNumber", ctx, "INVALID").Return(nil, errors.New("shipment not found"))

	result, err := svc.GetShipmentByTracking(ctx, "INVALID")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetShipmentByOrderID_Success(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()
	shipment := createTestShipment()

	mockRepo.On("GetByOrderID", ctx, "tenant-1", "order-1").Return(shipment, nil)

	result, err := svc.GetShipmentByOrderID(ctx, "tenant-1", "order-1")

	assert.NoError(t, err)
	assert.Equal(t, "order-1", result.OrderID)
}

func TestGetShipmentByOrderID_NotFound(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByOrderID", ctx, "tenant-1", "bad-order").Return(nil, errors.New("shipment not found"))

	result, err := svc.GetShipmentByOrderID(ctx, "tenant-1", "bad-order")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === ListShipments Tests ===

func TestListShipments_Success(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	shipments := []models.Shipment{*createTestShipment()}
	mockRepo.On("List", ctx, "tenant-1", 1, 20, "").Return(shipments, int64(1), nil)

	results, total, err := svc.ListShipments(ctx, "tenant-1", 1, 20, "")

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
}

func TestListShipments_WithStatusFilter(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	shipments := []models.Shipment{*createTestShipment()}
	mockRepo.On("List", ctx, "tenant-1", 1, 20, "in_transit").Return(shipments, int64(1), nil)

	results, total, err := svc.ListShipments(ctx, "tenant-1", 1, 20, "in_transit")

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
}

func TestListShipments_Empty(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("List", ctx, "tenant-1", 1, 20, "").Return([]models.Shipment{}, int64(0), nil)

	results, total, err := svc.ListShipments(ctx, "tenant-1", 1, 20, "")

	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, results, 0)
}

// === UpdateStatus Tests ===

func TestUpdateStatus_PickedUp(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	shipment := createTestShipment()
	shipment.Status = models.StatusLabelCreated

	mockRepo.On("GetByIDWithDetails", ctx, "shipment-1").Return(shipment, nil).Once()
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Shipment")).Return(nil)
	mockRepo.On("CreateEvent", ctx, mock.AnythingOfType("*models.ShipmentEvent")).Return(nil)
	mockRepo.On("GetByIDWithDetails", ctx, "shipment-1").Return(shipment, nil)

	req := &models.UpdateStatusRequest{
		Status:      "picked_up",
		Location:    "Gazipur, Dhaka",
		Description: "Package picked up by carrier",
	}

	result, err := svc.UpdateStatus(ctx, "shipment-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestUpdateStatus_Delivered(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	shipment := createTestShipment()
	shipment.Status = models.StatusOutForDelivery

	mockRepo.On("GetByIDWithDetails", ctx, "shipment-1").Return(shipment, nil).Once()
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Shipment")).Return(nil)
	mockRepo.On("CreateEvent", ctx, mock.AnythingOfType("*models.ShipmentEvent")).Return(nil)
	mockRepo.On("GetByIDWithDetails", ctx, "shipment-1").Return(shipment, nil)

	req := &models.UpdateStatusRequest{
		Status:      "delivered",
		Location:    "New York, NY",
		Description: "Package delivered",
		SignedBy:    "John Doe",
	}

	result, err := svc.UpdateStatus(ctx, "shipment-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "John Doe", shipment.SignedBy)
	assert.NotNil(t, shipment.DeliveredAt)
}

func TestUpdateStatus_InvalidTransition(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	shipment := createTestShipment()
	shipment.Status = models.StatusPending

	mockRepo.On("GetByIDWithDetails", ctx, "shipment-1").Return(shipment, nil)

	req := &models.UpdateStatusRequest{
		Status: "delivered",
	}

	result, err := svc.UpdateStatus(ctx, "shipment-1", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid status transition")
}

func TestUpdateStatus_NotFound(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByIDWithDetails", ctx, "nonexistent").Return(nil, errors.New("shipment not found"))

	req := &models.UpdateStatusRequest{Status: "in_transit"}

	result, err := svc.UpdateStatus(ctx, "nonexistent", req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUpdateStatus_Failed(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	shipment := createTestShipment()
	shipment.Status = models.StatusInTransit

	mockRepo.On("GetByIDWithDetails", ctx, "shipment-1").Return(shipment, nil).Once()
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Shipment")).Return(nil)
	mockRepo.On("CreateEvent", ctx, mock.AnythingOfType("*models.ShipmentEvent")).Return(nil)
	mockRepo.On("GetByIDWithDetails", ctx, "shipment-1").Return(shipment, nil)

	req := &models.UpdateStatusRequest{
		Status:      "failed",
		Description: "Delivery attempt failed - no one home",
	}

	result, err := svc.UpdateStatus(ctx, "shipment-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Delivery attempt failed - no one home", shipment.FailureReason)
}

// === CancelShipment Tests ===

func TestCancelShipment_Pending(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	shipment := createTestShipment()
	shipment.Status = models.StatusPending

	mockRepo.On("GetByID", ctx, "shipment-1").Return(shipment, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Shipment")).Return(nil)
	mockRepo.On("CreateEvent", ctx, mock.AnythingOfType("*models.ShipmentEvent")).Return(nil)

	result, err := svc.CancelShipment(ctx, "shipment-1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusCancelled, result.Status)
}

func TestCancelShipment_LabelCreated(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	shipment := createTestShipment()
	shipment.Status = models.StatusLabelCreated

	mockRepo.On("GetByID", ctx, "shipment-1").Return(shipment, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Shipment")).Return(nil)
	mockRepo.On("CreateEvent", ctx, mock.AnythingOfType("*models.ShipmentEvent")).Return(nil)

	result, err := svc.CancelShipment(ctx, "shipment-1")

	assert.NoError(t, err)
	assert.Equal(t, models.StatusCancelled, result.Status)
}

func TestCancelShipment_AlreadyInTransit(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	shipment := createTestShipment()
	shipment.Status = models.StatusInTransit

	mockRepo.On("GetByID", ctx, "shipment-1").Return(shipment, nil)

	result, err := svc.CancelShipment(ctx, "shipment-1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "can only be cancelled when pending or label_created")
}

func TestCancelShipment_NotFound(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.New("shipment not found"))

	result, err := svc.CancelShipment(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === CalculateRates Tests ===

func TestCalculateRates_Success(t *testing.T) {
	svc, _, mockCarrier := newTestService()
	ctx := context.Background()

	req := &models.CalculateRateRequest{
		TenantID: "tenant-1",
		FromAddress: models.AddressRequest{
			Name:       "Warehouse",
			Street:     "Plot 5, BSCIC Industrial Area",
			City:       "Gazipur",
			State:      "Dhaka",
			PostalCode: "1700",
			Country:    "BD",
		},
		ToAddress: models.AddressRequest{
			Name:       "Customer",
			Street:     "House 12, Road 5, Dhanmondi",
			City:       "Dhaka",
			State:      "Dhaka",
			PostalCode: "1205",
			Country:    "BD",
		},
		WeightOz: 1.0,
	}

	rates := []models.CarrierRateResponse{
		{Carrier: "pathao", ServiceType: "standard", Rate: 60.0, Currency: "BDT", EstimatedDays: 5},
		{Carrier: "pathao", ServiceType: "express", Rate: 108.0, Currency: "BDT", EstimatedDays: 2},
		{Carrier: "steadfast", ServiceType: "standard", Rate: 70.0, Currency: "BDT", EstimatedDays: 5},
	}

	mockCarrier.On("CalculateRates", req).Return(rates, nil)

	result, err := svc.CalculateRates(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Rates, 3)
	assert.Equal(t, "pathao", result.Rates[0].Carrier)
}

func TestCalculateRates_CarrierFailure(t *testing.T) {
	svc, _, mockCarrier := newTestService()
	ctx := context.Background()

	req := &models.CalculateRateRequest{
		TenantID: "tenant-1",
		FromAddress: models.AddressRequest{
			Name: "Warehouse", Street: "BSCIC", City: "Gazipur", State: "Dhaka", PostalCode: "1700", Country: "BD",
		},
		ToAddress: models.AddressRequest{
			Name: "Customer", Street: "Dhanmondi", City: "Dhaka", State: "Dhaka", PostalCode: "1209", Country: "BD",
		},
		WeightOz: 1.0,
	}

	mockCarrier.On("CalculateRates", req).Return(nil, errors.New("rate service unavailable"))

	result, err := svc.CalculateRates(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === Status Transition Validation Tests ===

func TestValidateStatusTransition_ValidTransitions(t *testing.T) {
	tests := []struct {
		from models.ShipmentStatus
		to   models.ShipmentStatus
	}{
		{models.StatusPending, models.StatusLabelCreated},
		{models.StatusPending, models.StatusCancelled},
		{models.StatusLabelCreated, models.StatusPickedUp},
		{models.StatusLabelCreated, models.StatusCancelled},
		{models.StatusPickedUp, models.StatusInTransit},
		{models.StatusPickedUp, models.StatusFailed},
		{models.StatusInTransit, models.StatusOutForDelivery},
		{models.StatusInTransit, models.StatusFailed},
		{models.StatusInTransit, models.StatusReturned},
		{models.StatusOutForDelivery, models.StatusDelivered},
		{models.StatusOutForDelivery, models.StatusFailed},
		{models.StatusOutForDelivery, models.StatusReturned},
		{models.StatusDelivered, models.StatusReturned},
		{models.StatusFailed, models.StatusInTransit},
		{models.StatusFailed, models.StatusReturned},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			err := validateStatusTransition(tt.from, tt.to)
			assert.NoError(t, err)
		})
	}
}

func TestValidateStatusTransition_InvalidTransitions(t *testing.T) {
	tests := []struct {
		from models.ShipmentStatus
		to   models.ShipmentStatus
	}{
		{models.StatusPending, models.StatusDelivered},
		{models.StatusPending, models.StatusInTransit},
		{models.StatusLabelCreated, models.StatusDelivered},
		{models.StatusDelivered, models.StatusInTransit},
		{models.StatusCancelled, models.StatusPending},
		{models.StatusCancelled, models.StatusInTransit},
		{models.StatusReturned, models.StatusDelivered},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			err := validateStatusTransition(tt.from, tt.to)
			assert.Error(t, err)
		})
	}
}

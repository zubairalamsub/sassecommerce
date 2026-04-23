package mocks

import (
	"context"

	"github.com/ecommerce/shipping-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockShipmentRepository struct {
	mock.Mock
}

func (m *MockShipmentRepository) Create(ctx context.Context, shipment *models.Shipment) error {
	args := m.Called(ctx, shipment)
	return args.Error(0)
}

func (m *MockShipmentRepository) GetByID(ctx context.Context, id string) (*models.Shipment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Shipment), args.Error(1)
}

func (m *MockShipmentRepository) GetByIDWithDetails(ctx context.Context, id string) (*models.Shipment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Shipment), args.Error(1)
}

func (m *MockShipmentRepository) GetByTrackingNumber(ctx context.Context, trackingNumber string) (*models.Shipment, error) {
	args := m.Called(ctx, trackingNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Shipment), args.Error(1)
}

func (m *MockShipmentRepository) GetByOrderID(ctx context.Context, tenantID, orderID string) (*models.Shipment, error) {
	args := m.Called(ctx, tenantID, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Shipment), args.Error(1)
}

func (m *MockShipmentRepository) List(ctx context.Context, tenantID string, page, pageSize int, status string) ([]models.Shipment, int64, error) {
	args := m.Called(ctx, tenantID, page, pageSize, status)
	return args.Get(0).([]models.Shipment), args.Get(1).(int64), args.Error(2)
}

func (m *MockShipmentRepository) Update(ctx context.Context, shipment *models.Shipment) error {
	args := m.Called(ctx, shipment)
	return args.Error(0)
}

func (m *MockShipmentRepository) CreateEvent(ctx context.Context, event *models.ShipmentEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockShipmentRepository) GetEvents(ctx context.Context, shipmentID string) ([]models.ShipmentEvent, error) {
	args := m.Called(ctx, shipmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ShipmentEvent), args.Error(1)
}

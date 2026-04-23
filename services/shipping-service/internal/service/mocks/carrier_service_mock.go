package mocks

import (
	"github.com/ecommerce/shipping-service/internal/models"
	"github.com/ecommerce/shipping-service/internal/service"
	"github.com/stretchr/testify/mock"
)

type MockCarrierService struct {
	mock.Mock
}

func (m *MockCarrierService) CreateLabel(carrier, serviceType string, shipment *models.Shipment) (*service.LabelResult, error) {
	args := m.Called(carrier, serviceType, shipment)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.LabelResult), args.Error(1)
}

func (m *MockCarrierService) CalculateRates(req *models.CalculateRateRequest) ([]models.CarrierRateResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.CarrierRateResponse), args.Error(1)
}

func (m *MockCarrierService) GetTrackingInfo(carrier, trackingNumber string) (*service.TrackingResult, error) {
	args := m.Called(carrier, trackingNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TrackingResult), args.Error(1)
}

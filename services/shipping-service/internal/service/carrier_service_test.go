package service

import (
	"testing"

	"github.com/ecommerce/shipping-service/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestSimulatedCarrier_CreateLabel_Success(t *testing.T) {
	carrier := NewSimulatedCarrierService()

	shipment := &models.Shipment{
		Carrier:     "pathao",
		ServiceType: "standard",
		WeightOz:    2.0,
	}

	result, err := carrier.CreateLabel("pathao", "standard", shipment)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.TrackingNumber)
	assert.Contains(t, result.TrackingNumber, "PA")
	assert.Contains(t, result.LabelURL, "pathao")
	assert.Greater(t, result.ShippingCost, 0.0)
	assert.False(t, result.EstimatedDelivery.IsZero())
}

func TestSimulatedCarrier_CreateLabel_EmptyCarrier(t *testing.T) {
	carrier := NewSimulatedCarrierService()

	shipment := &models.Shipment{WeightOz: 1.0}

	result, err := carrier.CreateLabel("", "standard", shipment)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "carrier is required")
}

func TestSimulatedCarrier_CreateLabel_DifferentCarriers(t *testing.T) {
	carrier := NewSimulatedCarrierService()
	carriers := []string{"pathao", "steadfast", "redx", "paperfly", "sundarban"}

	for _, c := range carriers {
		shipment := &models.Shipment{Carrier: c, WeightOz: 1.0}
		result, err := carrier.CreateLabel(c, "standard", shipment)

		assert.NoError(t, err, "failed for carrier: %s", c)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.TrackingNumber)
	}
}

func TestSimulatedCarrier_CreateLabel_ServiceTypes(t *testing.T) {
	carrier := NewSimulatedCarrierService()

	tests := []struct {
		serviceType string
		minCost     float64
	}{
		{"standard", 50.0},
		{"express", 90.0},
	}

	for _, tt := range tests {
		shipment := &models.Shipment{Carrier: "pathao", WeightOz: 1.0}
		result, err := carrier.CreateLabel("pathao", tt.serviceType, shipment)

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, result.ShippingCost, tt.minCost, "cost too low for %s", tt.serviceType)
	}
}

func TestSimulatedCarrier_CreateLabel_WeightSurcharge(t *testing.T) {
	carrier := NewSimulatedCarrierService()

	light := &models.Shipment{Carrier: "paperfly", WeightOz: 0.5}
	heavy := &models.Shipment{Carrier: "paperfly", WeightOz: 5.0}

	lightResult, _ := carrier.CreateLabel("paperfly", "standard", light)
	heavyResult, _ := carrier.CreateLabel("paperfly", "standard", heavy)

	assert.Greater(t, heavyResult.ShippingCost, lightResult.ShippingCost)
}

func TestSimulatedCarrier_CalculateRates(t *testing.T) {
	carrier := NewSimulatedCarrierService()

	req := &models.CalculateRateRequest{
		TenantID: "tenant-1",
		FromAddress: models.AddressRequest{
			Name: "Warehouse", Street: "BSCIC, Gazipur", City: "Gazipur", State: "Dhaka", PostalCode: "1700", Country: "BD",
		},
		ToAddress: models.AddressRequest{
			Name: "Customer", Street: "Dhanmondi 27", City: "Dhaka", State: "Dhaka", PostalCode: "1209", Country: "BD",
		},
		WeightOz: 1.0,
	}

	rates, err := carrier.CalculateRates(req)

	assert.NoError(t, err)
	assert.Len(t, rates, 10) // 5 carriers * 2 service types

	for _, rate := range rates {
		assert.NotEmpty(t, rate.Carrier)
		assert.NotEmpty(t, rate.ServiceType)
		assert.Greater(t, rate.Rate, 0.0)
		assert.Equal(t, "BDT", rate.Currency)
		assert.Greater(t, rate.EstimatedDays, 0)
	}
}

func TestSimulatedCarrier_GetTrackingInfo_Success(t *testing.T) {
	carrier := NewSimulatedCarrierService()

	result, err := carrier.GetTrackingInfo("pathao", "PA1234567890")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "in_transit", result.Status)
	assert.NotEmpty(t, result.Location)
	assert.NotEmpty(t, result.Description)
}

func TestSimulatedCarrier_GetTrackingInfo_EmptyTracking(t *testing.T) {
	carrier := NewSimulatedCarrierService()

	result, err := carrier.GetTrackingInfo("pathao", "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "tracking number is required")
}

func TestSimulatedCarrier_UniqueTrackingNumbers(t *testing.T) {
	carrier := NewSimulatedCarrierService()
	seen := make(map[string]bool)

	for i := 0; i < 10; i++ {
		shipment := &models.Shipment{Carrier: "pathao", WeightOz: 1.0}
		result, err := carrier.CreateLabel("pathao", "standard", shipment)
		assert.NoError(t, err)
		assert.False(t, seen[result.TrackingNumber], "duplicate tracking number: %s", result.TrackingNumber)
		seen[result.TrackingNumber] = true
	}
}

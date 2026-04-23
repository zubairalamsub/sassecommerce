package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ecommerce/shipping-service/internal/models"
)

// CarrierService abstracts carrier operations (label creation, rate calculation, tracking)
type CarrierService interface {
	CreateLabel(carrier, serviceType string, shipment *models.Shipment) (*LabelResult, error)
	CalculateRates(req *models.CalculateRateRequest) ([]models.CarrierRateResponse, error)
	GetTrackingInfo(carrier, trackingNumber string) (*TrackingResult, error)
}

type LabelResult struct {
	TrackingNumber    string
	LabelURL          string
	ShippingCost      float64
	EstimatedDelivery time.Time
}

type TrackingResult struct {
	Status      string
	Location    string
	Description string
	UpdatedAt   time.Time
}

// SimulatedCarrierService provides a simulated carrier for development/testing
type SimulatedCarrierService struct{}

func NewSimulatedCarrierService() CarrierService {
	return &SimulatedCarrierService{}
}

func (s *SimulatedCarrierService) CreateLabel(carrier, serviceType string, shipment *models.Shipment) (*LabelResult, error) {
	if carrier == "" {
		return nil, fmt.Errorf("carrier is required")
	}

	trackingNumber := s.generateTrackingNumber(carrier)
	cost := s.calculateCost(carrier, serviceType, shipment.WeightOz)
	estimatedDays := s.getEstimatedDays(serviceType)

	return &LabelResult{
		TrackingNumber:    trackingNumber,
		LabelURL:          fmt.Sprintf("https://labels.simulated.dev/%s/%s.pdf", carrier, trackingNumber),
		ShippingCost:      cost,
		EstimatedDelivery: time.Now().UTC().AddDate(0, 0, estimatedDays),
	}, nil
}

func (s *SimulatedCarrierService) CalculateRates(req *models.CalculateRateRequest) ([]models.CarrierRateResponse, error) {
	carriers := []string{models.CarrierPathao, models.CarrierSteadfast, models.CarrierRedX, models.CarrierPaperfly, models.CarrierSundarban}
	serviceTypes := []struct {
		name string
		days int
	}{
		{"standard", 5},
		{"express", 2},
	}

	var rates []models.CarrierRateResponse
	for _, carrier := range carriers {
		for _, st := range serviceTypes {
			cost := s.calculateCost(carrier, st.name, req.WeightOz)
			rates = append(rates, models.CarrierRateResponse{
				Carrier:       carrier,
				ServiceType:   st.name,
				Rate:          cost,
				Currency:      "BDT",
				EstimatedDays: st.days,
			})
		}
	}

	return rates, nil
}

func (s *SimulatedCarrierService) GetTrackingInfo(carrier, trackingNumber string) (*TrackingResult, error) {
	if trackingNumber == "" {
		return nil, fmt.Errorf("tracking number is required")
	}

	return &TrackingResult{
		Status:      "in_transit",
		Location:    "Sorting Hub, Gazipur, Dhaka",
		Description: "Package is in transit to destination",
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

func (s *SimulatedCarrierService) generateTrackingNumber(carrier string) string {
	n, _ := rand.Int(rand.Reader, big.NewInt(9999999999))
	prefix := strings.ToUpper(carrier[:2])
	return fmt.Sprintf("%s%010d", prefix, n.Int64())
}

func (s *SimulatedCarrierService) calculateCost(carrier, serviceType string, weightKg float64) float64 {
	// Base rates in BDT
	baseRate := 60.0
	switch carrier {
	case models.CarrierPathao:
		baseRate = 60.0
	case models.CarrierSteadfast:
		baseRate = 70.0
	case models.CarrierRedX:
		baseRate = 65.0
	case models.CarrierPaperfly:
		baseRate = 55.0
	case models.CarrierSundarban:
		baseRate = 80.0
	case models.CarrierSAParibahan:
		baseRate = 70.0
	}

	multiplier := 1.0
	switch serviceType {
	case "express":
		multiplier = 1.8
	}

	// Weight surcharge: BDT 20 per kg above 1 kg
	weightSurcharge := 0.0
	if weightKg > 1 {
		weightSurcharge = (weightKg - 1) * 20
	}

	cost := baseRate*multiplier + weightSurcharge
	return float64(int(cost*100)) / 100
}

func (s *SimulatedCarrierService) getEstimatedDays(serviceType string) int {
	switch serviceType {
	case "express":
		return 2
	default:
		return 5
	}
}

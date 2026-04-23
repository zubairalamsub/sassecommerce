package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ecommerce/shipping-service/internal/models"
	"github.com/ecommerce/shipping-service/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ShippingService interface {
	CreateShipment(ctx context.Context, req *models.CreateShipmentRequest) (*models.ShipmentResponse, error)
	GetShipment(ctx context.Context, id string) (*models.ShipmentResponse, error)
	GetShipmentByTracking(ctx context.Context, trackingNumber string) (*models.ShipmentResponse, error)
	GetShipmentByOrderID(ctx context.Context, tenantID, orderID string) (*models.ShipmentResponse, error)
	ListShipments(ctx context.Context, tenantID string, page, pageSize int, status string) ([]models.ShipmentResponse, int64, error)
	UpdateStatus(ctx context.Context, id string, req *models.UpdateStatusRequest) (*models.ShipmentResponse, error)
	CancelShipment(ctx context.Context, id string) (*models.ShipmentResponse, error)
	CalculateRates(ctx context.Context, req *models.CalculateRateRequest) (*models.RateCalculationResponse, error)
}

type shippingService struct {
	repo    repository.ShipmentRepository
	carrier CarrierService
	logger  *logrus.Logger
}

func NewShippingService(repo repository.ShipmentRepository, carrier CarrierService, logger *logrus.Logger) ShippingService {
	return &shippingService{
		repo:    repo,
		carrier: carrier,
		logger:  logger,
	}
}

func (s *shippingService) CreateShipment(ctx context.Context, req *models.CreateShipmentRequest) (*models.ShipmentResponse, error) {
	shipment := &models.Shipment{
		ID:                uuid.New().String(),
		TenantID:          req.TenantID,
		OrderID:           req.OrderID,
		Carrier:           req.Carrier,
		ServiceType:       req.ServiceType,
		Status:            models.StatusPending,
		WeightOz:          req.WeightOz,
		LengthIn:          req.LengthIn,
		WidthIn:           req.WidthIn,
		HeightIn:          req.HeightIn,
		InsuredValue:      req.InsuredValue,
		SignatureRequired: req.SignatureRequired,
		Currency:          "BDT",
		FromName:          req.FromAddress.Name,
		FromStreet:        req.FromAddress.Street,
		FromCity:          req.FromAddress.City,
		FromState:         req.FromAddress.State,
		FromPostalCode:    req.FromAddress.PostalCode,
		FromCountry:       req.FromAddress.Country,
		ToName:            req.ToAddress.Name,
		ToStreet:          req.ToAddress.Street,
		ToCity:            req.ToAddress.City,
		ToState:           req.ToAddress.State,
		ToPostalCode:      req.ToAddress.PostalCode,
		ToCountry:         req.ToAddress.Country,
	}

	// Build items
	for _, itemReq := range req.Items {
		shipment.Items = append(shipment.Items, models.ShipmentItem{
			ID:        uuid.New().String(),
			ProductID: itemReq.ProductID,
			VariantID: itemReq.VariantID,
			SKU:       itemReq.SKU,
			Name:      itemReq.Name,
			Quantity:  itemReq.Quantity,
			WeightOz:  itemReq.WeightOz,
		})
	}

	// Create shipping label via carrier
	if req.ServiceType == "" {
		shipment.ServiceType = "standard"
	}

	labelResult, err := s.carrier.CreateLabel(req.Carrier, shipment.ServiceType, shipment)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create shipping label")
		return nil, fmt.Errorf("failed to create shipping label: %w", err)
	}

	shipment.TrackingNumber = labelResult.TrackingNumber
	shipment.LabelURL = labelResult.LabelURL
	shipment.ShippingCost = labelResult.ShippingCost
	shipment.EstimatedDelivery = &labelResult.EstimatedDelivery
	shipment.Status = models.StatusLabelCreated

	// Save shipment
	if err := s.repo.Create(ctx, shipment); err != nil {
		s.logger.WithError(err).Error("Failed to create shipment")
		return nil, fmt.Errorf("failed to create shipment: %w", err)
	}

	// Record label created event
	now := time.Now().UTC()
	event := &models.ShipmentEvent{
		ID:          uuid.New().String(),
		ShipmentID:  shipment.ID,
		Status:      string(models.StatusLabelCreated),
		Description: fmt.Sprintf("Shipping label created with %s (%s)", shipment.Carrier, shipment.ServiceType),
		OccurredAt:  now,
	}
	if err := s.repo.CreateEvent(ctx, event); err != nil {
		s.logger.WithError(err).Warn("Failed to create shipment event")
	}

	s.logger.WithFields(logrus.Fields{
		"shipment_id":     shipment.ID,
		"tracking_number": shipment.TrackingNumber,
		"carrier":         shipment.Carrier,
	}).Info("Shipment created successfully")

	return toShipmentResponse(shipment), nil
}

func (s *shippingService) GetShipment(ctx context.Context, id string) (*models.ShipmentResponse, error) {
	shipment, err := s.repo.GetByIDWithDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	return toShipmentResponse(shipment), nil
}

func (s *shippingService) GetShipmentByTracking(ctx context.Context, trackingNumber string) (*models.ShipmentResponse, error) {
	shipment, err := s.repo.GetByTrackingNumber(ctx, trackingNumber)
	if err != nil {
		return nil, err
	}
	return toShipmentResponse(shipment), nil
}

func (s *shippingService) GetShipmentByOrderID(ctx context.Context, tenantID, orderID string) (*models.ShipmentResponse, error) {
	shipment, err := s.repo.GetByOrderID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	return toShipmentResponse(shipment), nil
}

func (s *shippingService) ListShipments(ctx context.Context, tenantID string, page, pageSize int, status string) ([]models.ShipmentResponse, int64, error) {
	shipments, total, err := s.repo.List(ctx, tenantID, page, pageSize, status)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.ShipmentResponse, len(shipments))
	for i, shipment := range shipments {
		responses[i] = *toShipmentResponse(&shipment)
	}

	return responses, total, nil
}

func (s *shippingService) UpdateStatus(ctx context.Context, id string, req *models.UpdateStatusRequest) (*models.ShipmentResponse, error) {
	shipment, err := s.repo.GetByIDWithDetails(ctx, id)
	if err != nil {
		return nil, err
	}

	newStatus := models.ShipmentStatus(req.Status)

	// Validate status transition
	if err := validateStatusTransition(shipment.Status, newStatus); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	shipment.Status = newStatus

	switch newStatus {
	case models.StatusPickedUp:
		shipment.ShippedAt = &now
	case models.StatusDelivered:
		shipment.DeliveredAt = &now
		shipment.ActualDelivery = &now
		if req.SignedBy != "" {
			shipment.SignedBy = req.SignedBy
		}
	case models.StatusFailed:
		if req.Description != "" {
			shipment.FailureReason = req.Description
		}
	}

	if err := s.repo.Update(ctx, shipment); err != nil {
		s.logger.WithError(err).Error("Failed to update shipment status")
		return nil, fmt.Errorf("failed to update shipment: %w", err)
	}

	// Record status change event
	event := &models.ShipmentEvent{
		ID:          uuid.New().String(),
		ShipmentID:  shipment.ID,
		Status:      req.Status,
		Location:    req.Location,
		Description: req.Description,
		OccurredAt:  now,
	}
	if err := s.repo.CreateEvent(ctx, event); err != nil {
		s.logger.WithError(err).Warn("Failed to create shipment event")
	}

	s.logger.WithFields(logrus.Fields{
		"shipment_id": shipment.ID,
		"new_status":  req.Status,
	}).Info("Shipment status updated")

	// Reload with details
	updated, err := s.repo.GetByIDWithDetails(ctx, id)
	if err != nil {
		return toShipmentResponse(shipment), nil
	}
	return toShipmentResponse(updated), nil
}

func (s *shippingService) CancelShipment(ctx context.Context, id string) (*models.ShipmentResponse, error) {
	shipment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Can only cancel if pending or label_created
	if shipment.Status != models.StatusPending && shipment.Status != models.StatusLabelCreated {
		return nil, errors.New("shipment can only be cancelled when pending or label_created")
	}

	now := time.Now().UTC()
	shipment.Status = models.StatusCancelled

	if err := s.repo.Update(ctx, shipment); err != nil {
		s.logger.WithError(err).Error("Failed to cancel shipment")
		return nil, fmt.Errorf("failed to cancel shipment: %w", err)
	}

	// Record cancellation event
	event := &models.ShipmentEvent{
		ID:          uuid.New().String(),
		ShipmentID:  shipment.ID,
		Status:      string(models.StatusCancelled),
		Description: "Shipment cancelled",
		OccurredAt:  now,
	}
	if err := s.repo.CreateEvent(ctx, event); err != nil {
		s.logger.WithError(err).Warn("Failed to create cancellation event")
	}

	s.logger.WithField("shipment_id", shipment.ID).Info("Shipment cancelled")

	return toShipmentResponse(shipment), nil
}

func (s *shippingService) CalculateRates(ctx context.Context, req *models.CalculateRateRequest) (*models.RateCalculationResponse, error) {
	rates, err := s.carrier.CalculateRates(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate rates")
		return nil, fmt.Errorf("failed to calculate rates: %w", err)
	}

	return &models.RateCalculationResponse{Rates: rates}, nil
}

// validateStatusTransition ensures only valid status transitions are allowed
func validateStatusTransition(current, next models.ShipmentStatus) error {
	validTransitions := map[models.ShipmentStatus][]models.ShipmentStatus{
		models.StatusPending:        {models.StatusLabelCreated, models.StatusCancelled},
		models.StatusLabelCreated:   {models.StatusPickedUp, models.StatusCancelled},
		models.StatusPickedUp:       {models.StatusInTransit, models.StatusFailed},
		models.StatusInTransit:      {models.StatusOutForDelivery, models.StatusFailed, models.StatusReturned},
		models.StatusOutForDelivery: {models.StatusDelivered, models.StatusFailed, models.StatusReturned},
		models.StatusDelivered:      {models.StatusReturned},
		models.StatusFailed:         {models.StatusInTransit, models.StatusReturned},
	}

	allowed, ok := validTransitions[current]
	if !ok {
		return fmt.Errorf("no transitions allowed from status %s", current)
	}

	for _, s := range allowed {
		if s == next {
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", current, next)
}

func toShipmentResponse(s *models.Shipment) *models.ShipmentResponse {
	resp := &models.ShipmentResponse{
		ID:             s.ID,
		TenantID:       s.TenantID,
		OrderID:        s.OrderID,
		Carrier:        s.Carrier,
		TrackingNumber: s.TrackingNumber,
		ServiceType:    s.ServiceType,
		LabelURL:       s.LabelURL,
		Status:         s.Status,
		FailureReason:  s.FailureReason,
		WeightOz:       s.WeightOz,
		ShippingCost:   s.ShippingCost,
		Currency:       s.Currency,
		FromAddress: models.AddressResponse{
			Name:       s.FromName,
			Street:     s.FromStreet,
			City:       s.FromCity,
			State:      s.FromState,
			PostalCode: s.FromPostalCode,
			Country:    s.FromCountry,
		},
		ToAddress: models.AddressResponse{
			Name:       s.ToName,
			Street:     s.ToStreet,
			City:       s.ToCity,
			State:      s.ToState,
			PostalCode: s.ToPostalCode,
			Country:    s.ToCountry,
		},
		EstimatedDelivery: s.EstimatedDelivery,
		ActualDelivery:    s.ActualDelivery,
		ShippedAt:         s.ShippedAt,
		DeliveredAt:       s.DeliveredAt,
		SignatureRequired: s.SignatureRequired,
		SignedBy:          s.SignedBy,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}

	for _, item := range s.Items {
		resp.Items = append(resp.Items, models.ShipmentItemResponse{
			ID:        item.ID,
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			SKU:       item.SKU,
			Name:      item.Name,
			Quantity:  item.Quantity,
			WeightOz:  item.WeightOz,
		})
	}

	for _, event := range s.Events {
		resp.Events = append(resp.Events, models.ShipmentEventResponse{
			ID:          event.ID,
			Status:      event.Status,
			Location:    event.Location,
			Description: event.Description,
			OccurredAt:  event.OccurredAt,
		})
	}

	return resp
}

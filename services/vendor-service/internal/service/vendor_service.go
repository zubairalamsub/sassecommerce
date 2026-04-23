package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ecommerce/vendor-service/internal/models"
	"github.com/ecommerce/vendor-service/internal/repository"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// VendorService defines the interface for vendor business logic
type VendorService interface {
	RegisterVendor(ctx context.Context, req *models.RegisterVendorRequest) (*models.VendorResponse, error)
	GetVendor(ctx context.Context, id string) (*models.VendorResponse, error)
	ListVendors(ctx context.Context, tenantID, status string, page, pageSize int) ([]models.VendorResponse, int64, error)
	UpdateVendor(ctx context.Context, id string, req *models.UpdateVendorRequest) (*models.VendorResponse, error)
	UpdateVendorStatus(ctx context.Context, id string, req *models.UpdateVendorStatusRequest) (*models.VendorResponse, error)
	GetVendorOrders(ctx context.Context, vendorID string, page, pageSize int) ([]models.VendorOrderResponse, int64, error)
	GetVendorAnalytics(ctx context.Context, vendorID string) (*models.VendorAnalyticsResponse, error)
	RecordOrder(ctx context.Context, vendorID, tenantID, orderID string, amount float64) error
}

type vendorService struct {
	repo   repository.VendorRepository
	writer *kafka.Writer
	logger *logrus.Logger
}

func NewVendorService(repo repository.VendorRepository, writer *kafka.Writer, logger *logrus.Logger) VendorService {
	return &vendorService{
		repo:   repo,
		writer: writer,
		logger: logger,
	}
}

func (s *vendorService) RegisterVendor(ctx context.Context, req *models.RegisterVendorRequest) (*models.VendorResponse, error) {
	// Check if email already registered
	existing, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("vendor with this email already exists")
	}

	commissionRate := req.CommissionRate
	if commissionRate <= 0 {
		commissionRate = 10 // default 10%
	}

	now := time.Now().UTC()
	vendor := &models.Vendor{
		ID:             uuid.New().String(),
		TenantID:       req.TenantID,
		Name:           req.Name,
		Email:          req.Email,
		Phone:          req.Phone,
		Description:    req.Description,
		LogoURL:        req.LogoURL,
		Address:        req.Address,
		City:           req.City,
		Country:        req.Country,
		Status:         models.StatusPending,
		CommissionRate: commissionRate,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, vendor); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, fmt.Errorf("vendor with this email already exists")
		}
		return nil, fmt.Errorf("failed to register vendor: %w", err)
	}

	s.publishEvent("VendorRegistered", vendor.TenantID, map[string]interface{}{
		"vendor_id": vendor.ID,
		"tenant_id": vendor.TenantID,
		"name":      vendor.Name,
		"email":     vendor.Email,
	})

	return toVendorResponse(vendor), nil
}

func (s *vendorService) GetVendor(ctx context.Context, id string) (*models.VendorResponse, error) {
	vendor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toVendorResponse(vendor), nil
}

func (s *vendorService) ListVendors(ctx context.Context, tenantID, status string, page, pageSize int) ([]models.VendorResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	vendors, total, err := s.repo.List(ctx, tenantID, status, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list vendors: %w", err)
	}

	responses := make([]models.VendorResponse, len(vendors))
	for i, v := range vendors {
		responses[i] = *toVendorResponse(&v)
	}
	return responses, total, nil
}

func (s *vendorService) UpdateVendor(ctx context.Context, id string, req *models.UpdateVendorRequest) (*models.VendorResponse, error) {
	vendor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		vendor.Name = req.Name
	}
	if req.Phone != "" {
		vendor.Phone = req.Phone
	}
	if req.Description != "" {
		vendor.Description = req.Description
	}
	if req.LogoURL != "" {
		vendor.LogoURL = req.LogoURL
	}
	if req.Address != "" {
		vendor.Address = req.Address
	}
	if req.City != "" {
		vendor.City = req.City
	}
	if req.Country != "" {
		vendor.Country = req.Country
	}
	vendor.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, vendor); err != nil {
		return nil, fmt.Errorf("failed to update vendor: %w", err)
	}

	return toVendorResponse(vendor), nil
}

func (s *vendorService) UpdateVendorStatus(ctx context.Context, id string, req *models.UpdateVendorStatusRequest) (*models.VendorResponse, error) {
	vendor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate status transition
	if !isValidStatusTransition(vendor.Status, req.Status) {
		return nil, fmt.Errorf("invalid status transition from %s to %s", vendor.Status, req.Status)
	}

	vendor.Status = req.Status
	vendor.UpdatedAt = time.Now().UTC()

	if req.Status == models.StatusApproved {
		now := time.Now().UTC()
		vendor.ApprovedAt = &now
	}
	if req.Status == models.StatusSuspended {
		vendor.SuspendReason = req.Reason
	}

	if err := s.repo.Update(ctx, vendor); err != nil {
		return nil, fmt.Errorf("failed to update vendor status: %w", err)
	}

	eventType := "VendorApproved"
	if req.Status == models.StatusSuspended {
		eventType = "VendorSuspended"
	} else if req.Status == models.StatusRejected {
		eventType = "VendorRejected"
	}

	s.publishEvent(eventType, vendor.TenantID, map[string]interface{}{
		"vendor_id": vendor.ID,
		"status":    vendor.Status,
		"reason":    req.Reason,
	})

	return toVendorResponse(vendor), nil
}

func (s *vendorService) GetVendorOrders(ctx context.Context, vendorID string, page, pageSize int) ([]models.VendorOrderResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	orders, total, err := s.repo.GetOrdersByVendor(ctx, vendorID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get vendor orders: %w", err)
	}

	responses := make([]models.VendorOrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = models.VendorOrderResponse{
			ID:         o.ID,
			VendorID:   o.VendorID,
			OrderID:    o.OrderID,
			Amount:     o.Amount,
			Commission: o.Commission,
			NetAmount:  o.NetAmount,
			Status:     o.Status,
			CreatedAt:  o.CreatedAt,
		}
	}
	return responses, total, nil
}

func (s *vendorService) GetVendorAnalytics(ctx context.Context, vendorID string) (*models.VendorAnalyticsResponse, error) {
	analytics, err := s.repo.GetVendorAnalytics(ctx, vendorID)
	if err != nil {
		return nil, err
	}
	return analytics, nil
}

func (s *vendorService) RecordOrder(ctx context.Context, vendorID, tenantID, orderID string, amount float64) error {
	vendor, err := s.repo.GetByID(ctx, vendorID)
	if err != nil {
		return fmt.Errorf("vendor not found: %w", err)
	}

	commission := amount * (vendor.CommissionRate / 100)
	netAmount := amount - commission

	order := &models.VendorOrder{
		ID:         uuid.New().String(),
		VendorID:   vendorID,
		TenantID:   tenantID,
		OrderID:    orderID,
		Amount:     amount,
		Commission: commission,
		NetAmount:  netAmount,
		Status:     "pending",
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to record vendor order: %w", err)
	}

	// Update vendor totals
	vendor.TotalRevenue += amount
	vendor.TotalOrders++
	vendor.UpdatedAt = time.Now().UTC()
	s.repo.Update(ctx, vendor)

	return nil
}

func isValidStatusTransition(from, to models.VendorStatus) bool {
	transitions := map[models.VendorStatus][]models.VendorStatus{
		models.StatusPending:   {models.StatusApproved, models.StatusRejected},
		models.StatusApproved:  {models.StatusSuspended},
		models.StatusSuspended: {models.StatusApproved},
		models.StatusRejected:  {models.StatusPending},
	}

	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func toVendorResponse(v *models.Vendor) *models.VendorResponse {
	return &models.VendorResponse{
		ID:             v.ID,
		TenantID:       v.TenantID,
		Name:           v.Name,
		Email:          v.Email,
		Phone:          v.Phone,
		Description:    v.Description,
		LogoURL:        v.LogoURL,
		Address:        v.Address,
		City:           v.City,
		Country:        v.Country,
		Status:         v.Status,
		CommissionRate: v.CommissionRate,
		TotalRevenue:   v.TotalRevenue,
		TotalOrders:    v.TotalOrders,
		TotalProducts:  v.TotalProducts,
		Rating:         v.Rating,
		ApprovedAt:     v.ApprovedAt,
		CreatedAt:      v.CreatedAt,
	}
}

func (s *vendorService) publishEvent(eventType, tenantID string, payload interface{}) {
	if s.writer == nil {
		return
	}

	event := models.VendorEvent{
		EventID:   uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal vendor event")
		return
	}

	err = s.writer.WriteMessages(context.Background(), kafka.Message{
		Topic: "vendor-events",
		Key:   []byte(tenantID),
		Value: data,
	})
	if err != nil {
		s.logger.WithError(err).Error("Failed to publish vendor event")
	}
}

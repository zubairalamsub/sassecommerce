package repository

import (
	"context"
	"errors"

	"github.com/ecommerce/shipping-service/internal/models"
	"gorm.io/gorm"
)

type ShipmentRepository interface {
	Create(ctx context.Context, shipment *models.Shipment) error
	GetByID(ctx context.Context, id string) (*models.Shipment, error)
	GetByIDWithDetails(ctx context.Context, id string) (*models.Shipment, error)
	GetByTrackingNumber(ctx context.Context, trackingNumber string) (*models.Shipment, error)
	GetByOrderID(ctx context.Context, tenantID, orderID string) (*models.Shipment, error)
	List(ctx context.Context, tenantID string, page, pageSize int, status string) ([]models.Shipment, int64, error)
	Update(ctx context.Context, shipment *models.Shipment) error
	CreateEvent(ctx context.Context, event *models.ShipmentEvent) error
	GetEvents(ctx context.Context, shipmentID string) ([]models.ShipmentEvent, error)
}

type shipmentRepository struct {
	db *gorm.DB
}

func NewShipmentRepository(db *gorm.DB) ShipmentRepository {
	return &shipmentRepository{db: db}
}

func (r *shipmentRepository) Create(ctx context.Context, shipment *models.Shipment) error {
	return r.db.WithContext(ctx).Create(shipment).Error
}

func (r *shipmentRepository) GetByID(ctx context.Context, id string) (*models.Shipment, error) {
	var shipment models.Shipment
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&shipment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shipment not found")
		}
		return nil, err
	}
	return &shipment, nil
}

func (r *shipmentRepository) GetByIDWithDetails(ctx context.Context, id string) (*models.Shipment, error) {
	var shipment models.Shipment
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("occurred_at DESC")
		}).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&shipment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shipment not found")
		}
		return nil, err
	}
	return &shipment, nil
}

func (r *shipmentRepository) GetByTrackingNumber(ctx context.Context, trackingNumber string) (*models.Shipment, error) {
	var shipment models.Shipment
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("occurred_at DESC")
		}).
		Where("tracking_number = ? AND deleted_at IS NULL", trackingNumber).
		First(&shipment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shipment not found")
		}
		return nil, err
	}
	return &shipment, nil
}

func (r *shipmentRepository) GetByOrderID(ctx context.Context, tenantID, orderID string) (*models.Shipment, error) {
	var shipment models.Shipment
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("occurred_at DESC")
		}).
		Where("tenant_id = ? AND order_id = ? AND deleted_at IS NULL", tenantID, orderID).
		First(&shipment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("shipment not found")
		}
		return nil, err
	}
	return &shipment, nil
}

func (r *shipmentRepository) List(ctx context.Context, tenantID string, page, pageSize int, status string) ([]models.Shipment, int64, error) {
	var shipments []models.Shipment
	var total int64

	offset := (page - 1) * pageSize

	query := r.db.WithContext(ctx).Model(&models.Shipment{}).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Scopes(func(db *gorm.DB) *gorm.DB {
			if status != "" {
				return db.Where("status = ?", status)
			}
			return db
		}).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&shipments).Error

	if err != nil {
		return nil, 0, err
	}

	return shipments, total, nil
}

func (r *shipmentRepository) Update(ctx context.Context, shipment *models.Shipment) error {
	return r.db.WithContext(ctx).Save(shipment).Error
}

func (r *shipmentRepository) CreateEvent(ctx context.Context, event *models.ShipmentEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *shipmentRepository) GetEvents(ctx context.Context, shipmentID string) ([]models.ShipmentEvent, error) {
	var events []models.ShipmentEvent
	err := r.db.WithContext(ctx).
		Where("shipment_id = ?", shipmentID).
		Order("occurred_at DESC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

package repository

import (
	"context"
	"fmt"

	"github.com/ecommerce/vendor-service/internal/models"
	"gorm.io/gorm"
)

// VendorRepository defines the interface for vendor data access
type VendorRepository interface {
	Create(ctx context.Context, vendor *models.Vendor) error
	GetByID(ctx context.Context, id string) (*models.Vendor, error)
	GetByEmail(ctx context.Context, email string) (*models.Vendor, error)
	List(ctx context.Context, tenantID string, status string, page, pageSize int) ([]models.Vendor, int64, error)
	Update(ctx context.Context, vendor *models.Vendor) error

	CreateOrder(ctx context.Context, order *models.VendorOrder) error
	GetOrdersByVendor(ctx context.Context, vendorID string, page, pageSize int) ([]models.VendorOrder, int64, error)
	GetVendorAnalytics(ctx context.Context, vendorID string) (*models.VendorAnalyticsResponse, error)
}

type gormVendorRepository struct {
	db *gorm.DB
}

func NewVendorRepository(db *gorm.DB) VendorRepository {
	return &gormVendorRepository{db: db}
}

func (r *gormVendorRepository) Create(ctx context.Context, vendor *models.Vendor) error {
	return r.db.WithContext(ctx).Create(vendor).Error
}

func (r *gormVendorRepository) GetByID(ctx context.Context, id string) (*models.Vendor, error) {
	var vendor models.Vendor
	if err := r.db.WithContext(ctx).First(&vendor, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("vendor not found")
		}
		return nil, err
	}
	return &vendor, nil
}

func (r *gormVendorRepository) GetByEmail(ctx context.Context, email string) (*models.Vendor, error) {
	var vendor models.Vendor
	if err := r.db.WithContext(ctx).First(&vendor, "email = ?", email).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("vendor not found")
		}
		return nil, err
	}
	return &vendor, nil
}

func (r *gormVendorRepository) List(ctx context.Context, tenantID string, status string, page, pageSize int) ([]models.Vendor, int64, error) {
	var vendors []models.Vendor
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Vendor{}).Where("tenant_id = ?", tenantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&vendors).Error
	return vendors, total, err
}

func (r *gormVendorRepository) Update(ctx context.Context, vendor *models.Vendor) error {
	return r.db.WithContext(ctx).Save(vendor).Error
}

func (r *gormVendorRepository) CreateOrder(ctx context.Context, order *models.VendorOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *gormVendorRepository) GetOrdersByVendor(ctx context.Context, vendorID string, page, pageSize int) ([]models.VendorOrder, int64, error) {
	var orders []models.VendorOrder
	var total int64

	r.db.WithContext(ctx).Model(&models.VendorOrder{}).Where("vendor_id = ?", vendorID).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Where("vendor_id = ?", vendorID).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&orders).Error
	return orders, total, err
}

func (r *gormVendorRepository) GetVendorAnalytics(ctx context.Context, vendorID string) (*models.VendorAnalyticsResponse, error) {
	var vendor models.Vendor
	if err := r.db.WithContext(ctx).First(&vendor, "id = ?", vendorID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("vendor not found")
		}
		return nil, err
	}

	var totalCommission float64
	r.db.WithContext(ctx).Model(&models.VendorOrder{}).
		Where("vendor_id = ?", vendorID).
		Select("COALESCE(SUM(commission), 0)").
		Scan(&totalCommission)

	return &models.VendorAnalyticsResponse{
		VendorID:       vendor.ID,
		TotalRevenue:   vendor.TotalRevenue,
		TotalOrders:    vendor.TotalOrders,
		TotalProducts:  vendor.TotalProducts,
		CommissionPaid: totalCommission,
		NetEarnings:    vendor.TotalRevenue - totalCommission,
		Rating:         vendor.Rating,
	}, nil
}

package repository

import (
	"context"

	"github.com/ecommerce/config-service/internal/models"
	"gorm.io/gorm"
)

type MenuRepository interface {
	// Menus
	CreateMenu(ctx context.Context, menu *models.Menu) error
	GetMenu(ctx context.Context, id string) (*models.Menu, error)
	GetMenuBySlug(ctx context.Context, tenantID, slug string) (*models.Menu, error)
	UpdateMenu(ctx context.Context, menu *models.Menu) error
	DeleteMenu(ctx context.Context, id string) error
	ListMenus(ctx context.Context, tenantID string) ([]models.Menu, error)
	ListMenusByLocation(ctx context.Context, tenantID, location string) ([]models.Menu, error)

	// Menu items
	CreateMenuItem(ctx context.Context, item *models.MenuItem) error
	GetMenuItem(ctx context.Context, id string) (*models.MenuItem, error)
	UpdateMenuItem(ctx context.Context, item *models.MenuItem) error
	DeleteMenuItem(ctx context.Context, id string) error
	GetMenuItems(ctx context.Context, menuID string) ([]models.MenuItem, error)
	BulkUpdatePositions(ctx context.Context, items []models.MenuItem) error
}

type menuRepository struct {
	db *gorm.DB
}

func NewMenuRepository(db *gorm.DB) MenuRepository {
	return &menuRepository{db: db}
}

func (r *menuRepository) CreateMenu(ctx context.Context, menu *models.Menu) error {
	return r.db.WithContext(ctx).Create(menu).Error
}

func (r *menuRepository) GetMenu(ctx context.Context, id string) (*models.Menu, error) {
	var menu models.Menu
	if err := r.db.WithContext(ctx).First(&menu, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}

func (r *menuRepository) GetMenuBySlug(ctx context.Context, tenantID, slug string) (*models.Menu, error) {
	var menu models.Menu
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND slug = ?", tenantID, slug).First(&menu).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}

func (r *menuRepository) UpdateMenu(ctx context.Context, menu *models.Menu) error {
	return r.db.WithContext(ctx).Save(menu).Error
}

func (r *menuRepository) DeleteMenu(ctx context.Context, id string) error {
	// Items cascade via FK
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Menu{}).Error
}

func (r *menuRepository) ListMenus(ctx context.Context, tenantID string) ([]models.Menu, error) {
	var menus []models.Menu
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).
		Order("location ASC, name ASC").Find(&menus).Error
	return menus, err
}

func (r *menuRepository) ListMenusByLocation(ctx context.Context, tenantID, location string) ([]models.Menu, error) {
	var menus []models.Menu
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND location = ? AND is_active = true", tenantID, location).
		Order("name ASC").Find(&menus).Error
	return menus, err
}

func (r *menuRepository) CreateMenuItem(ctx context.Context, item *models.MenuItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *menuRepository) GetMenuItem(ctx context.Context, id string) (*models.MenuItem, error) {
	var item models.MenuItem
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *menuRepository) UpdateMenuItem(ctx context.Context, item *models.MenuItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *menuRepository) DeleteMenuItem(ctx context.Context, id string) error {
	// Also delete child items
	r.db.WithContext(ctx).Where("parent_id = ?", id).Delete(&models.MenuItem{})
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.MenuItem{}).Error
}

func (r *menuRepository) GetMenuItems(ctx context.Context, menuID string) ([]models.MenuItem, error) {
	var items []models.MenuItem
	err := r.db.WithContext(ctx).Where("menu_id = ?", menuID).
		Order("position ASC, created_at ASC").Find(&items).Error
	return items, err
}

func (r *menuRepository) BulkUpdatePositions(ctx context.Context, items []models.MenuItem) error {
	for _, item := range items {
		if err := r.db.WithContext(ctx).Model(&models.MenuItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]interface{}{
				"position":  item.Position,
				"parent_id": item.ParentID,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

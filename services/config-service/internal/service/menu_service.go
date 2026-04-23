package service

import (
	"context"
	"fmt"

	"github.com/ecommerce/config-service/internal/models"
	"github.com/ecommerce/config-service/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type MenuService interface {
	// Menus
	CreateMenu(ctx context.Context, req *models.CreateMenuRequest) (*models.MenuResponse, error)
	GetMenu(ctx context.Context, id string) (*models.MenuResponse, error)
	GetMenuBySlug(ctx context.Context, tenantID, slug string) (*models.MenuResponse, error)
	UpdateMenu(ctx context.Context, id string, req *models.UpdateMenuRequest) (*models.MenuResponse, error)
	DeleteMenu(ctx context.Context, id string) error
	ListMenus(ctx context.Context, tenantID string) ([]models.MenuResponse, error)
	ListMenusByLocation(ctx context.Context, tenantID, location string) ([]models.MenuResponse, error)

	// Menu items
	CreateMenuItem(ctx context.Context, req *models.CreateMenuItemRequest) (*models.MenuItemResponse, error)
	UpdateMenuItem(ctx context.Context, id string, req *models.UpdateMenuItemRequest) (*models.MenuItemResponse, error)
	DeleteMenuItem(ctx context.Context, id string) error
	ReorderItems(ctx context.Context, menuID string, req *models.ReorderItemsRequest) error
}

type menuService struct {
	repo   repository.MenuRepository
	logger *logrus.Logger
}

func NewMenuService(repo repository.MenuRepository, logger *logrus.Logger) MenuService {
	return &menuService{
		repo:   repo,
		logger: logger,
	}
}

func (s *menuService) CreateMenu(ctx context.Context, req *models.CreateMenuRequest) (*models.MenuResponse, error) {
	// Validate location
	if !isValidLocation(req.Location) {
		return nil, fmt.Errorf("invalid location: must be header, footer, sidebar, or mobile")
	}

	// Check for duplicate slug
	existing, _ := s.repo.GetMenuBySlug(ctx, req.TenantID, req.Slug)
	if existing != nil {
		return nil, fmt.Errorf("menu with slug '%s' already exists for this tenant", req.Slug)
	}

	menu := &models.Menu{
		ID:          uuid.New().String(),
		TenantID:    req.TenantID,
		Name:        req.Name,
		Slug:        req.Slug,
		Location:    req.Location,
		Description: req.Description,
		IsActive:    true,
	}

	if err := s.repo.CreateMenu(ctx, menu); err != nil {
		return nil, fmt.Errorf("failed to create menu: %w", err)
	}

	return toMenuResponse(menu, nil), nil
}

func (s *menuService) GetMenu(ctx context.Context, id string) (*models.MenuResponse, error) {
	menu, err := s.repo.GetMenu(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("menu not found")
	}

	items, err := s.repo.GetMenuItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get menu items: %w", err)
	}

	tree := buildMenuTree(items)
	return toMenuResponse(menu, tree), nil
}

func (s *menuService) GetMenuBySlug(ctx context.Context, tenantID, slug string) (*models.MenuResponse, error) {
	menu, err := s.repo.GetMenuBySlug(ctx, tenantID, slug)
	if err != nil {
		return nil, fmt.Errorf("menu not found")
	}

	items, err := s.repo.GetMenuItems(ctx, menu.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get menu items: %w", err)
	}

	tree := buildMenuTree(items)
	return toMenuResponse(menu, tree), nil
}

func (s *menuService) UpdateMenu(ctx context.Context, id string, req *models.UpdateMenuRequest) (*models.MenuResponse, error) {
	menu, err := s.repo.GetMenu(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("menu not found")
	}

	if req.Name != "" {
		menu.Name = req.Name
	}
	if req.Location != "" {
		if !isValidLocation(req.Location) {
			return nil, fmt.Errorf("invalid location: must be header, footer, sidebar, or mobile")
		}
		menu.Location = req.Location
	}
	if req.Description != "" {
		menu.Description = req.Description
	}
	if req.IsActive != nil {
		menu.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateMenu(ctx, menu); err != nil {
		return nil, fmt.Errorf("failed to update menu: %w", err)
	}

	items, _ := s.repo.GetMenuItems(ctx, id)
	tree := buildMenuTree(items)
	return toMenuResponse(menu, tree), nil
}

func (s *menuService) DeleteMenu(ctx context.Context, id string) error {
	_, err := s.repo.GetMenu(ctx, id)
	if err != nil {
		return fmt.Errorf("menu not found")
	}

	if err := s.repo.DeleteMenu(ctx, id); err != nil {
		return fmt.Errorf("failed to delete menu: %w", err)
	}
	return nil
}

func (s *menuService) ListMenus(ctx context.Context, tenantID string) ([]models.MenuResponse, error) {
	menus, err := s.repo.ListMenus(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list menus: %w", err)
	}

	responses := make([]models.MenuResponse, len(menus))
	for i, m := range menus {
		items, _ := s.repo.GetMenuItems(ctx, m.ID)
		tree := buildMenuTree(items)
		responses[i] = *toMenuResponse(&m, tree)
	}
	return responses, nil
}

func (s *menuService) ListMenusByLocation(ctx context.Context, tenantID, location string) ([]models.MenuResponse, error) {
	if !isValidLocation(location) {
		return nil, fmt.Errorf("invalid location: must be header, footer, sidebar, or mobile")
	}

	menus, err := s.repo.ListMenusByLocation(ctx, tenantID, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list menus: %w", err)
	}

	responses := make([]models.MenuResponse, len(menus))
	for i, m := range menus {
		items, _ := s.repo.GetMenuItems(ctx, m.ID)
		tree := buildMenuTree(items)
		responses[i] = *toMenuResponse(&m, tree)
	}
	return responses, nil
}

func (s *menuService) CreateMenuItem(ctx context.Context, req *models.CreateMenuItemRequest) (*models.MenuItemResponse, error) {
	// Verify menu exists
	_, err := s.repo.GetMenu(ctx, req.MenuID)
	if err != nil {
		return nil, fmt.Errorf("menu not found")
	}

	// Verify parent exists if specified
	if req.ParentID != "" {
		_, err := s.repo.GetMenuItem(ctx, req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parent item not found")
		}
	}

	target := req.Target
	if target == "" {
		target = models.TargetSelf
	}

	item := &models.MenuItem{
		ID:       uuid.New().String(),
		MenuID:   req.MenuID,
		ParentID: req.ParentID,
		Label:    req.Label,
		URL:      req.URL,
		Icon:     req.Icon,
		Target:   target,
		CSSClass: req.CSSClass,
		Position: req.Position,
		IsActive: true,
		Metadata: req.Metadata,
	}

	if err := s.repo.CreateMenuItem(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to create menu item: %w", err)
	}

	return toMenuItemResponse(item), nil
}

func (s *menuService) UpdateMenuItem(ctx context.Context, id string, req *models.UpdateMenuItemRequest) (*models.MenuItemResponse, error) {
	item, err := s.repo.GetMenuItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("menu item not found")
	}

	if req.Label != "" {
		item.Label = req.Label
	}
	if req.URL != "" {
		item.URL = req.URL
	}
	if req.Icon != "" {
		item.Icon = req.Icon
	}
	if req.Target != "" {
		item.Target = req.Target
	}
	if req.CSSClass != "" {
		item.CSSClass = req.CSSClass
	}
	if req.Position != nil {
		item.Position = *req.Position
	}
	if req.IsActive != nil {
		item.IsActive = *req.IsActive
	}
	if req.ParentID != nil {
		item.ParentID = *req.ParentID
	}
	if req.Metadata != "" {
		item.Metadata = req.Metadata
	}

	if err := s.repo.UpdateMenuItem(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to update menu item: %w", err)
	}

	return toMenuItemResponse(item), nil
}

func (s *menuService) DeleteMenuItem(ctx context.Context, id string) error {
	_, err := s.repo.GetMenuItem(ctx, id)
	if err != nil {
		return fmt.Errorf("menu item not found")
	}

	if err := s.repo.DeleteMenuItem(ctx, id); err != nil {
		return fmt.Errorf("failed to delete menu item: %w", err)
	}
	return nil
}

func (s *menuService) ReorderItems(ctx context.Context, menuID string, req *models.ReorderItemsRequest) error {
	// Verify menu exists
	_, err := s.repo.GetMenu(ctx, menuID)
	if err != nil {
		return fmt.Errorf("menu not found")
	}

	items := make([]models.MenuItem, len(req.Items))
	for i, r := range req.Items {
		items[i] = models.MenuItem{
			ID:       r.ID,
			Position: r.Position,
			ParentID: r.ParentID,
		}
	}

	if err := s.repo.BulkUpdatePositions(ctx, items); err != nil {
		return fmt.Errorf("failed to reorder items: %w", err)
	}
	return nil
}

// === Helpers ===

func isValidLocation(location string) bool {
	switch location {
	case models.MenuLocationHeader, models.MenuLocationFooter,
		models.MenuLocationSidebar, models.MenuLocationMobile:
		return true
	}
	return false
}

func buildMenuTree(items []models.MenuItem) []models.MenuItemResponse {
	itemMap := make(map[string]*models.MenuItemResponse)

	// First pass: create response objects
	for i := range items {
		resp := toMenuItemResponse(&items[i])
		itemMap[items[i].ID] = resp
	}

	// Second pass: assign children to parents, track root IDs
	var rootIDs []string
	for _, item := range items {
		if item.ParentID == "" {
			rootIDs = append(rootIDs, item.ID)
		} else {
			if parent, ok := itemMap[item.ParentID]; ok {
				child := itemMap[item.ID]
				parent.Children = append(parent.Children, *child)
			} else {
				// Orphan item, treat as root
				rootIDs = append(rootIDs, item.ID)
			}
		}
	}

	// Third pass: collect roots (after all children are assigned)
	roots := make([]models.MenuItemResponse, 0, len(rootIDs))
	for _, id := range rootIDs {
		roots = append(roots, *itemMap[id])
	}

	return roots
}

func toMenuResponse(menu *models.Menu, items []models.MenuItemResponse) *models.MenuResponse {
	if items == nil {
		items = []models.MenuItemResponse{}
	}
	return &models.MenuResponse{
		ID:          menu.ID,
		TenantID:    menu.TenantID,
		Name:        menu.Name,
		Slug:        menu.Slug,
		Location:    menu.Location,
		Description: menu.Description,
		IsActive:    menu.IsActive,
		Items:       items,
		CreatedAt:   menu.CreatedAt,
		UpdatedAt:   menu.UpdatedAt,
	}
}

func toMenuItemResponse(item *models.MenuItem) *models.MenuItemResponse {
	return &models.MenuItemResponse{
		ID:       item.ID,
		MenuID:   item.MenuID,
		ParentID: item.ParentID,
		Label:    item.Label,
		URL:      item.URL,
		Icon:     item.Icon,
		Target:   item.Target,
		CSSClass: item.CSSClass,
		Position: item.Position,
		IsActive: item.IsActive,
		Metadata: item.Metadata,
		Children: []models.MenuItemResponse{},
	}
}

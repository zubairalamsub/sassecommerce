package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ecommerce/config-service/internal/models"
	repoMocks "github.com/ecommerce/config-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newMenuTestService() (*menuService, *repoMocks.MockMenuRepository) {
	mockRepo := new(repoMocks.MockMenuRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &menuService{
		repo:   mockRepo,
		logger: logger,
	}

	return svc, mockRepo
}

// === CreateMenu Tests ===

func TestCreateMenu_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenuBySlug", ctx, "tenant-1", "main-nav").Return(nil, errors.New("not found"))
	mockRepo.On("CreateMenu", ctx, mock.AnythingOfType("*models.Menu")).Return(nil)

	req := &models.CreateMenuRequest{
		TenantID: "tenant-1", Name: "Main Navigation",
		Slug: "main-nav", Location: "header",
	}

	result, err := svc.CreateMenu(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "Main Navigation", result.Name)
	assert.Equal(t, "header", result.Location)
	assert.True(t, result.IsActive)
}

func TestCreateMenu_InvalidLocation(t *testing.T) {
	svc, _ := newMenuTestService()
	ctx := context.Background()

	req := &models.CreateMenuRequest{
		TenantID: "tenant-1", Name: "Bad Menu",
		Slug: "bad", Location: "invalid",
	}

	result, err := svc.CreateMenu(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid location")
}

func TestCreateMenu_DuplicateSlug(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	existing := &models.Menu{ID: "m-1", TenantID: "tenant-1", Slug: "main-nav"}
	mockRepo.On("GetMenuBySlug", ctx, "tenant-1", "main-nav").Return(existing, nil)

	req := &models.CreateMenuRequest{
		TenantID: "tenant-1", Name: "Duplicate",
		Slug: "main-nav", Location: "header",
	}

	result, err := svc.CreateMenu(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCreateMenu_RepoFailure(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenuBySlug", ctx, "tenant-1", "test").Return(nil, errors.New("not found"))
	mockRepo.On("CreateMenu", ctx, mock.AnythingOfType("*models.Menu")).Return(errors.New("db error"))

	req := &models.CreateMenuRequest{
		TenantID: "tenant-1", Name: "Test", Slug: "test", Location: "header",
	}

	result, err := svc.CreateMenu(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetMenu Tests ===

func TestGetMenu_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1", TenantID: "tenant-1", Name: "Main Nav", Slug: "main-nav", Location: "header", IsActive: true}
	items := []models.MenuItem{
		{ID: "i-1", MenuID: "m-1", Label: "Home", URL: "/", Position: 0},
		{ID: "i-2", MenuID: "m-1", Label: "Shop", URL: "/shop", Position: 1},
	}

	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)
	mockRepo.On("GetMenuItems", ctx, "m-1").Return(items, nil)

	result, err := svc.GetMenu(ctx, "m-1")

	assert.NoError(t, err)
	assert.Equal(t, "Main Nav", result.Name)
	assert.Len(t, result.Items, 2)
}

func TestGetMenu_NotFound(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenu", ctx, "bad").Return(nil, errors.New("not found"))

	result, err := svc.GetMenu(ctx, "bad")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetMenu_WithTreeStructure(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1", Name: "Nav", Location: "header", IsActive: true}
	items := []models.MenuItem{
		{ID: "i-1", MenuID: "m-1", Label: "Shop", URL: "/shop", Position: 0},
		{ID: "i-2", MenuID: "m-1", ParentID: "i-1", Label: "Electronics", URL: "/shop/electronics", Position: 0},
		{ID: "i-3", MenuID: "m-1", ParentID: "i-1", Label: "Clothing", URL: "/shop/clothing", Position: 1},
	}

	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)
	mockRepo.On("GetMenuItems", ctx, "m-1").Return(items, nil)

	result, err := svc.GetMenu(ctx, "m-1")

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1) // Only "Shop" at root
	assert.Equal(t, "Shop", result.Items[0].Label)
	assert.Len(t, result.Items[0].Children, 2) // Electronics + Clothing
	assert.Equal(t, "Electronics", result.Items[0].Children[0].Label)
}

// === GetMenuBySlug Tests ===

func TestGetMenuBySlug_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1", TenantID: "tenant-1", Slug: "main-nav", Name: "Main Nav", Location: "header", IsActive: true}
	mockRepo.On("GetMenuBySlug", ctx, "tenant-1", "main-nav").Return(menu, nil)
	mockRepo.On("GetMenuItems", ctx, "m-1").Return([]models.MenuItem{}, nil)

	result, err := svc.GetMenuBySlug(ctx, "tenant-1", "main-nav")

	assert.NoError(t, err)
	assert.Equal(t, "Main Nav", result.Name)
}

func TestGetMenuBySlug_NotFound(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenuBySlug", ctx, "tenant-1", "bad").Return(nil, errors.New("not found"))

	result, err := svc.GetMenuBySlug(ctx, "tenant-1", "bad")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === UpdateMenu Tests ===

func TestUpdateMenu_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1", Name: "Old Name", Location: "header", IsActive: true}
	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)
	mockRepo.On("UpdateMenu", ctx, mock.AnythingOfType("*models.Menu")).Return(nil)
	mockRepo.On("GetMenuItems", ctx, "m-1").Return([]models.MenuItem{}, nil)

	req := &models.UpdateMenuRequest{Name: "New Name"}
	result, err := svc.UpdateMenu(ctx, "m-1", req)

	assert.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
}

func TestUpdateMenu_NotFound(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenu", ctx, "bad").Return(nil, errors.New("not found"))

	req := &models.UpdateMenuRequest{Name: "New Name"}
	result, err := svc.UpdateMenu(ctx, "bad", req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUpdateMenu_InvalidLocation(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1", Name: "Nav", Location: "header"}
	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)

	req := &models.UpdateMenuRequest{Location: "invalid"}
	result, err := svc.UpdateMenu(ctx, "m-1", req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid location")
}

func TestUpdateMenu_IsActive(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1", Name: "Nav", Location: "header", IsActive: true}
	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)
	mockRepo.On("UpdateMenu", ctx, mock.AnythingOfType("*models.Menu")).Return(nil)
	mockRepo.On("GetMenuItems", ctx, "m-1").Return([]models.MenuItem{}, nil)

	isActive := false
	req := &models.UpdateMenuRequest{IsActive: &isActive}
	result, err := svc.UpdateMenu(ctx, "m-1", req)

	assert.NoError(t, err)
	assert.False(t, result.IsActive)
}

// === DeleteMenu Tests ===

func TestDeleteMenu_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1"}
	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)
	mockRepo.On("DeleteMenu", ctx, "m-1").Return(nil)

	err := svc.DeleteMenu(ctx, "m-1")

	assert.NoError(t, err)
}

func TestDeleteMenu_NotFound(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenu", ctx, "bad").Return(nil, errors.New("not found"))

	err := svc.DeleteMenu(ctx, "bad")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// === ListMenus Tests ===

func TestListMenus_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menus := []models.Menu{
		{ID: "m-1", TenantID: "tenant-1", Name: "Header", Location: "header"},
		{ID: "m-2", TenantID: "tenant-1", Name: "Footer", Location: "footer"},
	}
	mockRepo.On("ListMenus", ctx, "tenant-1").Return(menus, nil)
	mockRepo.On("GetMenuItems", ctx, "m-1").Return([]models.MenuItem{}, nil)
	mockRepo.On("GetMenuItems", ctx, "m-2").Return([]models.MenuItem{}, nil)

	results, err := svc.ListMenus(ctx, "tenant-1")

	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

// === ListMenusByLocation Tests ===

func TestListMenusByLocation_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menus := []models.Menu{
		{ID: "m-1", TenantID: "tenant-1", Name: "Main Nav", Location: "header"},
	}
	mockRepo.On("ListMenusByLocation", ctx, "tenant-1", "header").Return(menus, nil)
	mockRepo.On("GetMenuItems", ctx, "m-1").Return([]models.MenuItem{}, nil)

	results, err := svc.ListMenusByLocation(ctx, "tenant-1", "header")

	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestListMenusByLocation_InvalidLocation(t *testing.T) {
	svc, _ := newMenuTestService()
	ctx := context.Background()

	results, err := svc.ListMenusByLocation(ctx, "tenant-1", "invalid")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "invalid location")
}

// === CreateMenuItem Tests ===

func TestCreateMenuItem_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1"}
	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)
	mockRepo.On("CreateMenuItem", ctx, mock.AnythingOfType("*models.MenuItem")).Return(nil)

	req := &models.CreateMenuItemRequest{
		MenuID: "m-1", Label: "Home", URL: "/",
	}

	result, err := svc.CreateMenuItem(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "Home", result.Label)
	assert.Equal(t, "/", result.URL)
	assert.Equal(t, "_self", result.Target)
}

func TestCreateMenuItem_MenuNotFound(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenu", ctx, "bad").Return(nil, errors.New("not found"))

	req := &models.CreateMenuItemRequest{
		MenuID: "bad", Label: "Home", URL: "/",
	}

	result, err := svc.CreateMenuItem(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "menu not found")
}

func TestCreateMenuItem_WithParent(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1"}
	parent := &models.MenuItem{ID: "i-1", MenuID: "m-1", Label: "Shop"}

	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)
	mockRepo.On("GetMenuItem", ctx, "i-1").Return(parent, nil)
	mockRepo.On("CreateMenuItem", ctx, mock.AnythingOfType("*models.MenuItem")).Return(nil)

	req := &models.CreateMenuItemRequest{
		MenuID: "m-1", ParentID: "i-1", Label: "Electronics", URL: "/electronics",
	}

	result, err := svc.CreateMenuItem(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "Electronics", result.Label)
}

func TestCreateMenuItem_ParentNotFound(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1"}
	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)
	mockRepo.On("GetMenuItem", ctx, "bad-parent").Return(nil, errors.New("not found"))

	req := &models.CreateMenuItemRequest{
		MenuID: "m-1", ParentID: "bad-parent", Label: "Item",
	}

	result, err := svc.CreateMenuItem(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parent item not found")
}

// === UpdateMenuItem Tests ===

func TestUpdateMenuItem_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	item := &models.MenuItem{ID: "i-1", MenuID: "m-1", Label: "Old", URL: "/old", Target: "_self", IsActive: true}
	mockRepo.On("GetMenuItem", ctx, "i-1").Return(item, nil)
	mockRepo.On("UpdateMenuItem", ctx, mock.AnythingOfType("*models.MenuItem")).Return(nil)

	req := &models.UpdateMenuItemRequest{Label: "New", URL: "/new"}
	result, err := svc.UpdateMenuItem(ctx, "i-1", req)

	assert.NoError(t, err)
	assert.Equal(t, "New", result.Label)
	assert.Equal(t, "/new", result.URL)
}

func TestUpdateMenuItem_NotFound(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenuItem", ctx, "bad").Return(nil, errors.New("not found"))

	req := &models.UpdateMenuItemRequest{Label: "New"}
	result, err := svc.UpdateMenuItem(ctx, "bad", req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === DeleteMenuItem Tests ===

func TestDeleteMenuItem_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	item := &models.MenuItem{ID: "i-1"}
	mockRepo.On("GetMenuItem", ctx, "i-1").Return(item, nil)
	mockRepo.On("DeleteMenuItem", ctx, "i-1").Return(nil)

	err := svc.DeleteMenuItem(ctx, "i-1")

	assert.NoError(t, err)
}

func TestDeleteMenuItem_NotFound(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenuItem", ctx, "bad").Return(nil, errors.New("not found"))

	err := svc.DeleteMenuItem(ctx, "bad")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// === ReorderItems Tests ===

func TestReorderItems_Success(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	menu := &models.Menu{ID: "m-1"}
	mockRepo.On("GetMenu", ctx, "m-1").Return(menu, nil)
	mockRepo.On("BulkUpdatePositions", ctx, mock.AnythingOfType("[]models.MenuItem")).Return(nil)

	req := &models.ReorderItemsRequest{
		Items: []models.ReorderItem{
			{ID: "i-1", Position: 1},
			{ID: "i-2", Position: 0},
		},
	}

	err := svc.ReorderItems(ctx, "m-1", req)

	assert.NoError(t, err)
}

func TestReorderItems_MenuNotFound(t *testing.T) {
	svc, mockRepo := newMenuTestService()
	ctx := context.Background()

	mockRepo.On("GetMenu", ctx, "bad").Return(nil, errors.New("not found"))

	req := &models.ReorderItemsRequest{
		Items: []models.ReorderItem{{ID: "i-1", Position: 0}},
	}

	err := svc.ReorderItems(ctx, "bad", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// === Helper Tests ===

func TestBuildMenuTree_Empty(t *testing.T) {
	result := buildMenuTree([]models.MenuItem{})
	assert.Empty(t, result)
}

func TestBuildMenuTree_FlatItems(t *testing.T) {
	items := []models.MenuItem{
		{ID: "i-1", Label: "Home", Position: 0},
		{ID: "i-2", Label: "Shop", Position: 1},
	}

	result := buildMenuTree(items)
	assert.Len(t, result, 2)
}

func TestBuildMenuTree_NestedItems(t *testing.T) {
	items := []models.MenuItem{
		{ID: "i-1", Label: "Shop", Position: 0},
		{ID: "i-2", ParentID: "i-1", Label: "Electronics", Position: 0},
		{ID: "i-3", ParentID: "i-1", Label: "Clothing", Position: 1},
		{ID: "i-4", Label: "About", Position: 1},
	}

	result := buildMenuTree(items)
	assert.Len(t, result, 2) // Shop + About
	assert.Equal(t, "Shop", result[0].Label)
	assert.Len(t, result[0].Children, 2)
	assert.Equal(t, "About", result[1].Label)
	assert.Empty(t, result[1].Children)
}

func TestBuildMenuTree_OrphanItems(t *testing.T) {
	items := []models.MenuItem{
		{ID: "i-1", ParentID: "nonexistent", Label: "Orphan", Position: 0},
	}

	result := buildMenuTree(items)
	assert.Len(t, result, 1) // Orphan treated as root
	assert.Equal(t, "Orphan", result[0].Label)
}

func TestIsValidLocation(t *testing.T) {
	assert.True(t, isValidLocation("header"))
	assert.True(t, isValidLocation("footer"))
	assert.True(t, isValidLocation("sidebar"))
	assert.True(t, isValidLocation("mobile"))
	assert.False(t, isValidLocation("invalid"))
	assert.False(t, isValidLocation(""))
}

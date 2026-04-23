package service

import (
	"context"
	"errors"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/ecommerce/product-service/internal/repository"
	"github.com/sirupsen/logrus"
)

// CategoryService defines the interface for category operations
type CategoryService interface {
	CreateCategory(ctx context.Context, req *models.CreateCategoryRequest) (*models.CategoryResponse, error)
	GetCategoryByID(ctx context.Context, id string) (*models.CategoryResponse, error)
	GetCategoryBySlug(ctx context.Context, tenantID, slug string) (*models.CategoryResponse, error)
	ListCategories(ctx context.Context, tenantID string, offset, limit int) ([]models.CategoryResponse, int64, error)
	ListCategoriesByParent(ctx context.Context, tenantID string, parentID *string, offset, limit int) ([]models.CategoryResponse, int64, error)
	UpdateCategory(ctx context.Context, id string, req *models.UpdateCategoryRequest) (*models.CategoryResponse, error)
	DeleteCategory(ctx context.Context, id string) error
	UpdateCategoryStatus(ctx context.Context, id string, status models.CategoryStatus) error
}

type categoryService struct {
	categoryRepo repository.CategoryRepository
	logger       *logrus.Logger
}

// NewCategoryService creates a new category service
func NewCategoryService(
	categoryRepo repository.CategoryRepository,
	logger *logrus.Logger,
) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
		logger:       logger,
	}
}

// CreateCategory creates a new category
func (s *categoryService) CreateCategory(ctx context.Context, req *models.CreateCategoryRequest) (*models.CategoryResponse, error) {
	// Check if slug already exists
	exists, err := s.categoryRepo.SlugExists(ctx, req.TenantID, req.Slug)
	if err != nil {
		s.logger.WithError(err).Error("Failed to check slug existence")
		return nil, errors.New("failed to check slug availability")
	}
	if exists {
		return nil, errors.New("slug already exists")
	}

	// Verify parent exists if provided
	if req.ParentID != nil {
		_, err := s.categoryRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			s.logger.WithError(err).Error("Parent category not found")
			return nil, errors.New("parent category not found")
		}
	}

	category := &models.Category{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		ParentID:    req.ParentID,
		Image:       req.Image,
		Icon:        req.Icon,
		SortOrder:   req.SortOrder,
		Status:      models.CategoryStatusActive,
		CreatedBy:   req.CreatedBy,
	}

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		s.logger.WithError(err).Error("Failed to create category")
		return nil, errors.New("failed to create category")
	}

	s.logger.WithFields(logrus.Fields{
		"category_id": category.ID.Hex(),
		"tenant_id":   category.TenantID,
		"slug":        category.Slug,
	}).Info("Category created successfully")

	return category.ToResponse(), nil
}

// GetCategoryByID retrieves a category by ID
func (s *categoryService) GetCategoryByID(ctx context.Context, id string) (*models.CategoryResponse, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("category_id", id).Error("Failed to get category")
		return nil, errors.New("category not found")
	}

	return category.ToResponse(), nil
}

// GetCategoryBySlug retrieves a category by slug
func (s *categoryService) GetCategoryBySlug(ctx context.Context, tenantID, slug string) (*models.CategoryResponse, error) {
	category, err := s.categoryRepo.GetBySlug(ctx, tenantID, slug)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"slug":      slug,
		}).Error("Failed to get category by slug")
		return nil, errors.New("category not found")
	}

	return category.ToResponse(), nil
}

// ListCategories retrieves categories with pagination
func (s *categoryService) ListCategories(ctx context.Context, tenantID string, offset, limit int) ([]models.CategoryResponse, int64, error) {
	categories, total, err := s.categoryRepo.List(ctx, tenantID, offset, limit)
	if err != nil {
		s.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to list categories")
		return nil, 0, errors.New("failed to retrieve categories")
	}

	responses := make([]models.CategoryResponse, len(categories))
	for i, category := range categories {
		responses[i] = *category.ToResponse()
	}

	return responses, total, nil
}

// ListCategoriesByParent retrieves categories by parent with pagination
func (s *categoryService) ListCategoriesByParent(ctx context.Context, tenantID string, parentID *string, offset, limit int) ([]models.CategoryResponse, int64, error) {
	categories, total, err := s.categoryRepo.ListByParent(ctx, tenantID, parentID, offset, limit)
	if err != nil {
		s.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to list categories by parent")
		return nil, 0, errors.New("failed to retrieve categories")
	}

	responses := make([]models.CategoryResponse, len(categories))
	for i, category := range categories {
		responses[i] = *category.ToResponse()
	}

	return responses, total, nil
}

// UpdateCategory updates a category
func (s *categoryService) UpdateCategory(ctx context.Context, id string, req *models.UpdateCategoryRequest) (*models.CategoryResponse, error) {
	// Get existing category
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("category_id", id).Error("Failed to get category for update")
		return nil, errors.New("category not found")
	}

	// Update fields if provided
	if req.Name != nil {
		category.Name = *req.Name
	}
	if req.Slug != nil {
		// Check if new slug already exists (excluding current category)
		exists, err := s.categoryRepo.SlugExists(ctx, category.TenantID, *req.Slug)
		if err != nil {
			s.logger.WithError(err).Error("Failed to check slug existence")
			return nil, errors.New("failed to check slug availability")
		}
		if exists && *req.Slug != category.Slug {
			return nil, errors.New("slug already exists")
		}
		category.Slug = *req.Slug
	}
	if req.Description != nil {
		category.Description = *req.Description
	}
	if req.ParentID != nil {
		// Verify parent exists if not nil
		if *req.ParentID != "" {
			_, err := s.categoryRepo.GetByID(ctx, *req.ParentID)
			if err != nil {
				return nil, errors.New("parent category not found")
			}
			category.ParentID = req.ParentID
		} else {
			category.ParentID = nil
		}
	}
	if req.Image != nil {
		category.Image = *req.Image
	}
	if req.Icon != nil {
		category.Icon = *req.Icon
	}
	if req.SortOrder != nil {
		category.SortOrder = *req.SortOrder
	}
	category.UpdatedBy = req.UpdatedBy

	// Save changes
	if err := s.categoryRepo.Update(ctx, id, category); err != nil {
		s.logger.WithError(err).WithField("category_id", id).Error("Failed to update category")
		return nil, errors.New("failed to update category")
	}

	s.logger.WithField("category_id", id).Info("Category updated successfully")

	return category.ToResponse(), nil
}

// DeleteCategory deletes a category (soft delete)
func (s *categoryService) DeleteCategory(ctx context.Context, id string) error {
	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		s.logger.WithError(err).WithField("category_id", id).Error("Failed to delete category")
		return errors.New("failed to delete category")
	}

	s.logger.WithField("category_id", id).Info("Category deleted successfully")

	return nil
}

// UpdateCategoryStatus updates a category's status
func (s *categoryService) UpdateCategoryStatus(ctx context.Context, id string, status models.CategoryStatus) error {
	if err := s.categoryRepo.UpdateStatus(ctx, id, status); err != nil {
		s.logger.WithError(err).WithField("category_id", id).Error("Failed to update category status")
		return errors.New("failed to update category status")
	}

	s.logger.WithFields(logrus.Fields{
		"category_id": id,
		"status":      status,
	}).Info("Category status updated successfully")

	return nil
}

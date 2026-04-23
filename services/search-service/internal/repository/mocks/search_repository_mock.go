package mocks

import (
	"context"

	"github.com/ecommerce/search-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockSearchRepository struct {
	mock.Mock
}

func (m *MockSearchRepository) EnsureIndex(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSearchRepository) IndexProduct(ctx context.Context, product *models.ProductDocument) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockSearchRepository) DeleteProduct(ctx context.Context, productID string) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

func (m *MockSearchRepository) Search(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SearchResponse), args.Error(1)
}

func (m *MockSearchRepository) Autocomplete(ctx context.Context, req *models.AutocompleteRequest) (*models.AutocompleteResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AutocompleteResponse), args.Error(1)
}

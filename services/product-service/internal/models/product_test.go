package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProduct_ToResponse_MapsFieldsAndHexEncodesID(t *testing.T) {
	id := primitive.NewObjectID()
	now := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)

	p := Product{
		ID:            id,
		TenantID:      "tenant-1",
		SKU:           "SKU-1",
		Name:          "Shirt",
		CategoryID:    "cat-1",
		Price:         500,
		InStock:       true,
		StockQuantity: 10,
		CreatedBy:     "admin-1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	resp := p.ToResponse()

	assert.Equal(t, id.Hex(), resp.ID, "the wire ID must be the ObjectID's hex encoding, not its raw bytes")
	assert.Equal(t, "tenant-1", resp.TenantID)
	assert.Equal(t, "SKU-1", resp.SKU)
	assert.Equal(t, 500.0, resp.Price)
	assert.True(t, resp.InStock)
	assert.Equal(t, 10, resp.StockQuantity)
}

// ProductResponse has no field for DeletedAt at all -- this pins that a
// soft-deleted product's response never carries the internal marker, so a
// future field addition to ProductResponse cannot accidentally leak it.
func TestProduct_ToResponse_OmitsDeletedAt(t *testing.T) {
	deletedAt := time.Now().UTC()
	p := Product{ID: primitive.NewObjectID(), Name: "Discontinued", DeletedAt: &deletedAt}

	raw, err := json.Marshal(p.ToResponse())
	assert.NoError(t, err)

	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	assert.NotContains(t, decoded, "deleted_at", "ToResponse must never leak the internal soft-delete marker to API clients")
}

func TestProduct_JSON_WireFieldNames(t *testing.T) {
	p := Product{
		ID: primitive.NewObjectID(), TenantID: "tenant-1", SKU: "SKU-1", Name: "Shirt",
		CategoryID: "cat-1", Price: 500, CompareAtPrice: 700, CostPerItem: 200,
		InStock: true, StockQuantity: 10, CreatedBy: "admin-1",
	}

	raw, err := json.Marshal(p)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &decoded))

	for _, key := range []string{
		"id", "tenant_id", "sku", "name", "slug", "description", "category_id",
		"price", "compare_at_price", "cost_per_item", "images", "tags",
		"status", "in_stock", "stock_quantity", "created_by", "created_at", "updated_at",
	} {
		assert.Contains(t, decoded, key, "expected the wire field %q", key)
	}
	assert.Equal(t, p.ID.Hex(), decoded["id"])
}

// Real finding: SEO and Dimensions are struct-valued (not pointers), and Go's
// encoding/json only ever treats false/0/""/nil/empty collections as "empty"
// for omitempty purposes -- a struct value is never considered empty no
// matter how zeroed its fields are. So `json:"seo,omitempty"` and
// `json:"dimensions,omitempty"` are silent no-ops: both keys always appear on
// the wire, contradicting what the tag implies to a reader. See the QA
// report for the fix (pointer types, or drop the tag).
func TestProduct_JSON_SEOAndDimensionsAlwaysPresentDespiteOmitempty(t *testing.T) {
	p := Product{ID: primitive.NewObjectID(), Name: "Bare Product"}

	raw, err := json.Marshal(p)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &decoded))

	assert.Contains(t, decoded, "seo", "omitempty has no effect on a non-pointer struct field, so this key is always sent even when every SEO field is zero")
	assert.Contains(t, decoded, "dimensions", "omitempty has no effect on a non-pointer struct field, so this key is always sent even when every Dimensions field is zero")
}

// TenantID is tagged json:"-" specifically so a client cannot inject another
// tenant's ID through the request body; the handler sets it from the
// authenticated JWT instead. This is the regression test for that guarantee.
func TestCreateProductRequest_TenantIDCannotBeSpoofedFromRequestBody(t *testing.T) {
	raw := []byte(`{"tenant_id":"attacker-tenant","sku":"SKU-1","name":"Shirt","category_id":"cat-1","price":10,"created_by":"u1"}`)

	var req CreateProductRequest
	assert.NoError(t, json.Unmarshal(raw, &req))

	assert.Empty(t, req.TenantID, "a client-supplied tenant_id in the body must never populate TenantID, or a request could write into another tenant's catalog")
	assert.Equal(t, "SKU-1", req.SKU)
}

// newBindingValidator is defined in category_test.go and shared across this
// package's test files.

func TestCreateProductRequest_BindingValidation(t *testing.T) {
	validate := newBindingValidator()

	base := func() CreateProductRequest {
		return CreateProductRequest{SKU: "SKU-1", Name: "Shirt", CategoryID: "cat-1", Price: 10, CreatedBy: "admin-1"}
	}

	tests := []struct {
		name    string
		mutate  func(*CreateProductRequest)
		wantErr bool
	}{
		{name: "valid request passes", mutate: func(r *CreateProductRequest) {}, wantErr: false},
		{name: "missing sku fails", mutate: func(r *CreateProductRequest) { r.SKU = "" }, wantErr: true},
		{name: "missing name fails", mutate: func(r *CreateProductRequest) { r.Name = "" }, wantErr: true},
		{name: "missing category_id fails", mutate: func(r *CreateProductRequest) { r.CategoryID = "" }, wantErr: true},
		{name: "missing created_by fails", mutate: func(r *CreateProductRequest) { r.CreatedBy = "" }, wantErr: true},
		{name: "zero price fails (gt=0)", mutate: func(r *CreateProductRequest) { r.Price = 0 }, wantErr: true},
		{name: "negative price fails (gt=0)", mutate: func(r *CreateProductRequest) { r.Price = -1 }, wantErr: true},
		{name: "slug is optional", mutate: func(r *CreateProductRequest) { r.Slug = "" }, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base()
			tt.mutate(&req)

			err := validate.Struct(req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// UpdateProductRequest uses pointers so the service layer can tell "field
// omitted, leave it alone" apart from "field explicitly set to its zero
// value" -- e.g. marking a product free (Price: 0) versus not touching Price.
func TestUpdateProductRequest_OptionalFieldsDistinguishAbsentFromZero(t *testing.T) {
	raw := []byte(`{"price":0,"name":"Renamed"}`)

	var req UpdateProductRequest
	assert.NoError(t, json.Unmarshal(raw, &req))

	if assert.NotNil(t, req.Price) {
		assert.Equal(t, 0.0, *req.Price, "price was explicitly provided as 0, so it must be applied, not skipped")
	}
	if assert.NotNil(t, req.Name) {
		assert.Equal(t, "Renamed", *req.Name)
	}
	assert.Nil(t, req.Brand, "an omitted field must stay nil so the service layer leaves it untouched")
	assert.Nil(t, req.CompareAtPrice)
	assert.Nil(t, req.Dimensions)
}

func TestProductStatus_WireValuesAreStable(t *testing.T) {
	tests := []struct {
		status ProductStatus
		want   string
	}{
		{ProductStatusDraft, `"draft"`},
		{ProductStatusActive, `"active"`},
		{ProductStatusInactive, `"inactive"`},
		{ProductStatusArchived, `"archived"`},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			raw, err := json.Marshal(tt.status)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(raw))
		})
	}
}

func TestProductVariant_JSONRoundTrip(t *testing.T) {
	variant := ProductVariant{
		ID:      "var-1",
		SKU:     "SKU-1-L-RED",
		Name:    "Large / Red",
		Price:   550,
		Options: map[string]string{"Size": "L", "Color": "Red"},
	}

	raw, err := json.Marshal(variant)
	assert.NoError(t, err)

	var decoded ProductVariant
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, variant, decoded)
	assert.Equal(t, "L", decoded.Options["Size"])
}

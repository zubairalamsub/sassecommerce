package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCategory_ToResponse_MapsFieldsAndHexEncodesID(t *testing.T) {
	id := primitive.NewObjectID()
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	parent := "parent-category-id"

	cat := Category{
		ID:        id,
		TenantID:  "tenant-1",
		Name:      "Shoes",
		Slug:      "shoes",
		ParentID:  &parent,
		SortOrder: 2,
		Status:    CategoryStatusActive,
		CreatedBy: "admin-1",
		CreatedAt: now,
		UpdatedAt: now,
	}

	resp := cat.ToResponse()

	assert.Equal(t, id.Hex(), resp.ID, "the wire ID must be the ObjectID's hex encoding, not its raw bytes")
	assert.Equal(t, "tenant-1", resp.TenantID)
	assert.Equal(t, "Shoes", resp.Name)
	assert.Same(t, &parent, resp.ParentID, "ParentID is not deep-copied")
	assert.Equal(t, CategoryStatusActive, resp.Status)
}

// CategoryResponse has no field for DeletedAt at all -- this pins that a
// soft-deleted category's response never carries the internal marker, so a
// future field addition to CategoryResponse cannot accidentally leak it.
func TestCategory_ToResponse_OmitsDeletedAt(t *testing.T) {
	deletedAt := time.Now().UTC()
	cat := Category{ID: primitive.NewObjectID(), Name: "Discontinued", DeletedAt: &deletedAt}

	raw, err := json.Marshal(cat.ToResponse())
	assert.NoError(t, err)

	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	assert.NotContains(t, decoded, "deleted_at", "ToResponse must never leak the internal soft-delete marker to API clients")
}

func TestCategory_JSON_WireFieldNames(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	parent := "parent-id-1"
	cat := Category{
		ID:          primitive.NewObjectID(),
		TenantID:    "tenant-1",
		Name:        "Shoes",
		Slug:        "shoes",
		Description: "Footwear",
		ParentID:    &parent,
		Image:       "https://cdn.example.com/img.png",
		Icon:        "shoe-icon",
		SortOrder:   2,
		Status:      CategoryStatusActive,
		CreatedBy:   "admin-1",
		UpdatedBy:   "admin-2",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	raw, err := json.Marshal(cat)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &decoded))

	for _, key := range []string{
		"id", "tenant_id", "name", "slug", "description", "parent_id",
		"image", "icon", "sort_order", "status", "created_by", "updated_by",
		"created_at", "updated_at",
	} {
		assert.Contains(t, decoded, key, "expected the wire field %q", key)
	}
	assert.Equal(t, cat.ID.Hex(), decoded["id"])
	assert.Equal(t, "active", decoded["status"])
	// DeletedAt is nil and tagged omitempty on a pointer, so it correctly
	// disappears (contrast with product.go's SEO/Dimensions, where the same
	// tag on a non-pointer struct field is a no-op -- see product_test.go).
	assert.NotContains(t, decoded, "deleted_at")
}

// TenantID is tagged json:"-" specifically so a client cannot inject another
// tenant's ID through the request body; the handler sets it from the
// authenticated JWT instead. This is the regression test for that guarantee.
func TestCreateCategoryRequest_TenantIDCannotBeSpoofedFromRequestBody(t *testing.T) {
	raw := []byte(`{"tenant_id":"attacker-tenant","name":"Shoes","slug":"shoes","created_by":"u1"}`)

	var req CreateCategoryRequest
	assert.NoError(t, json.Unmarshal(raw, &req))

	assert.Empty(t, req.TenantID, "a client-supplied tenant_id in the body must never populate TenantID, or a request could write into another tenant's catalog")
	assert.Equal(t, "Shoes", req.Name)
	assert.Equal(t, "u1", req.CreatedBy)
}

// UpdateCategoryRequest uses pointers so the service layer can tell "field
// omitted, leave it alone" apart from "field explicitly set to its zero
// value" -- this is what makes partial updates safe.
func TestUpdateCategoryRequest_OptionalFieldsDistinguishAbsentFromZero(t *testing.T) {
	raw := []byte(`{"name":"New Name","sort_order":0}`)

	var req UpdateCategoryRequest
	assert.NoError(t, json.Unmarshal(raw, &req))

	if assert.NotNil(t, req.Name) {
		assert.Equal(t, "New Name", *req.Name)
	}
	if assert.NotNil(t, req.SortOrder) {
		assert.Equal(t, 0, *req.SortOrder, "sort_order was explicitly provided as 0, so it must be applied, not skipped")
	}
	assert.Nil(t, req.Slug, "a field omitted from the request body must stay nil so the service layer knows not to touch it")
	assert.Nil(t, req.Description)
	assert.Nil(t, req.ParentID)
}

func TestCategoryStatus_WireValuesAreStable(t *testing.T) {
	tests := []struct {
		status CategoryStatus
		want   string
	}{
		{CategoryStatusActive, `"active"`},
		{CategoryStatusInactive, `"inactive"`},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			raw, err := json.Marshal(tt.status)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(raw))
		})
	}
}

// Gin's binding engine is exactly validator.New() with SetTagName("binding")
// (see gin-gonic/gin/binding/default_validator.go) -- mirroring that here
// exercises the real validation contract these tags declare, not just their
// presence.
func newBindingValidator() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding")
	return v
}

func TestCreateCategoryRequest_BindingValidation(t *testing.T) {
	validate := newBindingValidator()

	base := func() CreateCategoryRequest {
		return CreateCategoryRequest{Name: "Shoes", Slug: "shoes", CreatedBy: "admin-1"}
	}

	tests := []struct {
		name    string
		mutate  func(*CreateCategoryRequest)
		wantErr bool
	}{
		{name: "valid request passes", mutate: func(r *CreateCategoryRequest) {}, wantErr: false},
		{name: "missing name fails", mutate: func(r *CreateCategoryRequest) { r.Name = "" }, wantErr: true},
		{name: "missing slug fails", mutate: func(r *CreateCategoryRequest) { r.Slug = "" }, wantErr: true},
		{name: "missing created_by fails", mutate: func(r *CreateCategoryRequest) { r.CreatedBy = "" }, wantErr: true},
		{name: "description is optional", mutate: func(r *CreateCategoryRequest) { r.Description = "" }, wantErr: false},
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

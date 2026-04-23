package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultPage is the default page number
	DefaultPage = 1
	// DefaultPageSize is the default page size
	DefaultPageSize = 20
	// MaxPageSize is the maximum allowed page size
	MaxPageSize = 100
)

// Params holds pagination parameters
type Params struct {
	Page     int
	PageSize int
	Offset   int
	SortBy   string
	SortDir  string
}

// GetPaginationParams extracts pagination parameters from request
func GetPaginationParams(c *gin.Context) Params {
	page := getIntParam(c, "page", DefaultPage)
	if page < 1 {
		page = DefaultPage
	}

	pageSize := getIntParam(c, "page_size", DefaultPageSize)
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	offset := (page - 1) * pageSize

	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDir := c.DefaultQuery("sort_dir", "desc")
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	return Params{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
		SortBy:   sortBy,
		SortDir:  sortDir,
	}
}

// GetOrderBy returns ORDER BY clause for SQL
func (p Params) GetOrderBy() string {
	return p.SortBy + " " + p.SortDir
}

// Helper to get integer parameter
func getIntParam(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// CalculateTotalPages calculates total pages from total items
func CalculateTotalPages(totalItems int64, pageSize int) int {
	if pageSize == 0 {
		return 0
	}

	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize != 0 {
		totalPages++
	}

	return totalPages
}

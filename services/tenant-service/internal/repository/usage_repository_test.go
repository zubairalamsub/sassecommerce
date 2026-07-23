package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ecommerce/tenant-service/internal/models"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func init() {
	// The production query uses Postgres' octet_length(); the SQLite bundled
	// with mattn/go-sqlite3 predates its 3.43 builtin. Register it: len() of a
	// Go string is its byte length, which is exactly octet_length semantics.
	sql.Register("sqlite3_octet_length", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterFunc("octet_length", func(s string) int64 { return int64(len(s)) }, true)
		},
	})
}

func setupUsageDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(&sqlite.Dialector{
		DriverName: "sqlite3_octet_length",
		DSN:        ":memory:",
	}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Tenant{}, &models.AuditLog{}))
	return db
}

func TestGetTenantUsage_AggregatesPerTenant(t *testing.T) {
	db := setupUsageDB(t)
	repo := NewUsageRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	require.NoError(t, db.Create(&models.Tenant{ID: "t1", Name: "One", Slug: "one", Email: "one@x.com", Status: models.StatusActive, Tier: models.TierFree}).Error)
	require.NoError(t, db.Create(&models.Tenant{ID: "t2", Name: "Two", Slug: "two", Email: "two@x.com", Status: models.StatusActive, Tier: models.TierFree}).Error)

	// Two logs for t1 (one old, one recent), none for t2.
	require.NoError(t, db.Create(&models.AuditLog{ID: "a1", TenantID: "t1", Action: "CREATE", Resource: "tenant", CreatedAt: now.AddDate(0, 0, -60)}).Error)
	require.NoError(t, db.Create(&models.AuditLog{ID: "a2", TenantID: "t1", Action: "UPDATE", Resource: "tenant", CreatedAt: now.AddDate(0, 0, -1)}).Error)
	// A log with empty tenant_id must be excluded entirely.
	require.NoError(t, db.Create(&models.AuditLog{ID: "a3", TenantID: "", Action: "READ", Resource: "tenant", CreatedAt: now}).Error)

	rows, err := repo.GetTenantUsage(ctx, nil)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byID := map[string]TenantUsageRow{}
	for _, r := range rows {
		byID[r.TenantID] = r
	}
	assert.Equal(t, int64(2), byID["t1"].AuditLogCount)
	assert.Greater(t, byID["t1"].AuditLogBytesEstimate, int64(0))
	// Tenants with no logs still appear, zeroed.
	assert.Equal(t, int64(0), byID["t2"].AuditLogCount)
	assert.Equal(t, int64(0), byID["t2"].AuditLogBytesEstimate)
}

func TestGetTenantUsage_WindowBoundsAggregate(t *testing.T) {
	db := setupUsageDB(t)
	repo := NewUsageRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	require.NoError(t, db.Create(&models.Tenant{ID: "t1", Name: "One", Slug: "one", Email: "one@x.com", Status: models.StatusActive, Tier: models.TierFree}).Error)
	require.NoError(t, db.Create(&models.AuditLog{ID: "old", TenantID: "t1", Action: "CREATE", Resource: "tenant", CreatedAt: now.AddDate(0, 0, -60)}).Error)
	require.NoError(t, db.Create(&models.AuditLog{ID: "new", TenantID: "t1", Action: "UPDATE", Resource: "tenant", CreatedAt: now.AddDate(0, 0, -1)}).Error)

	// 30-day window: only the recent log counts.
	since := now.AddDate(0, 0, -30)
	rows, err := repo.GetTenantUsage(ctx, &since)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(1), rows[0].AuditLogCount)

	// nil window: all-time, both logs count.
	rows, err = repo.GetTenantUsage(ctx, nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(2), rows[0].AuditLogCount)
}

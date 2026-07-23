package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// TenantUsageRow represents one tenant's aggregated usage stats.
// The struct fields are mapped from the SQL aggregate query below.
type TenantUsageRow struct {
	TenantID              string    `json:"tenant_id" gorm:"column:tenant_id"`
	TenantName            string    `json:"tenant_name" gorm:"column:tenant_name"`
	TenantSlug            string    `json:"tenant_slug" gorm:"column:tenant_slug"`
	Tier                  string    `json:"tier" gorm:"column:tier"`
	Status                string    `json:"status" gorm:"column:status"`
	CreatedAt             time.Time `json:"created_at" gorm:"column:created_at"`
	AuditLogCount         int64     `json:"audit_log_count" gorm:"column:audit_log_count"`
	AuditLogBytesEstimate int64     `json:"audit_log_bytes_estimate" gorm:"column:audit_log_bytes_estimate"`
}

// UsageRepository exposes queries that aggregate usage information from
// tables this service owns directly (tenants + audit_logs).
type UsageRepository interface {
	// GetTenantUsage aggregates audit-log usage per tenant. A non-nil since
	// bounds the aggregate to logs created at/after that instant (cheap: the
	// audit_logs.created_at index covers it); nil means all-time.
	GetTenantUsage(ctx context.Context, since *time.Time) ([]TenantUsageRow, error)
}

type usageRepository struct {
	db *gorm.DB
}

func NewUsageRepository(db *gorm.DB) UsageRepository {
	return &usageRepository{db: db}
}

// GetTenantUsage returns one row per tenant (every tenant included even with
// zero audit logs) with an approximate byte-size of audit log data per tenant.
//
// The byte estimate is intentionally cheap: it sums octet_length of the most
// representative string columns (action, resource, resource_id, old_value,
// new_value, metadata, request_body). It is *not* an exact on-disk size — it
// is an indicator of how much audit content a tenant is generating.
func (r *usageRepository) GetTenantUsage(ctx context.Context, since *time.Time) ([]TenantUsageRow, error) {
	var rows []TenantUsageRow

	// Optional time bound on the audit_logs aggregate. Without it the subquery
	// scans the whole table, which grows without limit; with it the
	// audit_logs.created_at index keeps the aggregate cheap.
	sinceClause := ""
	args := []interface{}{}
	if since != nil {
		sinceClause = "AND created_at >= ?"
		args = append(args, *since)
	}

	query := `
		SELECT
			t.id                          AS tenant_id,
			t.name                        AS tenant_name,
			t.slug                        AS tenant_slug,
			t.tier                        AS tier,
			t.status                      AS status,
			t.created_at                  AS created_at,
			COALESCE(a.audit_log_count, 0)         AS audit_log_count,
			COALESCE(a.audit_log_bytes_estimate, 0) AS audit_log_bytes_estimate
		FROM tenants t
		LEFT JOIN (
			SELECT
				tenant_id,
				COUNT(*) AS audit_log_count,
				SUM(
					octet_length(COALESCE(action, '')) +
					octet_length(COALESCE(resource, '')) +
					octet_length(COALESCE(resource_id, '')) +
					octet_length(COALESCE(old_value, '')) +
					octet_length(COALESCE(new_value, '')) +
					octet_length(COALESCE(metadata, '')) +
					octet_length(COALESCE(request_body, ''))
				) AS audit_log_bytes_estimate
			FROM audit_logs
			WHERE tenant_id IS NOT NULL AND tenant_id <> '' ` + sinceClause + `
			GROUP BY tenant_id
		) a ON a.tenant_id = t.id
		WHERE t.deleted_at IS NULL
		ORDER BY t.created_at DESC
	`

	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

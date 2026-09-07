-- Read-only support-lookup access for Grafana.
--
-- Run with:  make grafana-lookup-setup
--
-- NOT in /docker-entrypoint-initdb.d, deliberately. The tables this reads are
-- created by the services themselves at startup (GORM AutoMigrate in
-- user-service, createTable() in order-service), which happens long after
-- Postgres finishes initialising. Granting on them at init time would fail on
-- tables that do not exist yet. This script is idempotent, so it is safe to
-- re-run after a schema change or on an existing deployment.
--
-- Two things it deliberately does NOT do:
--
--   * It does not use ALTER DEFAULT PRIVILEGES. That would hand grafana_ro
--     SELECT on every future table, including users.password_hash.
--   * It does not grant on base tables at all. Grafana reads views that list
--     their columns explicitly, so a column added to users later is invisible
--     to Grafana until someone chooses to expose it.

\set ON_ERROR_STOP on

-- ---------------------------------------------------------------------------
-- Role
-- ---------------------------------------------------------------------------
-- \gexec rather than a DO block: psql does not interpolate :'variables'
-- inside dollar-quoted strings, so the password in a DO $$ ... $$ body would
-- arrive as the literal text ":'grafana_ro_password'". Here the interpolation
-- happens in plain SQL, and format(%L) quotes the value for the statement
-- \gexec then runs.

-- Create the role if it is missing. Produces no row (and so runs nothing) when
-- it already exists.
SELECT 'CREATE ROLE grafana_ro LOGIN'
 WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'grafana_ro')
\gexec

-- Set the password either way, so re-running this rotates it.
SELECT format('ALTER ROLE grafana_ro LOGIN PASSWORD %L', :'grafana_ro_password')
\gexec

-- Belt and braces: this role must never be able to write, even if someone
-- later grants it a table by hand.
ALTER ROLE grafana_ro SET default_transaction_read_only = on;
ALTER ROLE grafana_ro SET statement_timeout = '15s';

-- One residual, measured rather than assumed. Postgres grants CONNECT on
-- every database to PUBLIC by default, so grafana_ro can open a connection
-- to payment_db and the rest. It can read nothing there: verified against
-- a payment_db containing a transactions table, where SELECT was denied and
-- information_schema.tables returned zero rows, because that view filters by
-- privilege. Closing it properly means REVOKE CONNECT ... FROM PUBLIC on all
-- fourteen databases, which is a change to shared infrastructure to remove an
-- exposure that leaks no data, so it is left alone and written down instead.

-- ---------------------------------------------------------------------------
-- order_db
-- ---------------------------------------------------------------------------
\c order_db

CREATE OR REPLACE VIEW support_order_lookup AS
SELECT
    o.id              AS order_id,
    o.tenant_id,
    o.customer_id,
    o.status,
    o.total_amount,
    o.currency,
    o.payment_id,
    o.reservation_id,
    o.tracking_number,
    o.carrier,
    o.shipping_city,
    o.shipping_country,
    o.created_at,
    o.updated_at,
    o.version
FROM order_read_model o;

COMMENT ON VIEW support_order_lookup IS
  'Read-only support lookup. Column list is explicit so a new column on '
  'order_read_model is not silently exposed to Grafana.';

-- The order's line items, for confirming you have the right order.
CREATE OR REPLACE VIEW support_order_items AS
SELECT
    i.order_id,
    i.sku,
    i.name,
    i.quantity,
    i.unit_price,
    i.product_id
FROM order_item_read_model i;

-- Full event history for one order. This is what answers "what happened to
-- it", which the read model cannot: the read model holds current state only.
CREATE OR REPLACE VIEW support_order_history AS
SELECT
    e.aggregate_id AS order_id,
    e.version,
    e.event_type,
    e.timestamp,
    e.event_data
FROM events e;

GRANT CONNECT ON DATABASE order_db TO grafana_ro;
GRANT USAGE ON SCHEMA public TO grafana_ro;
GRANT SELECT ON support_order_lookup, support_order_items, support_order_history TO grafana_ro;

-- Searching by tracking number or payment id would otherwise scan the table;
-- both are how support actually arrives at an order (a carrier query, a
-- gateway dispute). The existing indexes cover tenant_id and customer_id.
CREATE INDEX IF NOT EXISTS idx_order_tracking_number
    ON order_read_model (tracking_number) WHERE tracking_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_order_payment_id
    ON order_read_model (payment_id) WHERE payment_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- user_db
-- ---------------------------------------------------------------------------
\c user_db

-- password_hash is absent from this list on purpose, and that is the whole
-- reason Grafana reads a view rather than the table.
CREATE OR REPLACE VIEW support_customer_lookup AS
SELECT
    u.id            AS customer_id,
    u.tenant_id,
    u.email,
    u.username,
    u.first_name,
    u.last_name,
    u.phone,
    u.status,
    u.role,
    u.email_verified,
    u.last_login_at,
    u.created_at,
    u.deleted_at
FROM users u;

COMMENT ON VIEW support_customer_lookup IS
  'Read-only support lookup. Excludes password_hash; keep it that way.';

GRANT CONNECT ON DATABASE user_db TO grafana_ro;
GRANT USAGE ON SCHEMA public TO grafana_ro;
GRANT SELECT ON support_customer_lookup TO grafana_ro;

-- Phone and name are search entry points for support and are not indexed by
-- the application, which only ever looks users up by email or id.
CREATE INDEX IF NOT EXISTS idx_users_phone
    ON users (phone) WHERE phone <> '';
CREATE INDEX IF NOT EXISTS idx_users_name_lower
    ON users (lower(first_name), lower(last_name));

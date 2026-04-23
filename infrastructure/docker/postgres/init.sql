-- Create databases for each microservice

-- Tenant Service Database
CREATE DATABASE tenant_db;

-- User Service Database
CREATE DATABASE user_db;

-- Product Service Database
CREATE DATABASE product_db;

-- Order Service Database
CREATE DATABASE order_db;

-- Inventory Service Database
CREATE DATABASE inventory_db;

-- Payment Service Database
CREATE DATABASE payment_db;

-- Shipping Service Database
CREATE DATABASE shipping_db;

-- Promotion Service Database
CREATE DATABASE promotion_db;

-- Vendor Service Database
CREATE DATABASE vendor_db;

-- Analytics Service Database
CREATE DATABASE analytics_db;

-- Recommendation Service Database
CREATE DATABASE recommendation_db;

-- Configuration Service Database
CREATE DATABASE config_db;

-- Enable UUID extension on all databases
\c tenant_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c user_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c product_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c order_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c inventory_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c payment_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c shipping_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c promotion_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c vendor_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c analytics_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c recommendation_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c config_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

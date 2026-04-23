package seed

import (
	"context"

	"github.com/ecommerce/config-service/internal/models"
	"github.com/ecommerce/config-service/internal/repository"
	"github.com/ecommerce/config-service/internal/service"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type configDef struct {
	Namespace   string
	Key         string
	Value       string
	ValueType   string
	Description string
}

func SeedDefaults(ctx context.Context, svc service.ConfigService, logger *logrus.Logger) {
	defaults := getAllDefaults()

	for _, d := range defaults {
		_, err := svc.SetConfig(ctx, &models.SetConfigRequest{
			Namespace:   d.Namespace,
			Key:         d.Key,
			Value:       d.Value,
			ValueType:   d.ValueType,
			Description: d.Description,
			Environment: "all",
			UpdatedBy:   "seed",
		})
		if err != nil {
			// Config already exists, skip
			continue
		}
	}

	logger.Infof("Seeded %d default configuration entries", len(defaults))
}

// SeedDefaultMenus seeds default menu structures for a given tenant
func SeedDefaultMenus(ctx context.Context, menuRepo repository.MenuRepository, tenantID string, logger *logrus.Logger) {
	// Check if menus already exist for this tenant
	existing, _ := menuRepo.ListMenus(ctx, tenantID)
	if len(existing) > 0 {
		return
	}

	// Main Navigation (header)
	mainNavID := uuid.New().String()
	menuRepo.CreateMenu(ctx, &models.Menu{
		ID: mainNavID, TenantID: tenantID,
		Name: "Main Navigation", Slug: "main-navigation",
		Location: "header", Description: "Primary site navigation", IsActive: true,
	})

	mainItems := []struct {
		Label    string
		URL      string
		Icon     string
		Position int
		Children []struct {
			Label    string
			URL      string
			Position int
		}
	}{
		{Label: "Home", URL: "/", Icon: "home", Position: 0, Children: nil},
		{Label: "Shop", URL: "/shop", Icon: "shopping-bag", Position: 1, Children: []struct {
			Label    string
			URL      string
			Position int
		}{
			{Label: "All Products", URL: "/shop/all", Position: 0},
			{Label: "New Arrivals", URL: "/shop/new", Position: 1},
			{Label: "Best Sellers", URL: "/shop/best-sellers", Position: 2},
			{Label: "On Sale", URL: "/shop/sale", Position: 3},
		}},
		{Label: "Categories", URL: "/categories", Icon: "grid", Position: 2, Children: []struct {
			Label    string
			URL      string
			Position int
		}{
			{Label: "Electronics", URL: "/categories/electronics", Position: 0},
			{Label: "Clothing", URL: "/categories/clothing", Position: 1},
			{Label: "Home & Garden", URL: "/categories/home-garden", Position: 2},
			{Label: "Sports", URL: "/categories/sports", Position: 3},
		}},
		{Label: "Deals", URL: "/deals", Icon: "tag", Position: 3, Children: nil},
	}

	for _, mi := range mainItems {
		parentID := uuid.New().String()
		menuRepo.CreateMenuItem(ctx, &models.MenuItem{
			ID: parentID, MenuID: mainNavID, Label: mi.Label,
			URL: mi.URL, Icon: mi.Icon, Target: "_self",
			Position: mi.Position, IsActive: true,
		})
		for _, child := range mi.Children {
			menuRepo.CreateMenuItem(ctx, &models.MenuItem{
				ID: uuid.New().String(), MenuID: mainNavID, ParentID: parentID,
				Label: child.Label, URL: child.URL, Target: "_self",
				Position: child.Position, IsActive: true,
			})
		}
	}

	// Footer Menu
	footerID := uuid.New().String()
	menuRepo.CreateMenu(ctx, &models.Menu{
		ID: footerID, TenantID: tenantID,
		Name: "Footer Links", Slug: "footer-links",
		Location: "footer", Description: "Footer navigation columns", IsActive: true,
	})

	footerColumns := []struct {
		Label    string
		Position int
		Links    []struct {
			Label    string
			URL      string
			Position int
		}
	}{
		{Label: "About", Position: 0, Links: []struct {
			Label    string
			URL      string
			Position int
		}{
			{Label: "About Us", URL: "/about", Position: 0},
			{Label: "Careers", URL: "/careers", Position: 1},
			{Label: "Press", URL: "/press", Position: 2},
		}},
		{Label: "Support", Position: 1, Links: []struct {
			Label    string
			URL      string
			Position int
		}{
			{Label: "Help Center", URL: "/help", Position: 0},
			{Label: "Contact Us", URL: "/contact", Position: 1},
			{Label: "Returns", URL: "/returns", Position: 2},
			{Label: "Shipping Info", URL: "/shipping", Position: 3},
		}},
		{Label: "Legal", Position: 2, Links: []struct {
			Label    string
			URL      string
			Position int
		}{
			{Label: "Privacy Policy", URL: "/privacy", Position: 0},
			{Label: "Terms of Service", URL: "/terms", Position: 1},
			{Label: "Cookie Policy", URL: "/cookies", Position: 2},
		}},
	}

	for _, col := range footerColumns {
		colID := uuid.New().String()
		menuRepo.CreateMenuItem(ctx, &models.MenuItem{
			ID: colID, MenuID: footerID, Label: col.Label,
			URL: "", Target: "_self", Position: col.Position, IsActive: true,
		})
		for _, link := range col.Links {
			menuRepo.CreateMenuItem(ctx, &models.MenuItem{
				ID: uuid.New().String(), MenuID: footerID, ParentID: colID,
				Label: link.Label, URL: link.URL, Target: "_self",
				Position: link.Position, IsActive: true,
			})
		}
	}

	// Sidebar Menu
	sidebarID := uuid.New().String()
	menuRepo.CreateMenu(ctx, &models.Menu{
		ID: sidebarID, TenantID: tenantID,
		Name: "Account Sidebar", Slug: "account-sidebar",
		Location: "sidebar", Description: "User account navigation", IsActive: true,
	})

	sidebarItems := []struct {
		Label    string
		URL      string
		Icon     string
		Position int
	}{
		{Label: "My Profile", URL: "/account/profile", Icon: "user", Position: 0},
		{Label: "Orders", URL: "/account/orders", Icon: "package", Position: 1},
		{Label: "Wishlist", URL: "/account/wishlist", Icon: "heart", Position: 2},
		{Label: "Addresses", URL: "/account/addresses", Icon: "map-pin", Position: 3},
		{Label: "Payment Methods", URL: "/account/payments", Icon: "credit-card", Position: 4},
		{Label: "Settings", URL: "/account/settings", Icon: "settings", Position: 5},
	}

	for _, si := range sidebarItems {
		menuRepo.CreateMenuItem(ctx, &models.MenuItem{
			ID: uuid.New().String(), MenuID: sidebarID, Label: si.Label,
			URL: si.URL, Icon: si.Icon, Target: "_self",
			Position: si.Position, IsActive: true,
		})
	}

	// Mobile Menu
	mobileID := uuid.New().String()
	menuRepo.CreateMenu(ctx, &models.Menu{
		ID: mobileID, TenantID: tenantID,
		Name: "Mobile Navigation", Slug: "mobile-navigation",
		Location: "mobile", Description: "Mobile hamburger menu", IsActive: true,
	})

	mobileItems := []struct {
		Label    string
		URL      string
		Icon     string
		Position int
	}{
		{Label: "Home", URL: "/", Icon: "home", Position: 0},
		{Label: "Shop", URL: "/shop", Icon: "shopping-bag", Position: 1},
		{Label: "Categories", URL: "/categories", Icon: "grid", Position: 2},
		{Label: "Cart", URL: "/cart", Icon: "shopping-cart", Position: 3},
		{Label: "Account", URL: "/account", Icon: "user", Position: 4},
	}

	for _, mi := range mobileItems {
		menuRepo.CreateMenuItem(ctx, &models.MenuItem{
			ID: uuid.New().String(), MenuID: mobileID, Label: mi.Label,
			URL: mi.URL, Icon: mi.Icon, Target: "_self",
			Position: mi.Position, IsActive: true,
		})
	}

	logger.Infof("Seeded default menus for tenant %s (4 menus with items)", tenantID)
}

func getAllDefaults() []configDef {
	return flatten(
		globalDefaults(),
		kafkaDefaults(),
		businessVendorDefaults(),
		businessLoyaltyDefaults(),
		businessShippingDefaults(),
		businessRecommendationDefaults(),
		businessPromotionDefaults(),
		tenantPlanDefaults(),
		tenantFeatureFlagDefaults(),
		searchDefaults(),
		cartDefaults(),
		analyticsDefaults(),
		notificationDefaults(),
		paymentDefaults(),
		serverDefaults(),
		reviewDefaults(),
	)
}

func flatten(groups ...[]configDef) []configDef {
	var all []configDef
	for _, g := range groups {
		all = append(all, g...)
	}
	return all
}

func globalDefaults() []configDef {
	return []configDef{
		// Pagination
		{"global", "pagination.default_page_size", "20", "number", "Default number of items per page across all services"},
		{"global", "pagination.max_page_size", "100", "number", "Maximum allowed page size"},

		// Server timeouts
		{"global", "server.read_timeout_seconds", "15", "number", "HTTP server read timeout in seconds"},
		{"global", "server.write_timeout_seconds", "15", "number", "HTTP server write timeout in seconds"},
		{"global", "server.idle_timeout_seconds", "60", "number", "HTTP server idle timeout in seconds"},
		{"global", "server.shutdown_timeout_seconds", "30", "number", "Graceful shutdown timeout in seconds"},

		// Database defaults
		{"global", "database.host", "localhost", "string", "Default PostgreSQL host"},
		{"global", "database.port", "5432", "number", "Default PostgreSQL port"},
		{"global", "database.user", "postgres", "string", "Default PostgreSQL user"},
		{"global", "database.sslmode", "disable", "string", "Default PostgreSQL SSL mode"},

		// Redis defaults
		{"global", "redis.host", "localhost", "string", "Default Redis host"},
		{"global", "redis.port", "6379", "number", "Default Redis port"},
		{"global", "redis.db", "0", "number", "Default Redis database index"},

		// Locale & Currency
		{"global", "currency.default", "BDT", "string", "Default currency code (Bangladeshi Taka)"},
		{"global", "currency.symbol", "৳", "string", "Default currency symbol"},
		{"global", "currency.decimal_places", "2", "number", "Currency decimal places"},
		{"global", "locale.default_language", "bn", "string", "Default language code (Bangla)"},
		{"global", "locale.supported_languages", `["bn","en"]`, "json", "Supported language codes"},
		{"global", "locale.default_timezone", "Asia/Dhaka", "string", "Default timezone (UTC+6)"},
		{"global", "locale.date_format", "DD/MM/YYYY", "string", "Default date display format"},
		{"global", "locale.phone_country_code", "+880", "string", "Default phone country code (Bangladesh)"},
		{"global", "locale.default_country", "BD", "string", "Default country ISO code"},
		{"global", "address.format", `["address_line1","address_line2","upazila","district","division","postal_code"]`, "json", "Address field order for Bangladesh"},
		{"global", "address.divisions", `["Dhaka","Chittagong","Rajshahi","Khulna","Barisal","Sylhet","Rangpur","Mymensingh"]`, "json", "Bangladesh divisions list"},
		{"global", "tax.default_vat_rate", "15", "number", "Default VAT rate percentage (Bangladesh standard)"},
		{"global", "tax.vat_registration_required", "true", "boolean", "Whether VAT registration is required for vendors"},
	}
}

func kafkaDefaults() []configDef {
	return []configDef{
		// Kafka producer settings
		{"kafka", "producer.batch_size", "100", "number", "Kafka producer batch size"},
		{"kafka", "producer.batch_timeout_ms", "10", "number", "Kafka producer batch timeout in milliseconds"},
		{"kafka", "producer.required_acks", "1", "number", "Kafka required acknowledgments (0=none, 1=leader, -1=all)"},

		// Topic names
		{"kafka", "topics.order_events", "order-events", "string", "Kafka topic for order lifecycle events"},
		{"kafka", "topics.product_events", "product-events", "string", "Kafka topic for product CRUD events"},
		{"kafka", "topics.vendor_events", "vendor-events", "string", "Kafka topic for vendor status change events"},
		{"kafka", "topics.cart_events", "cart-events", "string", "Kafka topic for cart update events"},
		{"kafka", "topics.review_events", "review-events", "string", "Kafka topic for review submission events"},
		{"kafka", "topics.promotion_events", "promotion-events", "string", "Kafka topic for promotion and coupon events"},
		{"kafka", "topics.tenant_events", "tenant-events", "string", "Kafka topic for tenant lifecycle events"},
		{"kafka", "topics.price_events", "price-events", "string", "Kafka topic for product price change events"},
		{"kafka", "topics.inventory_events", "inventory-events", "string", "Kafka topic for stock level change events"},
		{"kafka", "topics.notification_events", "notification-events", "string", "Kafka topic for notification dispatch events"},
		{"kafka", "topics.shipping_events", "shipping-events", "string", "Kafka topic for shipment tracking events"},
		{"kafka", "topics.payment_events", "payment-events", "string", "Kafka topic for payment transaction events"},
		{"kafka", "topics.user_events", "user-events", "string", "Kafka topic for user registration/profile events"},

		// Consumer group IDs
		{"kafka", "consumer_groups.vendor_service", "vendor-service", "string", "Kafka consumer group for vendor service"},
		{"kafka", "consumer_groups.promotion_service", "promotion-service", "string", "Kafka consumer group for promotion service"},
		{"kafka", "consumer_groups.cart_service", "cart-service", "string", "Kafka consumer group for cart service"},
		{"kafka", "consumer_groups.search_service", "search-service", "string", "Kafka consumer group for search service"},
		{"kafka", "consumer_groups.analytics_service", "analytics-service", "string", "Kafka consumer group for analytics service"},
		{"kafka", "consumer_groups.recommendation_service", "recommendation-service", "string", "Kafka consumer group for recommendation service"},
		{"kafka", "consumer_groups.notification_service", "notification-service", "string", "Kafka consumer group for notification service"},
		{"kafka", "consumer_groups.order_service", "order-service-projections", "string", "Kafka consumer group for order service projections"},
	}
}

func businessVendorDefaults() []configDef {
	return []configDef{
		{"business.vendor", "default_commission_rate", "10", "number", "Default vendor commission percentage on orders"},
		{"business.vendor", "min_commission_rate", "1", "number", "Minimum allowed vendor commission rate"},
		{"business.vendor", "max_commission_rate", "50", "number", "Maximum allowed vendor commission rate"},
		{"business.vendor", "auto_approve_vendors", "false", "boolean", "Whether new vendors are auto-approved or require manual review"},
		{"business.vendor", "status_transitions", `{"pending":["approved","rejected"],"approved":["suspended"],"suspended":["approved"],"rejected":["pending"]}`, "json", "Allowed vendor status transitions map"},
	}
}

func businessLoyaltyDefaults() []configDef {
	return []configDef{
		{"business.loyalty", "earn_rate_per_dollar", "1", "number", "Loyalty points earned per dollar spent"},
		{"business.loyalty", "tier_bronze_threshold", "0", "number", "Minimum points for Bronze tier"},
		{"business.loyalty", "tier_silver_threshold", "1000", "number", "Minimum points for Silver tier"},
		{"business.loyalty", "tier_gold_threshold", "5000", "number", "Minimum points for Gold tier"},
		{"business.loyalty", "tier_platinum_threshold", "10000", "number", "Minimum points for Platinum tier"},
		{"business.loyalty", "points_expiry_days", "365", "number", "Days before unused loyalty points expire"},
		{"business.loyalty", "max_redeem_percentage", "50", "number", "Maximum percentage of order payable with points"},
	}
}

func businessShippingDefaults() []configDef {
	return []configDef{
		// Base rates by carrier (BDT)
		{"business.shipping", "carrier.default.base_rate", "60", "number", "Default carrier base shipping rate (BDT)"},
		{"business.shipping", "carrier.pathao.base_rate", "60", "number", "Pathao Courier base rate (BDT)"},
		{"business.shipping", "carrier.steadfast.base_rate", "70", "number", "Steadfast Courier base rate (BDT)"},
		{"business.shipping", "carrier.redx.base_rate", "65", "number", "RedX base rate (BDT)"},
		{"business.shipping", "carrier.paperfly.base_rate", "55", "number", "Paperfly base rate (BDT)"},
		{"business.shipping", "carrier.sundarban.base_rate", "80", "number", "Sundarban Courier base rate (BDT)"},
		{"business.shipping", "carrier.sa_paribahan.base_rate", "70", "number", "SA Paribahan base rate (BDT)"},

		// Zone-based pricing (inside vs outside Dhaka)
		{"business.shipping", "zone.inside_dhaka", "60", "number", "Shipping rate inside Dhaka (BDT)"},
		{"business.shipping", "zone.outside_dhaka", "120", "number", "Shipping rate outside Dhaka (BDT)"},
		{"business.shipping", "zone.sub_district", "150", "number", "Shipping rate to sub-districts/upazila (BDT)"},

		// Weight surcharge
		{"business.shipping", "weight.surcharge_threshold_kg", "1", "number", "Weight threshold in kg before surcharge applies"},
		{"business.shipping", "weight.surcharge_per_kg", "20", "number", "Surcharge per kg above threshold (BDT)"},

		// COD (Cash on Delivery - very common in BD)
		{"business.shipping", "cod.enabled", "true", "boolean", "Enable Cash on Delivery"},
		{"business.shipping", "cod.charge_percentage", "1", "number", "COD charge percentage on order total"},
		{"business.shipping", "cod.min_charge", "10", "number", "Minimum COD charge (BDT)"},

		// ETAs
		{"business.shipping", "eta.inside_dhaka_days", "2", "number", "Estimated delivery days inside Dhaka"},
		{"business.shipping", "eta.outside_dhaka_days", "5", "number", "Estimated delivery days outside Dhaka"},
		{"business.shipping", "eta.sub_district_days", "7", "number", "Estimated delivery days to sub-districts"},

		// Free shipping
		{"business.shipping", "free_shipping_threshold", "0", "number", "Minimum order amount for free shipping in BDT (0 = disabled)"},
		{"business.shipping", "default_service_type", "standard", "string", "Default shipping service type"},
		{"business.shipping", "weight_unit", "kg", "string", "Weight unit (kg for metric)"},
	}
}

func businessRecommendationDefaults() []configDef {
	return []configDef{
		// Interaction weights
		{"business.recommendation", "weight.view", "1.0", "number", "Score weight for product view interaction"},
		{"business.recommendation", "weight.wishlist", "2.0", "number", "Score weight for wishlist add interaction"},
		{"business.recommendation", "weight.cart", "3.0", "number", "Score weight for add-to-cart interaction"},
		{"business.recommendation", "weight.purchase", "5.0", "number", "Score weight for purchase interaction"},

		// Limits
		{"business.recommendation", "default_limit", "10", "number", "Default number of recommendations returned"},
		{"business.recommendation", "max_limit", "50", "number", "Maximum number of recommendations allowed per request"},
		{"business.recommendation", "history_depth", "20", "number", "Number of recent interactions used for content-based filtering"},

		// Training
		{"business.recommendation", "co_purchase_min_count", "2", "number", "Minimum co-purchase count to generate similarity"},
		{"business.recommendation", "score_normalization_divisor", "10.0", "number", "Divisor for normalizing co-purchase count to 0-1 score"},
	}
}

func businessPromotionDefaults() []configDef {
	return []configDef{
		{"business.promotion", "max_discount_percentage", "100", "number", "Maximum allowed discount percentage"},
		{"business.promotion", "default_max_uses_per_user", "1", "number", "Default coupon usage limit per user"},
		{"business.promotion", "coupon_code_min_length", "4", "number", "Minimum length for coupon codes"},
		{"business.promotion", "coupon_code_max_length", "20", "number", "Maximum length for coupon codes"},
		{"business.promotion", "allow_stacking", "false", "boolean", "Whether multiple coupons can be applied to one order"},
	}
}

func tenantPlanDefaults() []configDef {
	return []configDef{
		// Trial
		{"tenant.plans", "trial_period_days", "14", "number", "Number of days for free trial period"},

		// Free tier limits
		{"tenant.plans", "free.max_users", "5", "number", "Maximum users allowed on Free plan"},
		{"tenant.plans", "free.max_products", "100", "number", "Maximum products allowed on Free plan"},
		{"tenant.plans", "free.max_orders", "1000", "number", "Maximum orders per month on Free plan"},
		{"tenant.plans", "free.db_strategy", "pool", "string", "Database strategy for Free plan (pool=shared DB)"},

		// Starter tier limits
		{"tenant.plans", "starter.max_users", "25", "number", "Maximum users allowed on Starter plan"},
		{"tenant.plans", "starter.max_products", "1000", "number", "Maximum products allowed on Starter plan"},
		{"tenant.plans", "starter.max_orders", "10000", "number", "Maximum orders per month on Starter plan"},
		{"tenant.plans", "starter.db_strategy", "pool", "string", "Database strategy for Starter plan (pool=shared DB)"},

		// Professional tier limits
		{"tenant.plans", "professional.max_users", "100", "number", "Maximum users allowed on Professional plan"},
		{"tenant.plans", "professional.max_products", "10000", "number", "Maximum products allowed on Professional plan"},
		{"tenant.plans", "professional.max_orders", "100000", "number", "Maximum orders per month on Professional plan"},
		{"tenant.plans", "professional.db_strategy", "bridge", "string", "Database strategy for Professional plan (bridge=separate schema)"},

		// Enterprise tier limits
		{"tenant.plans", "enterprise.max_users", "-1", "number", "Maximum users on Enterprise plan (-1 = unlimited)"},
		{"tenant.plans", "enterprise.max_products", "-1", "number", "Maximum products on Enterprise plan (-1 = unlimited)"},
		{"tenant.plans", "enterprise.max_orders", "-1", "number", "Maximum orders on Enterprise plan (-1 = unlimited)"},
		{"tenant.plans", "enterprise.db_strategy", "silo", "string", "Database strategy for Enterprise plan (silo=dedicated DB)"},

		// Default branding
		{"tenant.plans", "default_primary_color", "#3b82f6", "string", "Default primary branding color for new tenants"},
		{"tenant.plans", "default_secondary_color", "#10b981", "string", "Default secondary branding color for new tenants"},
	}
}

func tenantFeatureFlagDefaults() []configDef {
	return []configDef{
		// Free tier features
		{"tenant.features.free", "guest_checkout", "true", "boolean", "Guest checkout enabled on Free plan"},
		{"tenant.features.free", "product_reviews", "true", "boolean", "Product reviews enabled on Free plan"},
		{"tenant.features.free", "wishlist", "false", "boolean", "Wishlist feature on Free plan"},
		{"tenant.features.free", "multi_currency", "false", "boolean", "Multi-currency support on Free plan"},
		{"tenant.features.free", "social_login", "false", "boolean", "Social login on Free plan"},
		{"tenant.features.free", "ai_recommendations", "false", "boolean", "AI recommendations on Free plan"},
		{"tenant.features.free", "loyalty_program", "false", "boolean", "Loyalty program on Free plan"},
		{"tenant.features.free", "subscriptions", "false", "boolean", "Subscription products on Free plan"},
		{"tenant.features.free", "gift_cards", "false", "boolean", "Gift cards on Free plan"},

		// Starter tier features
		{"tenant.features.starter", "guest_checkout", "true", "boolean", "Guest checkout on Starter plan"},
		{"tenant.features.starter", "product_reviews", "true", "boolean", "Product reviews on Starter plan"},
		{"tenant.features.starter", "wishlist", "true", "boolean", "Wishlist feature on Starter plan"},
		{"tenant.features.starter", "multi_currency", "false", "boolean", "Multi-currency on Starter plan"},
		{"tenant.features.starter", "social_login", "false", "boolean", "Social login on Starter plan"},
		{"tenant.features.starter", "ai_recommendations", "false", "boolean", "AI recommendations on Starter plan"},
		{"tenant.features.starter", "loyalty_program", "false", "boolean", "Loyalty program on Starter plan"},
		{"tenant.features.starter", "subscriptions", "false", "boolean", "Subscriptions on Starter plan"},
		{"tenant.features.starter", "gift_cards", "false", "boolean", "Gift cards on Starter plan"},

		// Professional tier features
		{"tenant.features.professional", "guest_checkout", "true", "boolean", "Guest checkout on Professional plan"},
		{"tenant.features.professional", "product_reviews", "true", "boolean", "Product reviews on Professional plan"},
		{"tenant.features.professional", "wishlist", "true", "boolean", "Wishlist on Professional plan"},
		{"tenant.features.professional", "multi_currency", "true", "boolean", "Multi-currency on Professional plan"},
		{"tenant.features.professional", "social_login", "true", "boolean", "Social login on Professional plan"},
		{"tenant.features.professional", "ai_recommendations", "true", "boolean", "AI recommendations on Professional plan"},
		{"tenant.features.professional", "loyalty_program", "true", "boolean", "Loyalty program on Professional plan"},
		{"tenant.features.professional", "subscriptions", "false", "boolean", "Subscriptions on Professional plan"},
		{"tenant.features.professional", "gift_cards", "false", "boolean", "Gift cards on Professional plan"},

		// Enterprise tier features
		{"tenant.features.enterprise", "guest_checkout", "true", "boolean", "Guest checkout on Enterprise plan"},
		{"tenant.features.enterprise", "product_reviews", "true", "boolean", "Product reviews on Enterprise plan"},
		{"tenant.features.enterprise", "wishlist", "true", "boolean", "Wishlist on Enterprise plan"},
		{"tenant.features.enterprise", "multi_currency", "true", "boolean", "Multi-currency on Enterprise plan"},
		{"tenant.features.enterprise", "social_login", "true", "boolean", "Social login on Enterprise plan"},
		{"tenant.features.enterprise", "ai_recommendations", "true", "boolean", "AI recommendations on Enterprise plan"},
		{"tenant.features.enterprise", "loyalty_program", "true", "boolean", "Loyalty program on Enterprise plan"},
		{"tenant.features.enterprise", "subscriptions", "true", "boolean", "Subscriptions on Enterprise plan"},
		{"tenant.features.enterprise", "gift_cards", "true", "boolean", "Gift cards on Enterprise plan"},
	}
}

func searchDefaults() []configDef {
	return []configDef{
		{"search", "default_page_size", "20", "number", "Default search results per page"},
		{"search", "max_page_size", "100", "number", "Maximum search results per page"},
		{"search", "autocomplete.default_limit", "10", "number", "Default autocomplete suggestions count"},
		{"search", "autocomplete.max_limit", "20", "number", "Maximum autocomplete suggestions"},
		{"search", "elasticsearch.shards", "1", "number", "Number of Elasticsearch index shards"},
		{"search", "elasticsearch.replicas", "0", "number", "Number of Elasticsearch index replicas"},
		{"search", "elasticsearch.ngram_min", "2", "number", "Minimum n-gram length for autocomplete analyzer"},
		{"search", "elasticsearch.ngram_max", "20", "number", "Maximum n-gram length for autocomplete analyzer"},
		{"search", "elasticsearch.index_name", "products", "string", "Elasticsearch index name for products"},
		{"search", "fuzziness", "AUTO", "string", "Elasticsearch fuzziness setting for search queries"},
	}
}

func cartDefaults() []configDef {
	return []configDef{
		{"cart", "ttl_days", "7", "number", "Days before an inactive cart expires in Redis"},
		{"cart", "max_items", "100", "number", "Maximum number of items allowed in a cart"},
		{"cart", "max_quantity_per_item", "99", "number", "Maximum quantity of a single item in cart"},
		{"cart", "merge_on_login", "true", "boolean", "Whether to merge guest cart with user cart on login"},
		{"cart", "redis_key_prefix", "cart:", "string", "Redis key prefix for cart storage"},
	}
}

func analyticsDefaults() []configDef {
	return []configDef{
		{"analytics", "default_period", "daily", "string", "Default reporting period (daily, weekly, monthly)"},
		{"analytics", "default_date_range_days", "30", "number", "Default date range in days when not specified"},
		{"analytics", "product_performance.default_limit", "10", "number", "Default number of products in performance reports"},
		{"analytics", "product_performance.max_limit", "100", "number", "Maximum products in performance reports"},
		{"analytics", "top_customers_limit", "10", "number", "Number of top customers shown in insights"},
		{"analytics", "customer_segment.high_value_threshold", "1000", "number", "Minimum spend for high-value customer segment"},
		{"analytics", "customer_segment.medium_value_threshold", "100", "number", "Minimum spend for medium-value customer segment"},
		{"analytics", "default_channel", "web", "string", "Default sales channel when not specified"},
	}
}

func notificationDefaults() []configDef {
	return []configDef{
		// Default notification preferences
		{"notification", "default.email_enabled", "true", "boolean", "Email notifications enabled by default"},
		{"notification", "default.sms_enabled", "true", "boolean", "SMS notifications enabled by default"},
		{"notification", "default.push_enabled", "true", "boolean", "Push notifications enabled by default"},

		// Provider configuration
		{"notification", "provider.email", "simulated-sendgrid", "string", "Email notification provider"},
		{"notification", "provider.sms", "simulated-sms-bd", "string", "SMS notification provider (BD local gateway)"},
		{"notification", "provider.push", "simulated-fcm", "string", "Push notification provider"},

		// Rate limits
		{"notification", "rate_limit.email_per_hour", "100", "number", "Maximum emails per user per hour"},
		{"notification", "rate_limit.sms_per_hour", "10", "number", "Maximum SMS per user per hour"},
		{"notification", "rate_limit.push_per_hour", "50", "number", "Maximum push notifications per user per hour"},
	}
}

func paymentDefaults() []configDef {
	return []configDef{
		// Payment gateways (Bangladesh)
		{"payment", "default_currency", "BDT", "string", "Default payment currency"},
		{"payment", "gateway.primary", "sslcommerz", "string", "Primary payment gateway (SSLCommerz)"},
		{"payment", "gateway.secondary", "aamarpay", "string", "Secondary payment gateway (AamarPay)"},

		// Mobile Financial Services (MFS)
		{"payment", "mfs.bkash.enabled", "true", "boolean", "bKash mobile payment enabled"},
		{"payment", "mfs.nagad.enabled", "true", "boolean", "Nagad mobile payment enabled"},
		{"payment", "mfs.rocket.enabled", "true", "boolean", "Rocket (DBBL) mobile payment enabled"},
		{"payment", "mfs.upay.enabled", "false", "boolean", "Upay mobile payment enabled"},

		// Cash on Delivery
		{"payment", "cod.enabled", "true", "boolean", "Cash on Delivery enabled"},
		{"payment", "cod.max_amount", "50000", "number", "Maximum order amount for COD (BDT)"},

		// Bank transfer
		{"payment", "bank_transfer.enabled", "true", "boolean", "Bank transfer payment enabled"},

		// Card payments
		{"payment", "card.enabled", "true", "boolean", "Card payments enabled via gateway"},
		{"payment", "card.supported_networks", `["visa","mastercard","amex"]`, "json", "Supported card networks"},

		// Transaction limits
		{"payment", "min_order_amount", "50", "number", "Minimum order amount (BDT)"},
		{"payment", "max_order_amount", "500000", "number", "Maximum order amount (BDT)"},
	}
}

func serverDefaults() []configDef {
	return []configDef{
		// Service ports
		{"services", "order_service.port", "8080", "number", "Order service port"},
		{"services", "tenant_service.port", "8081", "number", "Tenant service port"},
		{"services", "user_service.port", "8082", "number", "User service port"},
		{"services", "product_service.port", "8083", "number", "Product service port"},
		{"services", "inventory_service.port", "8084", "number", "Inventory service port"},
		{"services", "payment_service.port", "8085", "number", "Payment service port"},
		{"services", "shipping_service.port", "8086", "number", "Shipping service port"},
		{"services", "notification_service.port", "8087", "number", "Notification service port"},
		{"services", "review_service.port", "8088", "number", "Review service port"},
		{"services", "cart_service.port", "8089", "number", "Cart service port"},
		{"services", "search_service.port", "8090", "number", "Search service port"},
		{"services", "promotion_service.port", "8091", "number", "Promotion service port"},
		{"services", "vendor_service.port", "8092", "number", "Vendor service port"},
		{"services", "analytics_service.port", "8093", "number", "Analytics service port"},
		{"services", "recommendation_service.port", "8094", "number", "Recommendation service port"},
		{"services", "config_service.port", "8095", "number", "Configuration service port"},

		// Service URLs (for inter-service communication)
		{"services", "inventory_service.url", "http://localhost:8084", "string", "Inventory service base URL"},
		{"services", "payment_service.url", "http://localhost:8085", "string", "Payment service base URL"},
	}
}

func reviewDefaults() []configDef {
	return []configDef{
		{"review", "auto_approve", "true", "boolean", "Whether new reviews are auto-approved"},
		{"review", "min_rating", "1", "number", "Minimum allowed rating value"},
		{"review", "max_rating", "5", "number", "Maximum allowed rating value"},
		{"review", "min_title_length", "3", "number", "Minimum review title length"},
		{"review", "max_title_length", "200", "number", "Maximum review title length"},
		{"review", "max_comment_length", "5000", "number", "Maximum review comment length"},
		{"review", "allow_anonymous", "false", "boolean", "Whether anonymous reviews are allowed"},
	}
}

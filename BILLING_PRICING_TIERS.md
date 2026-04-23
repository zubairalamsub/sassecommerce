# Billing & Pricing Tiers

## Overview

Comprehensive pricing strategy for the multi-tenant e-commerce platform, designed to accommodate businesses of all sizes from startups to enterprises.

---

## Table of Contents

1. [Pricing Philosophy](#pricing-philosophy)
2. [Pricing Tiers](#pricing-tiers)
3. [Feature Comparison](#feature-comparison)
4. [Usage-Based Pricing](#usage-based-pricing)
5. [Add-Ons & Extensions](#add-ons--extensions)
6. [Enterprise Custom Pricing](#enterprise-custom-pricing)
7. [Billing Configuration](#billing-configuration)
8. [Trial & Migration](#trial--migration)

---

## Pricing Philosophy

### Core Principles

1. **Transparent**: No hidden fees, clear pricing
2. **Scalable**: Grow with your business
3. **Flexible**: Pay for what you use
4. **Fair**: Competitive rates with high value
5. **Predictable**: Fixed monthly/annual costs

### Value-Based Pricing

Pricing is based on:
- Business size and scale
- Feature requirements
- Support level needed
- Infrastructure resources used

---

## Pricing Tiers

### Tier 1: Free (Starter)

**Price**: $0/month

**Target**: Hobbyists, testing, very small stores

**Limits**:
```json
{
  "products": 100,
  "ordersPerMonth": 1000,
  "users": 5,
  "storageMb": 1000,
  "apiCallsPerMinute": 100,
  "bandwidth": "10GB/month"
}
```

**Features**:
- ✅ Core e-commerce features
- ✅ Subdomain (yourstore.example.com)
- ✅ SSL certificate
- ✅ Basic templates
- ✅ Email support (48-hour response)
- ✅ Product management
- ✅ Order management
- ✅ Basic analytics
- ✅ Stripe integration
- ✅ Standard shipping

**Limitations**:
- ❌ No custom domain
- ❌ Platform branding visible
- ❌ No advanced analytics
- ❌ No priority support
- ❌ No API access
- ❌ Limited email sends (100/month)

**Best For**:
- Testing the platform
- Personal projects
- Very low-volume stores
- MVPs and prototypes

---

### Tier 2: Starter

**Price**:
- **Monthly**: $29/month
- **Annual**: $290/year (Save $58 - 17% discount)

**Target**: Small businesses, side hustles

**Limits**:
```json
{
  "products": 1000,
  "ordersPerMonth": 10000,
  "users": 20,
  "storageMb": 5000,
  "apiCallsPerMinute": 500,
  "bandwidth": "50GB/month",
  "emailsPerMonth": 5000
}
```

**Features**:
- ✅ Everything in Free, plus:
- ✅ Custom domain
- ✅ Remove platform branding
- ✅ Advanced analytics
- ✅ Email support (24-hour response)
- ✅ API access (basic rate limits)
- ✅ Abandoned cart recovery
- ✅ Discount codes
- ✅ Product reviews
- ✅ Wishlist
- ✅ Multiple payment gateways
- ✅ Gift cards
- ✅ Facebook/Google integrations
- ✅ Email marketing (5,000 emails/month)

**Limitations**:
- ❌ No priority support
- ❌ No advanced features
- ❌ Limited integrations
- ❌ No dedicated account manager

**Best For**:
- Small online stores
- Local businesses going online
- Boutique shops
- Artists and creators

---

### Tier 3: Professional

**Price**:
- **Monthly**: $99/month
- **Annual**: $990/year (Save $198 - 17% discount)

**Target**: Growing businesses, established stores

**Limits**:
```json
{
  "products": 10000,
  "ordersPerMonth": 100000,
  "users": 100,
  "storageMb": 50000,
  "apiCallsPerMinute": 1000,
  "bandwidth": "500GB/month",
  "emailsPerMonth": 50000
}
```

**Features**:
- ✅ Everything in Starter, plus:
- ✅ Priority email support (4-hour response)
- ✅ Phone support (business hours)
- ✅ Advanced API access
- ✅ Multi-currency support
- ✅ Multi-language support
- ✅ Advanced analytics & reports
- ✅ Customer segmentation
- ✅ Loyalty program
- ✅ Subscriptions & recurring billing
- ✅ Pre-orders
- ✅ Digital products
- ✅ Wholesale/B2B features
- ✅ Advanced shipping rules
- ✅ Real-time carrier rates
- ✅ Inventory management
- ✅ Low stock alerts
- ✅ Custom email templates
- ✅ Abandoned cart automation
- ✅ Product recommendations (AI)
- ✅ A/B testing
- ✅ SEO optimization tools
- ✅ Google Analytics integration
- ✅ Facebook Pixel
- ✅ Webhook support
- ✅ Custom CSS/JS

**Limitations**:
- ❌ No 24/7 support
- ❌ No dedicated infrastructure
- ❌ No SLA guarantees
- ❌ No dedicated account manager

**Best For**:
- Established online stores
- Multi-product businesses
- Regional brands
- Growing D2C brands

---

### Tier 4: Enterprise

**Price**: Custom (Starting at $499/month)

**Target**: Large businesses, high-volume stores, enterprise customers

**Limits**:
```json
{
  "products": "Unlimited",
  "ordersPerMonth": "Unlimited",
  "users": "Unlimited",
  "storageMb": "Unlimited",
  "apiCallsPerMinute": 10000,
  "bandwidth": "Unlimited",
  "emailsPerMonth": "Unlimited"
}
```

**Features**:
- ✅ Everything in Professional, plus:
- ✅ 24/7 priority support
- ✅ Dedicated account manager
- ✅ 99.99% uptime SLA
- ✅ Dedicated infrastructure
- ✅ Multi-region deployment
- ✅ Custom integrations
- ✅ White-label option
- ✅ Advanced security features
- ✅ Custom checkout flows
- ✅ Advanced fraud detection
- ✅ PCI DSS Level 1 compliance
- ✅ HIPAA compliance (if needed)
- ✅ Dedicated IP addresses
- ✅ Custom rate limits
- ✅ Priority feature requests
- ✅ Quarterly business reviews
- ✅ Training & onboarding
- ✅ Custom contract terms
- ✅ Volume discounts
- ✅ Data residency options
- ✅ Advanced reporting & BI
- ✅ Headless commerce API
- ✅ Custom domain architecture
- ✅ SSO/SAML integration
- ✅ Role-based access control (advanced)
- ✅ Audit logs
- ✅ API documentation
- ✅ Sandbox environment

**Best For**:
- Enterprise corporations
- High-volume retailers
- Marketplaces
- International brands
- Companies with compliance requirements

---

## Feature Comparison Matrix

| Feature | Free | Starter | Professional | Enterprise |
|---------|------|---------|--------------|------------|
| **Core Features** |
| Products | 100 | 1,000 | 10,000 | Unlimited |
| Orders/month | 1,000 | 10,000 | 100,000 | Unlimited |
| Team members | 5 | 20 | 100 | Unlimited |
| Storage | 1GB | 5GB | 50GB | Unlimited |
| Bandwidth | 10GB | 50GB | 500GB | Unlimited |
| **Branding** |
| Custom domain | ❌ | ✅ | ✅ | ✅ |
| Remove branding | ❌ | ✅ | ✅ | ✅ |
| Custom CSS/JS | ❌ | ❌ | ✅ | ✅ |
| White-label | ❌ | ❌ | ❌ | ✅ |
| **Sales Features** |
| Payment gateways | 1 | 3 | Unlimited | Unlimited |
| Discount codes | ❌ | ✅ | ✅ | ✅ |
| Gift cards | ❌ | ✅ | ✅ | ✅ |
| Subscriptions | ❌ | ❌ | ✅ | ✅ |
| Pre-orders | ❌ | ❌ | ✅ | ✅ |
| Digital products | ❌ | ❌ | ✅ | ✅ |
| Wholesale/B2B | ❌ | ❌ | ✅ | ✅ |
| **Marketing** |
| Email marketing | ❌ | 5K/mo | 50K/mo | Unlimited |
| Abandoned cart | ❌ | ✅ | ✅ | ✅ |
| Loyalty program | ❌ | ❌ | ✅ | ✅ |
| Product reviews | ❌ | ✅ | ✅ | ✅ |
| Wishlist | ❌ | ✅ | ✅ | ✅ |
| A/B testing | ❌ | ❌ | ✅ | ✅ |
| **International** |
| Multi-currency | ❌ | ❌ | ✅ | ✅ |
| Multi-language | ❌ | ❌ | ✅ | ✅ |
| Multi-region | ❌ | ❌ | ❌ | ✅ |
| **Analytics** |
| Basic analytics | ✅ | ✅ | ✅ | ✅ |
| Advanced analytics | ❌ | ✅ | ✅ | ✅ |
| Custom reports | ❌ | ❌ | ✅ | ✅ |
| Data export | ❌ | ✅ | ✅ | ✅ |
| BI integrations | ❌ | ❌ | ❌ | ✅ |
| **API & Developers** |
| API access | ❌ | Basic | Advanced | Full |
| Webhooks | ❌ | ❌ | ✅ | ✅ |
| Headless commerce | ❌ | ❌ | ❌ | ✅ |
| Sandbox environment | ❌ | ❌ | ❌ | ✅ |
| **Support** |
| Email support | 48h | 24h | 4h | 1h |
| Phone support | ❌ | ❌ | Business hours | 24/7 |
| Chat support | ❌ | ❌ | ✅ | ✅ |
| Account manager | ❌ | ❌ | ❌ | ✅ |
| **SLA & Security** |
| Uptime guarantee | 99% | 99.5% | 99.9% | 99.99% |
| SSL certificate | ✅ | ✅ | ✅ | ✅ |
| PCI compliance | Basic | ✅ | ✅ | Level 1 |
| Dedicated infrastructure | ❌ | ❌ | ❌ | ✅ |
| IP whitelisting | ❌ | ❌ | ❌ | ✅ |
| SSO/SAML | ❌ | ❌ | ❌ | ✅ |

---

## Usage-Based Pricing

### Additional Charges (All Tiers)

#### Over-Limit Charges

**When you exceed plan limits**:

```yaml
Products:
  - Free tier: $0.10 per additional product/month
  - Paid tiers: $0.05 per additional product/month

Orders:
  - Free tier: $0.05 per additional order
  - Paid tiers: $0.02 per additional order

Storage:
  - All tiers: $0.10 per GB/month over limit

Bandwidth:
  - All tiers: $0.05 per GB over limit

API Calls:
  - Paid tiers: $0.01 per 1,000 calls over limit

Email Sends:
  - Starter: $0.10 per 1,000 emails over limit
  - Professional: $0.05 per 1,000 emails over limit
```

**Soft Limits**:
- 10% overage allowed without charges
- Charges only apply if exceeded for 3+ consecutive days
- Automatic upgrade prompts when nearing limits

---

### Transaction Fees

**Free Tier Only**:
- 2% transaction fee on all orders
- Waived with upgrade to paid plan

**All Paid Tiers**:
- 0% transaction fees
- Standard payment processor fees apply (Stripe: 2.9% + $0.30)

---

## Add-Ons & Extensions

### Available Add-Ons (All Paid Tiers)

```yaml
Additional Storage:
  - Price: $10/month per 10GB
  - Available for: Starter, Professional

Additional Users:
  - Price: $5/month per user
  - Available for: Starter, Professional

Priority Support Upgrade:
  - Price: $49/month
  - Features:
    - 1-hour email response
    - Phone support (24/7)
    - Dedicated Slack channel
  - Available for: Professional

Advanced Fraud Detection:
  - Price: $29/month
  - Features:
    - Machine learning fraud scoring
    - IP blacklisting
    - Velocity checks
    - Chargeback prevention
  - Available for: Professional, Enterprise

Multi-Store Management:
  - Price: $99/month
  - Features:
    - Manage multiple stores from one dashboard
    - Centralized inventory
    - Cross-store analytics
  - Available for: Professional, Enterprise

POS System Integration:
  - Price: $79/month
  - Features:
    - Sync online and offline sales
    - Unified inventory
    - In-store pickup
  - Available for: Professional, Enterprise

Advanced Analytics Pack:
  - Price: $49/month
  - Features:
    - Custom dashboards
    - Predictive analytics
    - Cohort analysis
    - LTV calculation
  - Available for: Professional, Enterprise

SMS Marketing:
  - Price: $0.01 per SMS
  - Minimum: $20/month for 2,000 SMS
  - Available for: All paid tiers

International Expansion Pack:
  - Price: $149/month
  - Features:
    - Additional currencies (unlimited)
    - Language translations
    - Tax calculation for 200+ countries
    - Multi-regional shipping
  - Available for: Professional, Enterprise
```

---

## Enterprise Custom Pricing

### Enterprise Pricing Calculator

Base enterprise pricing starts at **$499/month** and scales based on:

```typescript
const calculateEnterprisePrice = (requirements: {
  estimatedOrders: number;
  estimatedRevenue: number;
  numberOfStores: number;
  regions: string[];
  support: 'standard' | 'premium' | 'dedicated';
  customFeatures: string[];
  compliance: string[];
}) => {
  let basePrice = 499;

  // Volume tier pricing
  if (requirements.estimatedOrders > 1000000) {
    basePrice = 2999;
  } else if (requirements.estimatedOrders > 500000) {
    basePrice = 1999;
  } else if (requirements.estimatedOrders > 100000) {
    basePrice = 999;
  }

  // Revenue tier
  if (requirements.estimatedRevenue > 50000000) {
    basePrice += 1000;
  } else if (requirements.estimatedRevenue > 10000000) {
    basePrice += 500;
  }

  // Multi-store
  if (requirements.numberOfStores > 1) {
    basePrice += (requirements.numberOfStores - 1) * 99;
  }

  // Multi-region
  if (requirements.regions.length > 1) {
    basePrice += (requirements.regions.length - 1) * 199;
  }

  // Support level
  const supportPricing = {
    standard: 0,
    premium: 299,
    dedicated: 999,
  };
  basePrice += supportPricing[requirements.support];

  // Custom features (estimated)
  basePrice += requirements.customFeatures.length * 199;

  // Compliance requirements
  basePrice += requirements.compliance.length * 149;

  return {
    monthlyPrice: basePrice,
    annualPrice: basePrice * 12 * 0.8, // 20% discount
  };
};
```

### Enterprise Pricing Examples

**Example 1: Mid-Size Enterprise**
```yaml
Profile:
  - Orders: 150,000/month
  - Revenue: $5M/year
  - Stores: 1
  - Regions: US only
  - Support: Premium
  - Custom features: 2
  - Compliance: PCI DSS

Pricing:
  - Base: $999
  - Support: $299
  - Custom features: $398
  - Compliance: $149
  - Total: $1,845/month
  - Annual: $17,712 (save $4,428)
```

**Example 2: Large Enterprise**
```yaml
Profile:
  - Orders: 2M/month
  - Revenue: $100M/year
  - Stores: 5
  - Regions: US, EU, APAC
  - Support: Dedicated
  - Custom features: 5
  - Compliance: PCI DSS L1, GDPR, HIPAA

Pricing:
  - Base: $2,999
  - Revenue tier: $1,000
  - Multi-store (4 additional): $396
  - Multi-region (2 additional): $398
  - Support: $999
  - Custom features: $995
  - Compliance: $447
  - Total: $7,234/month
  - Annual: $69,446 (save $17,362)
```

---

## Billing Configuration

### Payment Methods

```yaml
Accepted Payment Methods:
  - Credit Cards (Visa, Mastercard, Amex)
  - Debit Cards
  - ACH (US only, Enterprise)
  - Wire Transfer (Enterprise, annual only)
  - PayPal

Payment Processing:
  - Automatic renewal
  - Invoice sent 7 days before renewal
  - 3-day grace period after failed payment
  - Email notifications for payment issues
```

### Billing Cycles

```yaml
Monthly Billing:
  - Charged on signup date
  - Prorated on plan changes
  - Auto-renewal unless cancelled

Annual Billing:
  - Charged upfront
  - 17% discount on all tiers
  - Refund policy: Pro-rated refund within 30 days
  - Remainder non-refundable

Billing Date Changes:
  - Enterprise only
  - Requires 30-day notice
  - One-time fee: $99
```

### Invoicing

```yaml
Invoice Details:
  - PDF invoices emailed automatically
  - Downloadable from dashboard
  - Company details customizable
  - Tax ID / VAT number supported
  - Line-item breakdown

Tax Handling:
  - US Sales Tax (where applicable)
  - EU VAT (with VAT ID validation)
  - Canadian GST/HST
  - Australian GST
  - Reverse charge for B2B (EU)
```

---

## Trial & Migration

### Free Trial

```yaml
Trial Period:
  - Duration: 14 days
  - Available for: All paid tiers
  - No credit card required
  - Full feature access
  - Convert to paid: Any time

Trial to Paid:
  - Seamless conversion
  - No data migration needed
  - Billing starts immediately
  - No setup fees

Trial Extensions:
  - Available on request
  - Maximum one extension per customer
  - Up to 7 additional days
```

### Plan Upgrades

```yaml
Upgrade Process:
  - Immediate feature access
  - Prorated billing
  - No data migration needed
  - Automatic limit increases

Prorated Charges:
  - Unused credit from current plan
  - Applied to new plan
  - Charged difference immediately

Example (Month 15 of 30):
  - Current: Starter $29/month
  - Unused credit: $14.50
  - New: Professional $99/month
  - Charge: $84.50 immediately
  - Next full charge: In 15 days ($99)
```

### Plan Downgrades

```yaml
Downgrade Process:
  - Takes effect at end of billing period
  - Data preserved for 30 days
  - Warning if over new limits
  - Must reduce before downgrade

Over-Limit Handling:
  - Grace period: 30 days
  - Must delete excess data
  - Or upgrade back
  - Auto-upgrade option available

Downgrade Restrictions:
  - Cannot downgrade with active features
  - Must disable premium features first
  - Subscriptions must be migrated/cancelled
```

### Cancellation

```yaml
Cancellation Policy:
  - Cancel any time
  - No cancellation fees
  - Access until end of billing period
  - Data export available
  - Data retention: 90 days

Data Export:
  - All data downloadable
  - Formats: CSV, JSON, XML
  - Includes:
    - Products
    - Orders
    - Customers
    - Analytics data
    - Images (URLs)

Reactivation:
  - Within 90 days: Full restoration
  - After 90 days: New account required
  - Same pricing if reactivated
```

---

## Volume Discounts

### Multi-Year Contracts (Enterprise)

```yaml
2-Year Contract:
  - Discount: 25% off monthly price
  - Payment: Annual upfront
  - Renewal: Automatic at contract price

3-Year Contract:
  - Discount: 30% off monthly price
  - Payment: Annual upfront
  - Renewal: Automatic at contract price

Benefits:
  - Price lock guarantee
  - No price increases
  - Priority feature requests
  - Dedicated success manager
```

### Multi-Store Discounts

```yaml
2-5 Stores:
  - Discount: 10% per store

6-10 Stores:
  - Discount: 15% per store

11+ Stores:
  - Discount: 20% per store
  - Custom enterprise pricing available
```

---

## Referral Program

```yaml
Referral Rewards:
  - Give: 20% off first 3 months
  - Get: $50 credit per successful referral

Requirements:
  - Referee must be paid customer
  - Minimum 3 months subscription
  - Maximum 10 referrals per year

Credits:
  - Applied to next invoice
  - Non-transferable
  - Expire after 12 months
```

---

## Non-Profit & Educational Discounts

```yaml
Eligibility:
  - Registered 501(c)(3) (US)
  - Registered charity (International)
  - Educational institutions
  - Open-source projects

Discount:
  - 30% off any paid plan
  - All features included
  - Annual billing required

Verification:
  - Proof of status required
  - Annual renewal review
  - Non-transferable
```

---

This comprehensive pricing structure provides flexibility for all business sizes while maintaining clear value at each tier!

# Tenant Admin Dashboard Design

## Overview

Comprehensive design specification for the multi-tenant admin dashboard, providing tenant owners and admins with complete control over their e-commerce store configuration.

---

## Table of Contents

1. [Dashboard Overview](#dashboard-overview)
2. [Navigation Structure](#navigation-structure)
3. [Page Specifications](#page-specifications)
4. [Components Library](#components-library)
5. [Responsive Design](#responsive-design)
6. [Configuration Interfaces](#configuration-interfaces)

---

## Dashboard Overview

### Key Principles

- **Configurable**: Every aspect should be customizable
- **Intuitive**: Easy to navigate and understand
- **Responsive**: Works on desktop, tablet, and mobile
- **Real-time**: Live updates and notifications
- **Accessible**: WCAG 2.1 AA compliant

### Technology Stack

**Frontend**:
- React 18+ with TypeScript
- Next.js 14 for SSR
- TailwindCSS for styling
- Shadcn/ui for components
- React Query for data fetching
- Zustand for state management
- Chart.js / Recharts for analytics

---

## Navigation Structure

```
┌─────────────────────────────────────────────────────────────────┐
│  TENANT ADMIN DASHBOARD                                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─ SIDEBAR ──────┐  ┌─ MAIN CONTENT ───────────────────────┐ │
│  │                 │  │                                       │ │
│  │ 📊 Dashboard    │  │  Page Content                        │ │
│  │                 │  │                                       │ │
│  │ 🛍️ Products     │  │                                       │ │
│  │   - All         │  │                                       │ │
│  │   - Add New     │  │                                       │ │
│  │   - Categories  │  │                                       │ │
│  │   - Inventory   │  │                                       │ │
│  │                 │  │                                       │ │
│  │ 📦 Orders       │  │                                       │ │
│  │   - All Orders  │  │                                       │ │
│  │   - Abandoned   │  │                                       │ │
│  │                 │  │                                       │ │
│  │ 👥 Customers    │  │                                       │ │
│  │                 │  │                                       │ │
│  │ 💰 Payments     │  │                                       │ │
│  │                 │  │                                       │ │
│  │ 📊 Analytics    │  │                                       │ │
│  │   - Overview    │  │                                       │ │
│  │   - Sales       │  │                                       │ │
│  │   - Products    │  │                                       │ │
│  │   - Customers   │  │                                       │ │
│  │                 │  │                                       │ │
│  │ ⚙️ Settings     │  │                                       │ │
│  │   - General     │  │                                       │ │
│  │   - Branding    │  │                                       │ │
│  │   - Payment     │  │                                       │ │
│  │   - Shipping    │  │                                       │ │
│  │   - Email       │  │                                       │ │
│  │   - Features    │  │                                       │ │
│  │   - Team        │  │                                       │ │
│  │   - Security    │  │                                       │ │
│  │                 │  │                                       │ │
│  │ 💳 Billing      │  │                                       │ │
│  │   - Plan        │  │                                       │ │
│  │   - Usage       │  │                                       │ │
│  │   - Invoices    │  │                                       │ │
│  │                 │  │                                       │ │
│  │ 🔌 Apps         │  │                                       │ │
│  │                 │  │                                       │ │
│  └─────────────────┘  └───────────────────────────────────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Page Specifications

### 1. Dashboard Overview

**Route**: `/admin/dashboard`

**Purpose**: High-level overview of store performance

**Layout**:

```
┌────────────────────────────────────────────────────────────────┐
│  Dashboard                                    🔔 ⚙️ 👤          │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [Date Range Selector: Last 30 days ▾]  [Export ▾]            │
│                                                                 │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌────────┐│
│  │ 💰 Revenue   │ │ 📦 Orders    │ │ 👥 Customers │ │ 🛍️ AOV │││
│  │              │ │              │ │              │ │        │││
│  │  $125,430    │ │     5,430    │ │     2,340    │ │  $231  │││
│  │  ↑ 27.7%     │ │  ↑ 12.5%     │ │  ↑ 18.3%     │ │ ↓ 2.1% │││
│  └──────────────┘ └──────────────┘ └──────────────┘ └────────┘│
│                                                                 │
│  ┌─ Revenue Overview ──────────────────────────────────────┐  │
│  │                                                          │  │
│  │  [Line Chart: Last 30 days revenue]                     │  │
│  │                                                          │  │
│  │  $15K ┤     ╭─╮                                         │  │
│  │       │    ╱   ╰╮    ╭╮                                 │  │
│  │  $10K ┤   ╱     ╰─╮ ╱ ╰╮                                │  │
│  │       │  ╱        ╰╯   ╰─╮                              │  │
│  │   $5K ┤─╯               ╰──                             │  │
│  │       └────────────────────────────────────────         │  │
│  │        Jan 1      Jan 15      Jan 30                    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Recent Orders ──────────┐  ┌─ Top Products ────────────┐  │
│  │                           │  │                            │  │
│  │  #1234  $299  Pending    │  │  Wireless Headphones $249  │  │
│  │  #1235  $450  Confirmed  │  │  125 sold  $31,125         │  │
│  │  #1236  $180  Shipped    │  │                            │  │
│  │  #1237  $320  Pending    │  │  Bluetooth Speaker   $89   │  │
│  │                           │  │  98 sold   $8,722          │  │
│  │  [View All Orders →]     │  │                            │  │
│  └───────────────────────────┘  │  [View All Products →]    │  │
│                                  └────────────────────────────┘  │
│                                                                 │
│  ┌─ Traffic Sources ─────────────────────────────────────────┐ │
│  │                                                            │ │
│  │  Direct          45%  ████████████████                    │ │
│  │  Organic Search  30%  ██████████                          │ │
│  │  Social Media    15%  █████                               │ │
│  │  Email           10%  ███                                 │ │
│  └────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
```

**Configurable Elements**:
- Date range presets (Today, Last 7 days, Last 30 days, Custom)
- Widget visibility (show/hide specific metrics)
- Widget order (drag and drop)
- Chart types (line, bar, area)
- Currency display
- Timezone

---

### 2. General Settings

**Route**: `/admin/settings/general`

**Purpose**: Basic store configuration

**Layout**:

```
┌────────────────────────────────────────────────────────────────┐
│  Settings > General                                            │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Store Information ─────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Store Name *                                            │  │
│  │  ┌────────────────────────────────────────────────────┐ │  │
│  │  │ ACME Store                                         │ │  │
│  │  └────────────────────────────────────────────────────┘ │  │
│  │                                                          │  │
│  │  Store URL                                               │  │
│  │  ┌────────────────────────────────────────────────────┐ │  │
│  │  │ acme-store.example.com                             │ │  │
│  │  └────────────────────────────────────────────────────┘ │  │
│  │  🔗 Connect Custom Domain                               │  │
│  │                                                          │  │
│  │  Contact Email *                                         │  │
│  │  ┌────────────────────────────────────────────────────┐ │  │
│  │  │ support@acme.com                                   │ │  │
│  │  └────────────────────────────────────────────────────┘ │  │
│  │                                                          │  │
│  │  Phone Number                                            │  │
│  │  ┌────────────────────────────────────────────────────┐ │  │
│  │  │ +1 (555) 123-4567                                  │ │  │
│  │  └────────────────────────────────────────────────────┘ │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Regional Settings ──────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Currency *                                              │  │
│  │  ┌────────────────────────────────────────────────────┐ │  │
│  │  │ USD - United States Dollar                       ▾ │ │  │
│  │  └────────────────────────────────────────────────────┘ │  │
│  │                                                          │  │
│  │  Timezone *                                              │  │
│  │  ┌────────────────────────────────────────────────────┐ │  │
│  │  │ America/New_York (EST)                           ▾ │ │  │
│  │  └────────────────────────────────────────────────────┘ │  │
│  │                                                          │  │
│  │  Language                                                │  │
│  │  ┌────────────────────────────────────────────────────┐ │  │
│  │  │ English                                          ▾ │ │  │
│  │  └────────────────────────────────────────────────────┘ │  │
│  │                                                          │  │
│  │  Date Format                                             │  │
│  │  ☑ MM/DD/YYYY  ☐ DD/MM/YYYY  ☐ YYYY-MM-DD             │  │
│  │                                                          │  │
│  │  Time Format                                             │  │
│  │  ☑ 12-hour     ☐ 24-hour                               │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Multi-Currency (Professional & Enterprise) ──────────────┐ │
│  │                                                          │  │
│  │  ☑ Enable multi-currency                                │  │
│  │                                                          │  │
│  │  Supported Currencies:                                   │  │
│  │  ☑ USD  ☑ EUR  ☑ GBP  ☐ CAD  ☐ AUD  [+ Add More]      │  │
│  │                                                          │  │
│  │  Auto-detect customer currency: ☑                       │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [Cancel]                           [Save Changes]             │
└────────────────────────────────────────────────────────────────┘
```

**Configuration Options**:
```typescript
{
  storeInformation: {
    name: string;
    url: string;
    customDomain?: string;
    contactEmail: string;
    supportEmail?: string;
    phone?: string;
    address?: {
      line1: string;
      line2?: string;
      city: string;
      state?: string;
      postalCode: string;
      country: string;
    };
  },
  regional: {
    defaultCurrency: string; // ISO 4217
    timezone: string; // IANA timezone
    language: string; // ISO 639-1
    dateFormat: 'MM/DD/YYYY' | 'DD/MM/YYYY' | 'YYYY-MM-DD';
    timeFormat: '12h' | '24h';
    weekStartsOn: 'sunday' | 'monday';
  },
  multiCurrency: {
    enabled: boolean;
    supportedCurrencies: string[];
    autoDetect: boolean;
    conversionProvider: 'manual' | 'openexchangerates' | 'currencyapi';
  }
}
```

---

### 3. Branding Settings

**Route**: `/admin/settings/branding`

**Layout**:

```
┌────────────────────────────────────────────────────────────────┐
│  Settings > Branding                                           │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Logo & Favicon ─────────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Logo                                                    │  │
│  │  ┌──────────────┐                                        │  │
│  │  │              │  [Upload Logo]  [Remove]              │  │
│  │  │  ACME LOGO   │                                        │  │
│  │  │              │  Recommended: 200x80px, PNG or SVG    │  │
│  │  └──────────────┘                                        │  │
│  │                                                          │  │
│  │  Favicon                                                 │  │
│  │  ┌────┐                                                  │  │
│  │  │ A  │           [Upload Favicon]  [Remove]            │  │
│  │  └────┘                                                  │  │
│  │                  Recommended: 32x32px or 64x64px        │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Colors ─────────────────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Primary Color                                           │  │
│  │  ┌────┐  ┌──────────────┐                               │  │
│  │  │████│  │ #FF6B00      │                               │  │
│  │  └────┘  └──────────────┘                               │  │
│  │                                                          │  │
│  │  Secondary Color                                         │  │
│  │  ┌────┐  ┌──────────────┐                               │  │
│  │  │████│  │ #1A1A1A      │                               │  │
│  │  └────┘  └──────────────┘                               │  │
│  │                                                          │  │
│  │  Accent Color                                            │  │
│  │  ┌────┐  ┌──────────────┐                               │  │
│  │  │████│  │ #00B4D8      │                               │  │
│  │  └────┘  └──────────────┘                               │  │
│  │                                                          │  │
│  │  [Reset to Default Colors]                               │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Typography ─────────────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Font Family                                             │  │
│  │  ┌────────────────────────────────────────────────────┐ │  │
│  │  │ Inter                                            ▾ │ │  │
│  │  └────────────────────────────────────────────────────┘ │  │
│  │                                                          │  │
│  │  Available Fonts:                                        │  │
│  │  • Inter (Default)  • Roboto  • Open Sans               │  │
│  │  • Poppins  • Montserrat  • Custom Font                 │  │
│  │                                                          │  │
│  │  [Upload Custom Font]                                    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Preview ────────────────────────────────────────────────┐  │
│  │                                                          │  │
│  │  ┌─ Storefront Preview ─────────────────────────────┐   │  │
│  │  │                                                   │   │  │
│  │  │  [ACME LOGO]            🔍 Search    🛒 Cart     │   │  │
│  │  │                                                   │   │  │
│  │  │  ┌──────────┐                                     │   │  │
│  │  │  │ Product  │  Product Name         $99.99       │   │  │
│  │  │  │  Image   │  [Add to Cart]                     │   │  │
│  │  │  └──────────┘                                     │   │  │
│  │  └───────────────────────────────────────────────────┘   │  │
│  │                                                          │  │
│  │  ☐ Show preview on mobile                               │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [Cancel]                           [Save Changes]             │
└────────────────────────────────────────────────────────────────┘
```

**Configuration Options**:
```typescript
{
  logo: {
    url?: string;
    width?: number;
    height?: number;
  },
  favicon: {
    url?: string;
  },
  colors: {
    primary: string;      // Hex color
    secondary: string;    // Hex color
    accent: string;       // Hex color
    success: string;      // Hex color
    warning: string;      // Hex color
    error: string;        // Hex color
  },
  typography: {
    fontFamily: string;
    headingFont?: string;
    bodyFont?: string;
    customFontUrl?: string;
  },
  customCss?: string;
  customJs?: string;
}
```

---

### 4. Payment Settings

**Route**: `/admin/settings/payment`

**Layout**:

```
┌────────────────────────────────────────────────────────────────┐
│  Settings > Payment                                            │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Payment Providers ──────────────────────────────────────┐  │
│  │                                                          │  │
│  │  ┌─ Stripe ──────────────────────────────────────────┐  │  │
│  │  │  [Stripe Logo]                        ☑ Enabled  │  │  │
│  │  │                                                    │  │  │
│  │  │  Status: ✓ Connected                              │  │  │
│  │  │                                                    │  │  │
│  │  │  Publishable Key                                   │  │  │
│  │  │  pk_live_51K...                                    │  │  │
│  │  │                                                    │  │  │
│  │  │  [Configure]  [Disconnect]                         │  │  │
│  │  └────────────────────────────────────────────────────┘  │  │
│  │                                                          │  │
│  │  ┌─ PayPal ──────────────────────────────────────────┐  │  │
│  │  │  [PayPal Logo]                       ☑ Enabled   │  │  │
│  │  │                                                    │  │  │
│  │  │  Status: ✓ Connected                              │  │  │
│  │  │                                                    │  │  │
│  │  │  Mode: Production                                  │  │  │
│  │  │                                                    │  │  │
│  │  │  [Configure]  [Disconnect]                         │  │  │
│  │  └────────────────────────────────────────────────────┘  │  │
│  │                                                          │  │
│  │  ┌─ Apple Pay ───────────────────────────────────────┐  │  │
│  │  │  [Apple Pay Logo]                    ☐ Disabled  │  │  │
│  │  │                                                    │  │  │
│  │  │  Status: Not Connected                             │  │  │
│  │  │                                                    │  │  │
│  │  │  [Connect]                                         │  │  │
│  │  └────────────────────────────────────────────────────┘  │  │
│  │                                                          │  │
│  │  [+ Add Payment Provider]                                │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Payment Options ────────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Accepted Card Types                                     │  │
│  │  ☑ Visa  ☑ Mastercard  ☑ American Express              │  │
│  │  ☑ Discover  ☐ Diners Club  ☐ JCB                      │  │
│  │                                                          │  │
│  │  Payment Capture                                         │  │
│  │  ☑ Automatic   ☐ Manual                                 │  │
│  │                                                          │  │
│  │  ☑ Enable 3D Secure authentication                      │  │
│  │  ☑ Save cards for future use                            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Additional Options ─────────────────────────────────────┐  │
│  │                                                          │  │
│  │  ☑ Enable gift cards                                    │  │
│  │  ☑ Enable store credit                                  │  │
│  │  ☐ Enable cryptocurrency (Enterprise only)              │  │
│  │                                                          │  │
│  │  Test Mode                                               │  │
│  │  ☐ Enable test mode for all payment providers           │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [Cancel]                           [Save Changes]             │
└────────────────────────────────────────────────────────────────┘
```

**Configuration Options**:
```typescript
{
  providers: {
    stripe: {
      enabled: boolean;
      publishableKey: string;
      secretKey: string;    // Encrypted
      webhookSecret: string;
    },
    paypal: {
      enabled: boolean;
      mode: 'sandbox' | 'live';
      clientId: string;
      clientSecret: string; // Encrypted
    },
    applePay: {
      enabled: boolean;
      merchantId: string;
      certificate: string;
    },
  },
  options: {
    acceptedCards: ('visa' | 'mastercard' | 'amex' | 'discover')[];
    captureMode: 'automatic' | 'manual';
    enable3DSecure: boolean;
    saveCards: boolean;
  },
  additional: {
    giftCards: boolean;
    storeCredit: boolean;
    cryptocurrency: boolean;
  },
  testMode: boolean;
}
```

---

### 5. Shipping Settings

**Route**: `/admin/settings/shipping`

**Layout**:

```
┌────────────────────────────────────────────────────────────────┐
│  Settings > Shipping                                           │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Shipping Zones ─────────────────────────────────────────┐  │
│  │                                                          │  │
│  │  ┌─ United States ──────────────────────────────────┐   │  │
│  │  │  Countries: United States                        │   │  │
│  │  │                                                   │   │  │
│  │  │  Shipping Rates:                                  │   │  │
│  │  │  • Standard Shipping  $5.99   (3-5 business days)│   │  │
│  │  │  • Express Shipping   $14.99  (1-2 business days)│   │  │
│  │  │  • Overnight          $29.99  (Next day)         │   │  │
│  │  │                                                   │   │  │
│  │  │  [Edit Zone]  [Delete]                            │   │  │
│  │  └───────────────────────────────────────────────────┘   │  │
│  │                                                          │  │
│  │  ┌─ International ───────────────────────────────────┐   │  │
│  │  │  Countries: All other countries                   │   │  │
│  │  │                                                   │   │  │
│  │  │  Shipping Rates:                                  │   │  │
│  │  │  • Standard International  $19.99 (7-14 days)    │   │  │
│  │  │  • Express International   $49.99 (3-5 days)     │   │  │
│  │  │                                                   │   │  │
│  │  │  [Edit Zone]  [Delete]                            │   │  │
│  │  └───────────────────────────────────────────────────┘   │  │
│  │                                                          │  │
│  │  [+ Add Shipping Zone]                                   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Shipping Options ───────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Free Shipping                                           │  │
│  │  ☑ Enable free shipping                                 │  │
│  │                                                          │  │
│  │  Minimum Order Amount for Free Shipping                  │  │
│  │  ┌─────────────┐                                         │  │
│  │  │ $50.00      │                                         │  │
│  │  └─────────────┘                                         │  │
│  │                                                          │  │
│  │  Handling Time                                           │  │
│  │  ┌─────────────┐ business days                           │  │
│  │  │ 1-2         │                                         │  │
│  │  └─────────────┘                                         │  │
│  │                                                          │  │
│  │  ☑ Calculate shipping in real-time (requires carrier API)│  │
│  │  ☑ Allow local pickup                                   │  │
│  │  ☐ Restrict shipping to specific countries              │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Carrier Integration ────────────────────────────────────┐  │
│  │                                                          │  │
│  │  ┌─ FedEx ────────────────────────────────────────┐     │  │
│  │  │  Status: ✓ Connected                           │     │  │
│  │  │  [Configure]  [Disconnect]                      │     │  │
│  │  └─────────────────────────────────────────────────┘     │  │
│  │                                                          │  │
│  │  ┌─ UPS ──────────────────────────────────────────┐     │  │
│  │  │  Status: Not Connected                          │     │  │
│  │  │  [Connect]                                      │     │  │
│  │  └─────────────────────────────────────────────────┘     │  │
│  │                                                          │  │
│  │  [+ Add Carrier]                                         │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [Cancel]                           [Save Changes]             │
└────────────────────────────────────────────────────────────────┘
```

**Configuration Options**:
```typescript
{
  zones: Array<{
    id: string;
    name: string;
    countries: string[];  // ISO country codes
    rates: Array<{
      name: string;
      price: number;
      estimatedDays: string;
      conditions?: {
        minWeight?: number;
        maxWeight?: number;
        minPrice?: number;
        maxPrice?: number;
      };
    }>;
  }>,
  options: {
    freeShipping: {
      enabled: boolean;
      minimumAmount?: number;
    },
    handlingTime: {
      min: number;
      max: number;
      unit: 'days';
    },
    realTimeRates: boolean;
    localPickup: boolean;
    restrictions: {
      enabled: boolean;
      allowedCountries?: string[];
      blockedCountries?: string[];
    };
  },
  carriers: {
    fedex?: {
      enabled: boolean;
      accountNumber: string;
      meterNumber: string;
      apiKey: string;
    },
    ups?: {
      enabled: boolean;
      accountNumber: string;
      apiKey: string;
    },
    usps?: {
      enabled: boolean;
      userId: string;
    },
  }
}
```

---

### 6. Features Settings

**Route**: `/admin/settings/features`

**Layout**:

```
┌────────────────────────────────────────────────────────────────┐
│  Settings > Features                                           │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Shopping Features ──────────────────────────────────────┐  │
│  │                                                          │  │
│  │  ☑ Wishlist                                              │  │
│  │    Allow customers to save products for later            │  │
│  │                                                          │  │
│  │  ☑ Product Reviews                                       │  │
│  │    Enable customer product reviews and ratings           │  │
│  │    • Require purchase to review: ☑                       │  │
│  │    • Auto-publish reviews: ☐ (require moderation)       │  │
│  │                                                          │  │
│  │  ☑ Advanced Search                                       │  │
│  │    Faceted search with filters                           │  │
│  │                                                          │  │
│  │  ☑ AI-Powered Recommendations                            │  │
│  │    Smart product recommendations                         │  │
│  │                                                          │  │
│  │  ☑ Guest Checkout                                        │  │
│  │    Allow customers to checkout without an account        │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Premium Features (Professional & Enterprise) ──────────────┐│
│  │                                                          │  │
│  │  ☑ Multi-Currency                                        │  │
│  │    Support multiple currencies with auto-conversion      │  │
│  │                                                          │  │
│  │  ☐ Subscriptions                                         │  │
│  │    Sell subscription-based products                      │  │
│  │                                                          │  │
│  │  ☑ Gift Cards                                            │  │
│  │    Sell and redeem gift cards                            │  │
│  │                                                          │  │
│  │  ☐ Loyalty Program                                       │  │
│  │    Reward repeat customers with points                   │  │
│  │                                                          │  │
│  │  ☐ Pre-Orders                                            │  │
│  │    Accept orders for upcoming products                   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Social Features ────────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Social Login                                            │  │
│  │  ☑ Google    ☐ Facebook    ☐ Apple                      │  │
│  │                                                          │  │
│  │  ☑ Social Sharing                                        │  │
│  │    Share products on social media                        │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Marketing Features ─────────────────────────────────────┐  │
│  │                                                          │  │
│  │  ☑ Email Marketing                                       │  │
│  │    Send promotional emails to customers                  │  │
│  │                                                          │  │
│  │  ☑ Abandoned Cart Recovery                              │  │
│  │    Automatically email customers about abandoned carts   │  │
│  │    • Wait time before sending: [1] hours                 │  │
│  │    • Number of reminder emails: [3]                      │  │
│  │                                                          │  │
│  │  ☑ Discount Codes                                        │  │
│  │    Create coupon codes and promotions                    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [Cancel]                           [Save Changes]             │
└────────────────────────────────────────────────────────────────┘
```

---

### 7. Billing & Usage

**Route**: `/admin/billing`

**Layout**:

```
┌────────────────────────────────────────────────────────────────┐
│  Billing & Usage                                               │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Current Plan ───────────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Professional Plan         $99/month                     │  │
│  │                                                          │  │
│  │  ✓ 10,000 products         ✓ Custom domain              │  │
│  │  ✓ 100,000 orders/month    ✓ Priority support           │  │
│  │  ✓ 100 team members        ✓ Advanced analytics         │  │
│  │  ✓ 50GB storage                                          │  │
│  │                                                          │  │
│  │  Next billing date: February 1, 2024                     │  │
│  │                                                          │  │
│  │  [Change Plan]  [Cancel Subscription]                    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Usage This Month ───────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Products                                                │  │
│  │  1,250 / 10,000 (12.5%)                                  │  │
│  │  ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░                │  │
│  │                                                          │  │
│  │  Orders                                                  │  │
│  │  5,430 / 100,000 (5.4%)                                  │  │
│  │  ██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░                │  │
│  │                                                          │  │
│  │  Users                                                   │  │
│  │  15 / 100 (15%)                                          │  │
│  │  █████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░                │  │
│  │                                                          │  │
│  │  Storage                                                 │  │
│  │  8.9 GB / 50 GB (17.8%)                                  │  │
│  │  ██████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░                 │  │
│  │                                                          │  │
│  │  API Calls                                               │  │
│  │  125,430 today (Rate limit: 1,000/min)                   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Payment Method ──────────────────────────────────────────┐ │
│  │                                                          │  │
│  │  💳 Visa ending in 4242                                  │  │
│  │     Expires 12/2025                                      │  │
│  │                                                          │  │
│  │  [Update Payment Method]                                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Billing History ─────────────────────────────────────────┐ │
│  │                                                          │  │
│  │  Date          Amount    Status    Invoice              │  │
│  │  ────────────  ────────  ────────  ─────────            │  │
│  │  Jan 1, 2024   $99.00    Paid      📄 Download          │  │
│  │  Dec 1, 2023   $99.00    Paid      📄 Download          │  │
│  │  Nov 1, 2023   $99.00    Paid      📄 Download          │  │
│  │                                                          │  │
│  │  [View All Invoices →]                                   │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
```

---

## Components Library

### Reusable Components

```typescript
// Configuration Toggle
<ConfigToggle
  name="Enable Feature"
  description="Description of what this feature does"
  enabled={true}
  onChange={(enabled) => console.log(enabled)}
/>

// Configuration Input
<ConfigInput
  label="Store Name"
  value="ACME Store"
  required={true}
  helpText="This will be displayed on your storefront"
  onChange={(value) => console.log(value)}
/>

// Configuration Select
<ConfigSelect
  label="Currency"
  value="USD"
  options={[
    { value: 'USD', label: 'USD - United States Dollar' },
    { value: 'EUR', label: 'EUR - Euro' },
  ]}
  onChange={(value) => console.log(value)}
/>

// Color Picker
<ColorPicker
  label="Primary Color"
  value="#FF6B00"
  onChange={(color) => console.log(color)}
/>

// File Upload
<FileUpload
  label="Logo"
  accept="image/png,image/jpg,image/svg+xml"
  maxSize={2 * 1024 * 1024} // 2MB
  onUpload={(file) => console.log(file)}
/>

// Configuration Card
<ConfigCard
  title="Payment Providers"
  description="Configure payment methods for your store"
>
  {children}
</ConfigCard>
```

---

This provides a complete, highly configurable admin dashboard design with all necessary configuration interfaces!

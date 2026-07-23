'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import {
  ArrowRight,
  ShoppingBag,
  CreditCard,
  Banknote,
  CheckCircle,
  Loader2,
} from 'lucide-react';
import { useCartStore } from '@/stores/cart';
import { useAuthStore } from '@/stores/auth';
import { useProductStore } from '@/stores/products';
import { useDeliveryProfileStore } from '@/stores/delivery-profiles';
import { formatCurrency } from '@/lib/utils';
import { orderApi, promotionApi, paymentApi, type CouponValidateResponse, type ShippingRate } from '@/lib/api';
import { Tag, X, Truck } from 'lucide-react';
import { DEFAULT_TENANT_ID as TENANT_ID } from '@/lib/tenant';

const DHAKA_ZONE = ['Dhaka', 'Gazipur', 'Narayanganj', 'Tongi', 'Savar'];

const BD_CITIES = [
  'Dhaka',
  'Gazipur',
  'Narayanganj',
  'Tongi',
  'Savar',
  'Chittagong',
  'Sylhet',
  'Rajshahi',
  'Khulna',
  'Barisal',
  'Rangpur',
  'Mymensingh',
  'Comilla',
  'Bogra',
  'Jessore',
  'Cox\'s Bazar',
  'Dinajpur',
  'Tangail',
  'Brahmanbaria',
  'Narsingdi',
];

const PAYMENT_METHODS = [
  { id: 'bkash', label: 'bKash', description: 'Pay with bKash mobile wallet' },
  { id: 'nagad', label: 'Nagad', description: 'Pay with Nagad mobile wallet' },
  { id: 'rocket', label: 'Rocket', description: 'Pay with Rocket mobile banking' },
  { id: 'cod', label: 'Cash on Delivery', description: 'Pay when you receive the product' },
];

function estimatedDaysFromString(est: string, fallback: number): number {
  const match = est.match(/(\d+)/);
  return match ? parseInt(match[1], 10) : fallback;
}

export default function CheckoutPage() {
  const router = useRouter();
  const items = useCartStore((s) => s.items);
  const clearCart = useCartStore((s) => s.clearCart);
  const user = useAuthStore((s) => s.user);
  const token = useAuthStore((s) => s.token);
  const allProducts = useProductStore((s) => s.products);
  const dpProfiles = useDeliveryProfileStore((s) => s.profiles);
  const dpGetDefault = useDeliveryProfileStore((s) => s.getDefaultProfile);

  // Wait for Zustand to hydrate from localStorage
  const [hydrated, setHydrated] = useState(false);
  useEffect(() => { setHydrated(true); }, []);

  // Resolve the highest-rate delivery profile among all cart items
  function getCartDeliveryProfile() {
    const defaultProfile = dpGetDefault();
    let highest = defaultProfile;
    for (const item of items) {
      const product = allProducts.find((p) => p.id === item.productId);
      const profileId = product?.delivery_profile_id;
      const profile = profileId
        ? dpProfiles.find((p) => p.id === profileId) || defaultProfile
        : defaultProfile;
      // Use the profile with the highest outside_dhaka_rate as the "most expensive"
      if (profile.outside_dhaka_rate > highest.outside_dhaka_rate) {
        highest = profile;
      }
    }
    return highest;
  }

  const [formData, setFormData] = useState({
    name: user ? `${user.first_name} ${user.last_name}` : '',
    phone: user?.phone ?? '+880',
    email: user?.email ?? '',
    street: '',
    city: '',
    postalCode: '',
  });
  const [paymentMethod, setPaymentMethod] = useState('bkash');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [orderPlaced, setOrderPlaced] = useState(false);
  const [orderError, setOrderError] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [couponCode, setCouponCode] = useState('');
  const [couponApplied, setCouponApplied] = useState<CouponValidateResponse | null>(null);
  const [couponError, setCouponError] = useState('');
  const [couponLoading, setCouponLoading] = useState(false);
  const [shippingRates, setShippingRates] = useState<ShippingRate[]>([]);
  const [selectedCarrier, setSelectedCarrier] = useState<ShippingRate | null>(null);

  const subtotal = items.reduce((sum, i) => sum + i.price * i.quantity, 0);
  const discount = couponApplied?.discount_amount ?? 0;
  const shippingCost = selectedCarrier?.rate ?? (shippingRates.length > 0 ? shippingRates[0].rate : 0);
  const grandTotal = Math.max(0, subtotal + shippingCost - discount);

  function handleChange(
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
  ) {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
    if (errors[name]) {
      setErrors((prev) => {
        const next = { ...prev };
        delete next[name];
        return next;
      });
    }
    if (name === 'city') {
      fetchShippingRates(value);
    } else if (name === 'postalCode' && formData.city) {
      fetchShippingRates(formData.city);
    }
  }

  function fetchShippingRates(city: string) {
    if (!city) return;
    setSelectedCarrier(null);
    const isDhaka = DHAKA_ZONE.includes(city);
    const profile = getCartDeliveryProfile();
    const stdRate = isDhaka ? profile.inside_dhaka_rate : profile.outside_dhaka_rate;
    const expRate = isDhaka ? profile.inside_dhaka_express_rate : profile.outside_dhaka_express_rate;
    const estDays = isDhaka
      ? estimatedDaysFromString(profile.estimated_delivery_dhaka, 2)
      : estimatedDaysFromString(profile.estimated_delivery_outside, 4);
    const rates: ShippingRate[] = [
      { carrier: 'standard', service_type: isDhaka ? 'Inside Dhaka' : 'Outside Dhaka', rate: stdRate, currency: 'BDT', estimated_days: estDays },
      { carrier: 'express', service_type: isDhaka ? 'Inside Dhaka Express' : 'Outside Dhaka Express', rate: expRate, currency: 'BDT', estimated_days: Math.max(1, estDays - 1) },
    ];
    setShippingRates(rates);
    setSelectedCarrier(rates[0]);
  }

  function validate(): boolean {
    const newErrors: Record<string, string> = {};
    if (!formData.name.trim()) newErrors.name = 'Name is required';
    if (!formData.phone.trim() || formData.phone.length < 14)
      newErrors.phone = 'Valid phone number is required (+880XXXXXXXXXX)';
    if (!formData.email.trim() || !formData.email.includes('@'))
      newErrors.email = 'Valid email is required';
    if (!formData.street.trim()) newErrors.street = 'Street address is required';
    if (!formData.city) newErrors.city = 'Please select a city';
    if (!formData.postalCode.trim()) newErrors.postalCode = 'Postal code is required';
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }

  async function handleApplyCoupon() {
    if (!couponCode.trim()) return;
    setCouponError('');
    setCouponLoading(true);
    try {
      const res = await promotionApi.validate(couponCode.trim(), {
        tenant_id: TENANT_ID,
        user_id: user?.id || 'guest',
        order_total: subtotal,
      });
      if (res.valid) {
        setCouponApplied(res);
      } else {
        setCouponError(res.message || 'Invalid coupon code');
        setCouponApplied(null);
      }
    } catch {
      setCouponError('Unable to validate coupon. Please try again.');
      setCouponApplied(null);
    } finally {
      setCouponLoading(false);
    }
  }

  async function handlePlaceOrder() {
    if (!validate()) return;

    setIsSubmitting(true);
    setOrderError('');
    try {
      const address = {
        street: formData.street,
        city: formData.city,
        state: formData.city,
        postal_code: formData.postalCode,
        country: 'Bangladesh',
      };

      const orderReq: Parameters<typeof orderApi.create>[0] = {
        tenant_id: TENANT_ID,
        shipping_address: address,
        billing_address: address,
      };

      if (user) {
        orderReq.customer_id = user.id;
      } else {
        orderReq.guest_email = formData.email;
        orderReq.guest_name = formData.name;
        orderReq.guest_phone = formData.phone;
      }

      const order = await orderApi.create(orderReq, TENANT_ID, token || undefined);
      const orderId = order.order_id || order.id || '';

      // Add items sequentially (event-sourced order uses optimistic concurrency,
      // so parallel adds cause version conflicts)
      for (const item of items) {
        await orderApi.addItem(orderId, {
          product_id: item.productId,
          variant_id: item.variantId || '',
          sku: item.sku,
          name: item.name,
          quantity: item.quantity,
          unit_price: item.price,
        }, TENANT_ID, token || undefined);
      }

      // Record payment (skip for COD — customer pays on delivery)
      if (paymentMethod !== 'cod' && user) {
        try {
          await paymentApi.process({
            tenant_id: TENANT_ID,
            customer_id: user.id,
            order_id: orderId,
            amount: grandTotal,
            currency: 'BDT',
            method: paymentMethod,
          }, TENANT_ID, token || undefined);
        } catch {
          // Payment recording failed — order still placed
        }
      }

      setIsSubmitting(false);
      setOrderPlaced(true);
      const auth = user && token ? { userId: user.id, tenantId: TENANT_ID, token } : undefined;
      clearCart(auth);
      setTimeout(() => {
        router.push(`/orders/${orderId}`);
      }, 2000);
    } catch (err) {
      setIsSubmitting(false);
      setOrderError(
        err instanceof Error ? err.message : 'Failed to place order. Please try again.',
      );
    }
  }

  if (!hydrated) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (items.length === 0 && !orderPlaced) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-20 text-center sm:px-6 lg:px-8">
        <div className="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-surface-hover">
          <ShoppingBag className="h-10 w-10 text-text-muted" />
        </div>
        <h1 className="mt-6 text-2xl font-bold text-text">Your cart is empty</h1>
        <p className="mt-2 text-text-secondary">Add some products before checkout.</p>
        <Link
          href="/products"
          className="mt-8 inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-3 font-medium text-white transition-colors hover:bg-primary-dark"
        >
          Browse Products
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>
    );
  }

  if (orderPlaced) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-20 text-center sm:px-6 lg:px-8">
        <div className="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
          <CheckCircle className="h-12 w-12 text-green-600 dark:text-green-400" />
        </div>
        <h1 className="mt-6 text-2xl font-bold text-text">Order Placed Successfully!</h1>
        <p className="mt-2 text-text-secondary">
          Thank you for your order. You will be redirected shortly.
        </p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold text-text">Checkout</h1>
      <p className="mt-1 text-text-secondary">Complete your order details below</p>

      <div className="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-3">
        {/* Left column - Forms */}
        <div className="lg:col-span-2 space-y-8">
          {/* Shipping Information */}
          <div className="rounded-xl border border-border bg-surface p-6">
            <h2 className="text-lg font-semibold text-text">Shipping Information</h2>

            <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
              {/* Full Name */}
              <div className="sm:col-span-2">
                <label className="mb-1 block text-sm font-medium text-text-secondary">
                  Full Name
                </label>
                <input
                  type="text"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="e.g. Rahim Uddin"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.name ? 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-900/20' : 'border-border bg-surface text-text'
                  }`}
                />
                {errors.name && (
                  <p className="mt-1 text-xs text-red-500">{errors.name}</p>
                )}
              </div>

              {/* Phone */}
              <div>
                <label className="mb-1 block text-sm font-medium text-text-secondary">
                  Phone Number
                </label>
                <input
                  type="tel"
                  name="phone"
                  value={formData.phone}
                  onChange={handleChange}
                  placeholder="+880 1XXXXXXXXX"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.phone ? 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-900/20' : 'border-border bg-surface text-text'
                  }`}
                />
                {errors.phone && (
                  <p className="mt-1 text-xs text-red-500">{errors.phone}</p>
                )}
              </div>

              {/* Email */}
              <div>
                <label className="mb-1 block text-sm font-medium text-text-secondary">
                  Email Address
                </label>
                <input
                  type="email"
                  name="email"
                  value={formData.email}
                  onChange={handleChange}
                  placeholder="you@example.com"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.email ? 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-900/20' : 'border-border bg-surface text-text'
                  }`}
                />
                {errors.email && (
                  <p className="mt-1 text-xs text-red-500">{errors.email}</p>
                )}
              </div>

              {/* Street Address */}
              <div className="sm:col-span-2">
                <label className="mb-1 block text-sm font-medium text-text-secondary">
                  Street Address
                </label>
                <input
                  type="text"
                  name="street"
                  value={formData.street}
                  onChange={handleChange}
                  placeholder="House #, Road #, Area"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.street ? 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-900/20' : 'border-border bg-surface text-text'
                  }`}
                />
                {errors.street && (
                  <p className="mt-1 text-xs text-red-500">{errors.street}</p>
                )}
              </div>

              {/* City */}
              <div>
                <label className="mb-1 block text-sm font-medium text-text-secondary">
                  City
                </label>
                <select
                  name="city"
                  value={formData.city}
                  onChange={handleChange}
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.city ? 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-900/20' : 'border-border bg-surface text-text'
                  }`}
                >
                  <option value="">Select city</option>
                  {BD_CITIES.map((city) => (
                    <option key={city} value={city}>
                      {city}
                    </option>
                  ))}
                </select>
                {errors.city && (
                  <p className="mt-1 text-xs text-red-500">{errors.city}</p>
                )}
              </div>

              {/* Postal Code */}
              <div>
                <label className="mb-1 block text-sm font-medium text-text-secondary">
                  Postal Code
                </label>
                <input
                  type="text"
                  name="postalCode"
                  value={formData.postalCode}
                  onChange={handleChange}
                  placeholder="e.g. 1205"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.postalCode ? 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-900/20' : 'border-border bg-surface text-text'
                  }`}
                />
                {errors.postalCode && (
                  <p className="mt-1 text-xs text-red-500">{errors.postalCode}</p>
                )}
              </div>
            </div>
          </div>

          {/* Payment Method */}
          <div className="rounded-xl border border-border bg-surface p-6">
            <h2 className="text-lg font-semibold text-text">Payment Method</h2>

            <div className="mt-4 space-y-3">
              {PAYMENT_METHODS.map((method) => (
                <label
                  key={method.id}
                  className={`flex cursor-pointer items-center gap-4 rounded-lg border p-4 transition-colors ${
                    paymentMethod === method.id
                      ? 'border-primary bg-primary-light'
                      : 'border-border hover:border-primary/40'
                  }`}
                >
                  <input
                    type="radio"
                    name="paymentMethod"
                    value={method.id}
                    checked={paymentMethod === method.id}
                    onChange={() => setPaymentMethod(method.id)}
                    className="h-4 w-4 text-primary focus:ring-primary"
                  />
                  <div className="flex items-center gap-3">
                    {method.id === 'cod' ? (
                      <Banknote className="h-5 w-5 text-text-secondary" />
                    ) : (
                      <CreditCard className="h-5 w-5 text-text-secondary" />
                    )}
                    <div>
                      <span className="text-sm font-medium text-text">
                        {method.label}
                      </span>
                      <p className="text-xs text-text-secondary">{method.description}</p>
                    </div>
                  </div>
                </label>
              ))}
            </div>
          </div>
        </div>

        {/* Right column - Order summary */}
        <div>
          <div className="sticky top-8 rounded-xl border border-border bg-surface p-6">
            <h2 className="text-lg font-semibold text-text">Order Summary</h2>

            {/* Items list */}
            <div className="mt-4 divide-y divide-border-light">
              {items.map((item) => (
                <div
                  key={`${item.productId}-${item.variantId ?? ''}`}
                  className="flex items-center gap-3 py-3"
                >
                  <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-primary/20 to-primary/5">
                    <span className="text-sm font-bold text-primary/40">
                      {item.name.charAt(0)}
                    </span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="truncate text-sm font-medium text-text">
                      {item.name}
                    </p>
                    <p className="text-xs text-text-secondary">Qty: {item.quantity}</p>
                  </div>
                  <span className="text-sm font-medium text-text">
                    {formatCurrency(item.price * item.quantity)}
                  </span>
                </div>
              ))}
            </div>

            {/* Delivery Option */}
            {shippingRates.length > 0 && (
              <div className="mt-4 border-t border-border pt-4">
                <h3 className="mb-2 flex items-center gap-1.5 text-sm font-medium text-text-secondary">
                  <Truck className="h-4 w-4" />
                  Delivery Option
                </h3>
                <div className="mb-2 rounded-md bg-blue-50 dark:bg-blue-900/30 px-3 py-1.5">
                  <span className="text-xs font-medium text-blue-700 dark:text-blue-300">
                    {DHAKA_ZONE.includes(formData.city) ? 'Inside Dhaka Zone' : 'Outside Dhaka'}
                  </span>
                </div>
                <div className="space-y-2">
                  {shippingRates.map((rate) => {
                    const isSelected = selectedCarrier?.carrier === rate.carrier;
                    const isExpress = rate.carrier === 'express';
                    return (
                      <label
                        key={rate.carrier}
                        className={`flex cursor-pointer items-center justify-between rounded-lg border px-3 py-3 transition-colors ${
                          isSelected
                            ? 'border-primary bg-primary/5'
                            : 'border-border hover:border-primary/40'
                        }`}
                      >
                        <div className="flex items-center gap-2">
                          <input
                            type="radio"
                            name="shippingCarrier"
                            checked={isSelected}
                            onChange={() => setSelectedCarrier(rate)}
                            className="h-3.5 w-3.5 text-primary focus:ring-primary"
                          />
                          <div>
                            <div className="flex items-center gap-1.5">
                              <span className="text-sm font-medium text-text">
                                {isExpress ? 'Express Delivery' : 'Standard Delivery'}
                              </span>
                              {isExpress && (
                                <span className="rounded-full bg-amber-100 dark:bg-amber-900/40 px-1.5 py-0.5 text-[10px] font-semibold text-amber-700 dark:text-amber-300">FAST</span>
                              )}
                            </div>
                            <p className="text-xs text-text-secondary">
                              {rate.estimated_days === 1 ? 'Next day delivery' : `${rate.estimated_days}-${rate.estimated_days + 1} business days`}
                            </p>
                          </div>
                        </div>
                        <span className="text-sm font-semibold text-text">
                          ৳{rate.rate}
                        </span>
                      </label>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Coupon Code */}
            <div className="mt-4 border-t border-border pt-4">
              {couponApplied ? (
                <div className="flex items-center justify-between rounded-lg bg-green-50 dark:bg-green-900/30 px-3 py-2">
                  <div className="flex items-center gap-2 text-sm text-green-700 dark:text-green-300">
                    <Tag className="h-4 w-4" />
                    <span className="font-medium">{couponApplied.code}</span>
                    <span className="text-green-600 dark:text-green-400">applied</span>
                  </div>
                  <button
                    onClick={() => { setCouponApplied(null); setCouponCode(''); setCouponError(''); }}
                    className="text-green-600 hover:text-green-800 dark:text-green-400 dark:hover:text-green-200"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ) : (
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-text-secondary">
                    Coupon Code
                  </label>
                  <div className="flex gap-2">
                    <div className="relative flex-1">
                      <Tag className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" />
                      <input
                        type="text"
                        value={couponCode}
                        onChange={(e) => { setCouponCode(e.target.value.toUpperCase()); setCouponError(''); }}
                        onKeyDown={(e) => e.key === 'Enter' && handleApplyCoupon()}
                        placeholder="Enter code"
                        className="w-full rounded-lg border border-border bg-surface py-2 pl-9 pr-3 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                    </div>
                    <button
                      onClick={handleApplyCoupon}
                      disabled={couponLoading || !couponCode.trim()}
                      className="rounded-lg bg-text px-4 py-2 text-sm font-medium text-surface transition-colors hover:bg-text/85 disabled:opacity-50"
                    >
                      {couponLoading ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        'Apply'
                      )}
                    </button>
                  </div>
                  {couponError && (
                    <p className="mt-1.5 text-xs text-red-500">{couponError}</p>
                  )}
                </div>
              )}
            </div>

            {/* Totals */}
            <div className="mt-4 space-y-3 border-t border-border pt-4">
              <div className="flex justify-between text-sm">
                <span className="text-text-secondary">Subtotal</span>
                <span className="font-medium text-text">
                  {formatCurrency(subtotal)}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-text-secondary">
                  Delivery
                  {selectedCarrier && (
                    <span className="ml-1 text-text-muted">
                      ({selectedCarrier.carrier === 'express' ? 'Express' : 'Standard'})
                    </span>
                  )}
                </span>
                <span className="font-medium text-text">
                  {shippingRates.length > 0 ? (
                    formatCurrency(shippingCost)
                  ) : (
                    <span className="text-text-muted text-xs">Select city</span>
                  )}
                </span>
              </div>
              {couponApplied && (
                <div className="flex justify-between text-sm">
                  <span className="text-green-600 dark:text-green-400">Discount ({couponApplied.code})</span>
                  <span className="font-medium text-green-600 dark:text-green-400">
                    -{formatCurrency(discount)}
                  </span>
                </div>
              )}
              <div className="border-t border-border pt-3">
                <div className="flex justify-between">
                  <span className="text-base font-semibold text-text">Total</span>
                  <span className="text-lg font-bold text-text">
                    {formatCurrency(grandTotal)}
                  </span>
                </div>
              </div>
            </div>

            {/* Order error */}
            {orderError && (
              <div className="mt-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 px-4 py-3 text-sm text-red-700 dark:text-red-300">
                {orderError}
              </div>
            )}

            {/* Place Order button */}
            <button
              onClick={handlePlaceOrder}
              disabled={isSubmitting}
              className="mt-6 flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-6 py-3 font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-60"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="h-5 w-5 animate-spin" />
                  Processing...
                </>
              ) : (
                <>
                  Place Order
                  <ArrowRight className="h-4 w-4" />
                </>
              )}
            </button>

            <p className="mt-3 text-center text-xs text-text-muted">
              By placing your order you agree to our Terms &amp; Conditions
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

'use client';

import { useState } from 'react';
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
import { formatCurrency } from '@/lib/utils';

const BD_CITIES = [
  'Dhaka',
  'Chittagong',
  'Sylhet',
  'Rajshahi',
  'Khulna',
  'Barisal',
  'Rangpur',
  'Mymensingh',
  'Comilla',
  'Gazipur',
  'Narayanganj',
];

const PAYMENT_METHODS = [
  { id: 'bkash', label: 'bKash', description: 'Pay with bKash mobile wallet' },
  { id: 'nagad', label: 'Nagad', description: 'Pay with Nagad mobile wallet' },
  { id: 'rocket', label: 'Rocket', description: 'Pay with Rocket mobile banking' },
  { id: 'cod', label: 'Cash on Delivery', description: 'Pay when you receive the product' },
];

const SHIPPING_COST = 100;

export default function CheckoutPage() {
  const router = useRouter();
  const items = useCartStore((s) => s.items);
  const total = useCartStore((s) => s.total);
  const clearCart = useCartStore((s) => s.clearCart);
  const user = useAuthStore((s) => s.user);

  const subtotal = total();
  const grandTotal = subtotal + SHIPPING_COST;

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
  const [errors, setErrors] = useState<Record<string, string>>({});

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

  async function handlePlaceOrder() {
    if (!validate()) return;

    setIsSubmitting(true);
    // Simulate order placement
    await new Promise((resolve) => setTimeout(resolve, 1500));
    setIsSubmitting(false);
    setOrderPlaced(true);

    // Generate a demo order ID and redirect after a short delay
    const orderId = `ORD-${Date.now().toString(36).toUpperCase()}`;
    clearCart();
    setTimeout(() => {
      router.push(`/orders/${orderId}`);
    }, 2000);
  }

  if (items.length === 0 && !orderPlaced) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-20 text-center sm:px-6 lg:px-8">
        <div className="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-gray-100">
          <ShoppingBag className="h-10 w-10 text-gray-400" />
        </div>
        <h1 className="mt-6 text-2xl font-bold text-gray-900">Your cart is empty</h1>
        <p className="mt-2 text-gray-500">Add some products before checkout.</p>
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
        <div className="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-green-100">
          <CheckCircle className="h-12 w-12 text-green-600" />
        </div>
        <h1 className="mt-6 text-2xl font-bold text-gray-900">Order Placed Successfully!</h1>
        <p className="mt-2 text-gray-500">
          Thank you for your order. You will be redirected shortly.
        </p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold text-gray-900">Checkout</h1>
      <p className="mt-1 text-gray-500">Complete your order details below</p>

      <div className="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-3">
        {/* Left column - Forms */}
        <div className="lg:col-span-2 space-y-8">
          {/* Shipping Information */}
          <div className="rounded-xl border border-gray-200 bg-white p-6">
            <h2 className="text-lg font-semibold text-gray-900">Shipping Information</h2>

            <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
              {/* Full Name */}
              <div className="sm:col-span-2">
                <label className="mb-1 block text-sm font-medium text-gray-700">
                  Full Name
                </label>
                <input
                  type="text"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="e.g. Rahim Uddin"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.name ? 'border-red-300 bg-red-50' : 'border-gray-200'
                  }`}
                />
                {errors.name && (
                  <p className="mt-1 text-xs text-red-500">{errors.name}</p>
                )}
              </div>

              {/* Phone */}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">
                  Phone Number
                </label>
                <input
                  type="tel"
                  name="phone"
                  value={formData.phone}
                  onChange={handleChange}
                  placeholder="+880 1XXXXXXXXX"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.phone ? 'border-red-300 bg-red-50' : 'border-gray-200'
                  }`}
                />
                {errors.phone && (
                  <p className="mt-1 text-xs text-red-500">{errors.phone}</p>
                )}
              </div>

              {/* Email */}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">
                  Email Address
                </label>
                <input
                  type="email"
                  name="email"
                  value={formData.email}
                  onChange={handleChange}
                  placeholder="you@example.com"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.email ? 'border-red-300 bg-red-50' : 'border-gray-200'
                  }`}
                />
                {errors.email && (
                  <p className="mt-1 text-xs text-red-500">{errors.email}</p>
                )}
              </div>

              {/* Street Address */}
              <div className="sm:col-span-2">
                <label className="mb-1 block text-sm font-medium text-gray-700">
                  Street Address
                </label>
                <input
                  type="text"
                  name="street"
                  value={formData.street}
                  onChange={handleChange}
                  placeholder="House #, Road #, Area"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.street ? 'border-red-300 bg-red-50' : 'border-gray-200'
                  }`}
                />
                {errors.street && (
                  <p className="mt-1 text-xs text-red-500">{errors.street}</p>
                )}
              </div>

              {/* City */}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">
                  City
                </label>
                <select
                  name="city"
                  value={formData.city}
                  onChange={handleChange}
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.city ? 'border-red-300 bg-red-50' : 'border-gray-200'
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
                <label className="mb-1 block text-sm font-medium text-gray-700">
                  Postal Code
                </label>
                <input
                  type="text"
                  name="postalCode"
                  value={formData.postalCode}
                  onChange={handleChange}
                  placeholder="e.g. 1205"
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm transition-colors focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary ${
                    errors.postalCode ? 'border-red-300 bg-red-50' : 'border-gray-200'
                  }`}
                />
                {errors.postalCode && (
                  <p className="mt-1 text-xs text-red-500">{errors.postalCode}</p>
                )}
              </div>
            </div>
          </div>

          {/* Payment Method */}
          <div className="rounded-xl border border-gray-200 bg-white p-6">
            <h2 className="text-lg font-semibold text-gray-900">Payment Method</h2>

            <div className="mt-4 space-y-3">
              {PAYMENT_METHODS.map((method) => (
                <label
                  key={method.id}
                  className={`flex cursor-pointer items-center gap-4 rounded-lg border p-4 transition-colors ${
                    paymentMethod === method.id
                      ? 'border-primary bg-primary-light'
                      : 'border-gray-200 hover:border-gray-300'
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
                      <Banknote className="h-5 w-5 text-gray-600" />
                    ) : (
                      <CreditCard className="h-5 w-5 text-gray-600" />
                    )}
                    <div>
                      <span className="text-sm font-medium text-gray-900">
                        {method.label}
                      </span>
                      <p className="text-xs text-gray-500">{method.description}</p>
                    </div>
                  </div>
                </label>
              ))}
            </div>
          </div>
        </div>

        {/* Right column - Order summary */}
        <div>
          <div className="sticky top-8 rounded-xl border border-gray-200 bg-white p-6">
            <h2 className="text-lg font-semibold text-gray-900">Order Summary</h2>

            {/* Items list */}
            <div className="mt-4 divide-y divide-gray-100">
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
                    <p className="truncate text-sm font-medium text-gray-900">
                      {item.name}
                    </p>
                    <p className="text-xs text-gray-500">Qty: {item.quantity}</p>
                  </div>
                  <span className="text-sm font-medium text-gray-900">
                    {formatCurrency(item.price * item.quantity)}
                  </span>
                </div>
              ))}
            </div>

            {/* Totals */}
            <div className="mt-4 space-y-3 border-t border-gray-200 pt-4">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Subtotal</span>
                <span className="font-medium text-gray-900">
                  {formatCurrency(subtotal)}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Shipping</span>
                <span className="font-medium text-gray-900">
                  {formatCurrency(SHIPPING_COST)}
                </span>
              </div>
              <div className="border-t border-gray-200 pt-3">
                <div className="flex justify-between">
                  <span className="text-base font-semibold text-gray-900">Total</span>
                  <span className="text-lg font-bold text-gray-900">
                    {formatCurrency(grandTotal)}
                  </span>
                </div>
              </div>
            </div>

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

            <p className="mt-3 text-center text-xs text-gray-400">
              By placing your order you agree to our Terms &amp; Conditions
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

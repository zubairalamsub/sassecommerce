'use client';

import Link from 'next/link';
import { Minus, Plus, Trash2, ShoppingBag, ArrowRight } from 'lucide-react';
import { useCartStore } from '@/stores/cart';
import { formatCurrency } from '@/lib/utils';

const SHIPPING_COST = 100;

export default function CartPage() {
  const items = useCartStore((s) => s.items);
  const removeItem = useCartStore((s) => s.removeItem);
  const updateQuantity = useCartStore((s) => s.updateQuantity);
  const total = useCartStore((s) => s.total);
  const itemCount = useCartStore((s) => s.itemCount);

  const subtotal = total();
  const shipping = items.length > 0 ? SHIPPING_COST : 0;
  const grandTotal = subtotal + shipping;

  if (items.length === 0) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-20 text-center sm:px-6 lg:px-8">
        <div className="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-gray-100">
          <ShoppingBag className="h-10 w-10 text-gray-400" />
        </div>
        <h1 className="mt-6 text-2xl font-bold text-gray-900">Your cart is empty</h1>
        <p className="mt-2 text-gray-500">
          Looks like you haven&apos;t added anything to your cart yet.
        </p>
        <Link
          href="/products"
          className="mt-8 inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-3 font-medium text-white transition-colors hover:bg-primary-dark"
        >
          Continue Shopping
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold text-gray-900">Shopping Cart</h1>
      <p className="mt-1 text-gray-500">
        {itemCount()} {itemCount() === 1 ? 'item' : 'items'} in your cart
      </p>

      <div className="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-3">
        {/* Cart items */}
        <div className="lg:col-span-2">
          <div className="divide-y divide-gray-200 rounded-xl border border-gray-200 bg-white">
            {items.map((item) => (
              <div
                key={`${item.productId}-${item.variantId ?? ''}`}
                className="flex gap-4 p-4 sm:p-6"
              >
                {/* Image placeholder */}
                <div className="flex h-20 w-20 flex-shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-primary/20 to-primary/5">
                  <span className="text-2xl font-bold text-primary/40">
                    {item.name.charAt(0)}
                  </span>
                </div>

                {/* Item details */}
                <div className="flex flex-1 flex-col sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <h3 className="font-medium text-gray-900">{item.name}</h3>
                    <p className="mt-0.5 text-sm text-gray-500">SKU: {item.sku}</p>
                    <p className="mt-1 text-sm font-medium text-gray-700">
                      {formatCurrency(item.price)}
                    </p>
                  </div>

                  <div className="mt-3 flex items-center gap-4 sm:mt-0">
                    {/* Quantity controls */}
                    <div className="inline-flex items-center rounded-lg border border-gray-200">
                      <button
                        onClick={() =>
                          updateQuantity(item.productId, item.quantity - 1, item.variantId)
                        }
                        className="flex h-8 w-8 items-center justify-center text-gray-600 transition-colors hover:bg-gray-50"
                      >
                        <Minus className="h-3 w-3" />
                      </button>
                      <span className="flex h-8 w-10 items-center justify-center border-x border-gray-200 text-sm font-medium">
                        {item.quantity}
                      </span>
                      <button
                        onClick={() =>
                          updateQuantity(item.productId, item.quantity + 1, item.variantId)
                        }
                        className="flex h-8 w-8 items-center justify-center text-gray-600 transition-colors hover:bg-gray-50"
                      >
                        <Plus className="h-3 w-3" />
                      </button>
                    </div>

                    {/* Line total */}
                    <span className="min-w-[80px] text-right font-semibold text-gray-900">
                      {formatCurrency(item.price * item.quantity)}
                    </span>

                    {/* Remove button */}
                    <button
                      onClick={() => removeItem(item.productId, item.variantId)}
                      className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500"
                      title="Remove item"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <Link
            href="/products"
            className="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary transition-colors hover:text-primary-dark"
          >
            <ArrowRight className="h-4 w-4 rotate-180" />
            Continue Shopping
          </Link>
        </div>

        {/* Cart summary sidebar */}
        <div>
          <div className="rounded-xl border border-gray-200 bg-white p-6">
            <h2 className="text-lg font-semibold text-gray-900">Order Summary</h2>

            <div className="mt-4 space-y-3">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Subtotal</span>
                <span className="font-medium text-gray-900">
                  {formatCurrency(subtotal)}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Shipping</span>
                <span className="font-medium text-gray-900">
                  {formatCurrency(shipping)}
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

            <Link
              href="/checkout"
              className="mt-6 flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-6 py-3 font-medium text-white transition-colors hover:bg-primary-dark"
            >
              Proceed to Checkout
              <ArrowRight className="h-4 w-4" />
            </Link>

            <p className="mt-4 text-center text-xs text-gray-400">
              Shipping calculated as flat BDT 100 for all orders
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

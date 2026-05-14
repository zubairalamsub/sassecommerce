'use client';

import { useState, useEffect, useMemo, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft,
  Save,
  Loader2,
  Sparkles,
  BadgePercent,
  Ticket,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  promotionApi,
  type Coupon,
  type CreateCouponRequest,
  type CreatePromotionRequest,
  type DiscountType,
  type Promotion,
} from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { toast } from '@/stores/toast';

type FormKind = 'promotion' | 'coupon';

function isoForInput(d: Date): string {
  // <input type="datetime-local"> expects "YYYY-MM-DDTHH:mm" (local time, no tz).
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function inputToIso(s: string): string {
  // Convert "YYYY-MM-DDTHH:mm" (local) -> full ISO with tz (what Go time.Time parses).
  if (!s) return '';
  return new Date(s).toISOString();
}

function generateCode(): string {
  const chars = 'ABCDEFGHJKMNPQRSTUVWXYZ23456789';
  let out = '';
  for (let i = 0; i < 8; i++) {
    out += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return out;
}

export default function NewPromotionPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-[40vh] items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      }
    >
      <NewPromotionForm />
    </Suspense>
  );
}

function NewPromotionForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialKind: FormKind = searchParams.get('type') === 'coupon' ? 'coupon' : 'promotion';

  const { tenantId, token } = useAuthStore();
  const [kind, setKind] = useState<FormKind>(initialKind);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  // Shared promotion fields
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [discountType, setDiscountType] = useState<DiscountType>('percentage');
  const [discountValue, setDiscountValue] = useState('');
  const [appliesTo, setAppliesTo] = useState<'all' | 'categories' | 'products'>('all');
  const [appliesToList, setAppliesToList] = useState('');
  const [minOrderAmount, setMinOrderAmount] = useState('');
  const [maxDiscount, setMaxDiscount] = useState('');
  const [stackWithCoupons, setStackWithCoupons] = useState(false);

  const now = useMemo(() => new Date(), []);
  const defaultEnd = useMemo(() => {
    const d = new Date();
    d.setDate(d.getDate() + 30);
    return d;
  }, []);
  const [startDate, setStartDate] = useState(isoForInput(now));
  const [endDate, setEndDate] = useState(isoForInput(defaultEnd));
  const [status, setStatus] = useState<'draft' | 'active'>('active');

  // Coupon-only fields
  const [code, setCode] = useState('');
  const [promotionId, setPromotionId] = useState('');
  const [maxUses, setMaxUses] = useState('');
  const [maxUsesPerUser, setMaxUsesPerUser] = useState('1');
  const [customerEligibility, setCustomerEligibility] = useState<'all' | 'new' | 'specific'>('all');
  const [specificCustomers, setSpecificCustomers] = useState('');

  const [activePromotions, setActivePromotions] = useState<Promotion[]>([]);

  // Load active promotions so the coupon form can link to one.
  useEffect(() => {
    if (!tenantId) return;
    promotionApi
      .listActivePromotions(tenantId, token || undefined)
      .then((list) => setActivePromotions(Array.isArray(list) ? list : []))
      .catch(() => setActivePromotions([]));
  }, [tenantId, token]);

  function resetError() {
    if (error) setError('');
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!tenantId) {
      setError('Missing tenant context — please sign in again.');
      return;
    }
    setSaving(true);
    setError('');

    try {
      if (kind === 'promotion') {
        if (!name.trim()) throw new Error('Name is required.');
        const dv = Number(discountValue);
        if (!dv || dv <= 0) throw new Error('Discount value must be greater than 0.');
        if (!startDate || !endDate) throw new Error('Start and end dates are required.');
        if (new Date(endDate) <= new Date(startDate))
          throw new Error('End date must be after start date.');

        const payload: CreatePromotionRequest = {
          tenant_id: tenantId,
          name: name.trim(),
          description: description.trim() || undefined,
          discount_type: discountType,
          discount_value: dv,
          min_order_amount: minOrderAmount ? Number(minOrderAmount) : 0,
          max_discount: maxDiscount ? Number(maxDiscount) : 0,
          start_date: inputToIso(startDate),
          end_date: inputToIso(endDate),
        };

        await promotionApi.createPromotion(payload, tenantId, token || undefined);
        toast.success('Promotion created');
        router.push('/admin/promotions');
        return;
      }

      // Coupon
      if (!code.trim()) throw new Error('Coupon code is required.');
      if (code.trim().length < 3) throw new Error('Code must be at least 3 characters.');
      if (!promotionId) throw new Error('Select a promotion to attach this coupon to.');

      const payload: CreateCouponRequest = {
        tenant_id: tenantId,
        promotion_id: promotionId,
        code: code.trim().toUpperCase(),
        max_uses: maxUses ? Number(maxUses) : 0,
        max_uses_per_user: maxUsesPerUser ? Number(maxUsesPerUser) : 1,
      };

      const created = await promotionApi.createCoupon(payload, tenantId, token || undefined);

      // Cache locally so the list page can show it (no list endpoint exists).
      try {
        const raw = sessionStorage.getItem('admin:recent_coupons');
        const existing: Coupon[] = raw ? JSON.parse(raw) : [];
        sessionStorage.setItem(
          'admin:recent_coupons',
          JSON.stringify([created, ...existing].slice(0, 50)),
        );
      } catch {
        /* ignore */
      }

      toast.success('Coupon created');
      router.push('/admin/promotions');
    } catch (err) {
      const msg = (err as Error).message || 'Failed to save. Please try again.';
      setError(msg);
      toast.error(msg);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link
          href="/admin/promotions"
          className="rounded-lg p-2 text-text-muted transition-colors hover:bg-surface-hover hover:text-text"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-text">
            New {kind === 'promotion' ? 'Promotion' : 'Coupon'}
          </h1>
          <p className="text-sm text-text-secondary">
            {kind === 'promotion'
              ? 'Create a discount that applies automatically.'
              : 'Create a redeemable code customers enter at checkout.'}
          </p>
        </div>
      </div>

      {/* Kind toggle */}
      <div className="inline-flex rounded-lg border border-border bg-surface p-1 shadow-sm">
        <button
          type="button"
          onClick={() => setKind('promotion')}
          className={cn(
            'inline-flex items-center gap-2 rounded-md px-4 py-1.5 text-sm font-medium transition-colors',
            kind === 'promotion'
              ? 'bg-primary text-white shadow-sm'
              : 'text-text-secondary hover:text-text',
          )}
        >
          <BadgePercent className="h-4 w-4" />
          Promotion
        </button>
        <button
          type="button"
          onClick={() => setKind('coupon')}
          className={cn(
            'inline-flex items-center gap-2 rounded-md px-4 py-1.5 text-sm font-medium transition-colors',
            kind === 'coupon'
              ? 'bg-primary text-white shadow-sm'
              : 'text-text-secondary hover:text-text',
          )}
        >
          <Ticket className="h-4 w-4" />
          Coupon
        </button>
      </div>

      {error && (
        <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {kind === 'promotion' ? (
          <PromotionFields
            name={name}
            setName={(v) => {
              resetError();
              setName(v);
            }}
            description={description}
            setDescription={setDescription}
            discountType={discountType}
            setDiscountType={setDiscountType}
            discountValue={discountValue}
            setDiscountValue={setDiscountValue}
            appliesTo={appliesTo}
            setAppliesTo={setAppliesTo}
            appliesToList={appliesToList}
            setAppliesToList={setAppliesToList}
            minOrderAmount={minOrderAmount}
            setMinOrderAmount={setMinOrderAmount}
            maxDiscount={maxDiscount}
            setMaxDiscount={setMaxDiscount}
            startDate={startDate}
            setStartDate={setStartDate}
            endDate={endDate}
            setEndDate={setEndDate}
            status={status}
            setStatus={setStatus}
            stackWithCoupons={stackWithCoupons}
            setStackWithCoupons={setStackWithCoupons}
          />
        ) : (
          <CouponFields
            code={code}
            setCode={(v) => {
              resetError();
              setCode(v.toUpperCase());
            }}
            promotionId={promotionId}
            setPromotionId={setPromotionId}
            promotions={activePromotions}
            maxUses={maxUses}
            setMaxUses={setMaxUses}
            maxUsesPerUser={maxUsesPerUser}
            setMaxUsesPerUser={setMaxUsesPerUser}
            customerEligibility={customerEligibility}
            setCustomerEligibility={setCustomerEligibility}
            specificCustomers={specificCustomers}
            setSpecificCustomers={setSpecificCustomers}
            startDate={startDate}
            setStartDate={setStartDate}
            endDate={endDate}
            setEndDate={setEndDate}
            onGenerateCode={() => {
              resetError();
              setCode(generateCode());
            }}
          />
        )}

        {/* Actions */}
        <div className="flex items-center justify-between rounded-xl border border-border bg-surface p-6 shadow-sm">
          <Link
            href="/admin/promotions"
            className="rounded-lg border border-border px-4 py-2.5 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={saving}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:cursor-not-allowed disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            {saving ? 'Saving...' : `Create ${kind === 'promotion' ? 'Promotion' : 'Coupon'}`}
          </button>
        </div>
      </form>
    </div>
  );
}

/* ---------------- Promotion sub-form ---------------- */

interface PromotionFieldsProps {
  name: string;
  setName: (v: string) => void;
  description: string;
  setDescription: (v: string) => void;
  discountType: DiscountType;
  setDiscountType: (v: DiscountType) => void;
  discountValue: string;
  setDiscountValue: (v: string) => void;
  appliesTo: 'all' | 'categories' | 'products';
  setAppliesTo: (v: 'all' | 'categories' | 'products') => void;
  appliesToList: string;
  setAppliesToList: (v: string) => void;
  minOrderAmount: string;
  setMinOrderAmount: (v: string) => void;
  maxDiscount: string;
  setMaxDiscount: (v: string) => void;
  startDate: string;
  setStartDate: (v: string) => void;
  endDate: string;
  setEndDate: (v: string) => void;
  status: 'draft' | 'active';
  setStatus: (v: 'draft' | 'active') => void;
  stackWithCoupons: boolean;
  setStackWithCoupons: (v: boolean) => void;
}

function PromotionFields(p: PromotionFieldsProps) {
  return (
    <>
      <Section title="Details">
        <Field label="Name" required>
          <input
            type="text"
            required
            value={p.name}
            onChange={(e) => p.setName(e.target.value)}
            placeholder="e.g. Eid Sale 25% off"
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </Field>

        <Field label="Description">
          <textarea
            rows={3}
            value={p.description}
            onChange={(e) => p.setDescription(e.target.value)}
            placeholder="Shown to customers in the storefront banner..."
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </Field>
      </Section>

      <Section title="Discount">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Type" required>
            <select
              value={p.discountType}
              onChange={(e) => p.setDiscountType(e.target.value as DiscountType)}
              className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="percentage">Percentage off</option>
              <option value="fixed_amount">Fixed amount off</option>
              <option value="free_shipping">Free shipping</option>
            </select>
          </Field>

          <Field
            label={
              p.discountType === 'percentage'
                ? 'Discount (%)'
                : p.discountType === 'fixed_amount'
                  ? 'Discount amount (BDT)'
                  : 'Value (informational)'
            }
            required
          >
            <div className="relative">
              {p.discountType === 'fixed_amount' && (
                <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-sm text-text-muted">
                  ৳
                </span>
              )}
              {p.discountType === 'percentage' && (
                <span className="absolute right-3.5 top-1/2 -translate-y-1/2 text-sm text-text-muted">
                  %
                </span>
              )}
              <input
                type="number"
                min="0"
                step="0.01"
                required
                value={p.discountValue}
                onChange={(e) => p.setDiscountValue(e.target.value)}
                placeholder="0"
                className={cn(
                  'w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary',
                  p.discountType === 'fixed_amount' && 'pl-8',
                  p.discountType === 'percentage' && 'pr-8',
                )}
              />
            </div>
          </Field>
        </div>

        <Field label="Applies to">
          <select
            value={p.appliesTo}
            onChange={(e) => p.setAppliesTo(e.target.value as 'all' | 'categories' | 'products')}
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="all">All products</option>
            <option value="categories">Specific categories</option>
            <option value="products">Specific products</option>
          </select>
        </Field>
        {p.appliesTo !== 'all' && (
          <Field
            label={
              p.appliesTo === 'categories'
                ? 'Category IDs (comma-separated)'
                : 'Product IDs (comma-separated)'
            }
            hint="The backend currently stores these alongside the promotion; they are not yet enforced server-side."
          >
            <input
              type="text"
              value={p.appliesToList}
              onChange={(e) => p.setAppliesToList(e.target.value)}
              placeholder="cat_123, cat_456"
              className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 font-mono text-xs text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </Field>
        )}

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Min order amount (BDT)">
            <div className="relative">
              <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-sm text-text-muted">
                ৳
              </span>
              <input
                type="number"
                min="0"
                step="0.01"
                value={p.minOrderAmount}
                onChange={(e) => p.setMinOrderAmount(e.target.value)}
                placeholder="0"
                className="w-full rounded-lg border border-border bg-surface py-2.5 pl-8 pr-3.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </Field>
          {p.discountType === 'percentage' && (
            <Field label="Max discount cap (BDT)">
              <div className="relative">
                <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-sm text-text-muted">
                  ৳
                </span>
                <input
                  type="number"
                  min="0"
                  step="0.01"
                  value={p.maxDiscount}
                  onChange={(e) => p.setMaxDiscount(e.target.value)}
                  placeholder="No cap"
                  className="w-full rounded-lg border border-border bg-surface py-2.5 pl-8 pr-3.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </Field>
          )}
        </div>
      </Section>

      <Section title="Schedule & status">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Start date" required>
            <input
              type="datetime-local"
              required
              value={p.startDate}
              onChange={(e) => p.setStartDate(e.target.value)}
              className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </Field>
          <Field label="End date" required>
            <input
              type="datetime-local"
              required
              value={p.endDate}
              onChange={(e) => p.setEndDate(e.target.value)}
              className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </Field>
        </div>

        <Field label="Status">
          <select
            value={p.status}
            onChange={(e) => p.setStatus(e.target.value as 'draft' | 'active')}
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="active">Active</option>
            <option value="draft">Draft</option>
          </select>
        </Field>

        <label className="flex items-start gap-3 rounded-lg border border-border bg-surface-hover/40 p-3">
          <input
            type="checkbox"
            checked={p.stackWithCoupons}
            onChange={(e) => p.setStackWithCoupons(e.target.checked)}
            className="mt-0.5 h-4 w-4 rounded border-border text-primary focus:ring-primary"
          />
          <div>
            <div className="text-sm font-medium text-text">Allow stacking with coupon codes</div>
            <div className="text-xs text-text-muted">
              Customers can use a coupon on top of this promotion.
            </div>
          </div>
        </label>
      </Section>
    </>
  );
}

/* ---------------- Coupon sub-form ---------------- */

interface CouponFieldsProps {
  code: string;
  setCode: (v: string) => void;
  promotionId: string;
  setPromotionId: (v: string) => void;
  promotions: Promotion[];
  maxUses: string;
  setMaxUses: (v: string) => void;
  maxUsesPerUser: string;
  setMaxUsesPerUser: (v: string) => void;
  customerEligibility: 'all' | 'new' | 'specific';
  setCustomerEligibility: (v: 'all' | 'new' | 'specific') => void;
  specificCustomers: string;
  setSpecificCustomers: (v: string) => void;
  startDate: string;
  setStartDate: (v: string) => void;
  endDate: string;
  setEndDate: (v: string) => void;
  onGenerateCode: () => void;
}

function CouponFields(p: CouponFieldsProps) {
  return (
    <>
      <Section title="Code">
        <Field
          label="Coupon code"
          required
          hint="Uppercase letters and numbers, 3–50 characters."
        >
          <div className="flex gap-2">
            <input
              type="text"
              required
              minLength={3}
              maxLength={50}
              value={p.code}
              onChange={(e) => p.setCode(e.target.value)}
              placeholder="SAVE20"
              className="w-full flex-1 rounded-lg border border-border bg-surface px-3.5 py-2.5 font-mono text-sm uppercase text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              style={{ textTransform: 'uppercase' }}
            />
            <button
              type="button"
              onClick={p.onGenerateCode}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover hover:text-text"
            >
              <Sparkles className="h-4 w-4" />
              Generate
            </button>
          </div>
        </Field>

        <Field
          label="Linked promotion"
          required
          hint="Coupons inherit their discount value from the parent promotion."
        >
          <select
            required
            value={p.promotionId}
            onChange={(e) => p.setPromotionId(e.target.value)}
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="">Select an active promotion</option>
            {p.promotions.map((promo) => (
              <option key={promo.id} value={promo.id}>
                {promo.name} —{' '}
                {promo.discount_type === 'percentage'
                  ? `${promo.discount_value}% off`
                  : promo.discount_type === 'fixed_amount'
                    ? `৳${promo.discount_value} off`
                    : 'Free shipping'}
              </option>
            ))}
          </select>
          {p.promotions.length === 0 && (
            <p className="mt-1 text-xs text-text-muted">
              No active promotions yet —{' '}
              <Link href="/admin/promotions/new" className="text-primary hover:underline">
                create one first
              </Link>
              .
            </p>
          )}
        </Field>
      </Section>

      <Section title="Usage limits">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Total usage limit" hint="Leave blank for unlimited.">
            <input
              type="number"
              min="0"
              value={p.maxUses}
              onChange={(e) => p.setMaxUses(e.target.value)}
              placeholder="Unlimited"
              className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </Field>
          <Field label="Per-customer limit">
            <input
              type="number"
              min="1"
              value={p.maxUsesPerUser}
              onChange={(e) => p.setMaxUsesPerUser(e.target.value)}
              placeholder="1"
              className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </Field>
        </div>
      </Section>

      <Section title="Customer eligibility">
        <Field label="Who can use this coupon?">
          <select
            value={p.customerEligibility}
            onChange={(e) =>
              p.setCustomerEligibility(e.target.value as 'all' | 'new' | 'specific')
            }
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="all">All customers</option>
            <option value="new">New customers only</option>
            <option value="specific">Specific customers</option>
          </select>
        </Field>
        {p.customerEligibility === 'specific' && (
          <Field
            label="Customer IDs (comma-separated)"
            hint="Not yet enforced server-side — stored for future use."
          >
            <input
              type="text"
              value={p.specificCustomers}
              onChange={(e) => p.setSpecificCustomers(e.target.value)}
              placeholder="user_123, user_456"
              className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 font-mono text-xs text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </Field>
        )}
      </Section>

      <Section title="Schedule">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Start date">
            <input
              type="datetime-local"
              value={p.startDate}
              onChange={(e) => p.setStartDate(e.target.value)}
              className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </Field>
          <Field label="End date">
            <input
              type="datetime-local"
              value={p.endDate}
              onChange={(e) => p.setEndDate(e.target.value)}
              className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </Field>
        </div>
        <p className="text-xs text-text-muted">
          Coupon validity is currently controlled by the parent promotion&apos;s schedule.
        </p>
      </Section>
    </>
  );
}

/* ---------------- Shared layout helpers ---------------- */

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
      <h2 className="mb-4 text-lg font-semibold text-text">{title}</h2>
      <div className="space-y-4">{children}</div>
    </div>
  );
}

function Field({
  label,
  required,
  hint,
  children,
}: {
  label: string;
  required?: boolean;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-text">
        {label} {required && <span className="text-red-500">*</span>}
      </label>
      {children}
      {hint && <p className="mt-1 text-xs text-text-muted">{hint}</p>}
    </div>
  );
}

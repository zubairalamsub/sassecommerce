'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { ArrowLeft, Plus, MapPin, Star, Pencil, Trash2, X } from 'lucide-react';
import AuthGuard from '@/components/auth/auth-guard';
import { useAddressStore, type SavedAddress } from '@/stores/addresses';
import { cn } from '@/lib/utils';

const emptyForm = { label: '', street: '', city: '', state: '', postalCode: '', country: 'Bangladesh', phone: '', isDefault: false };

function AddressForm({ initial, onSave, onCancel }: {
  initial?: SavedAddress;
  onSave: (data: Omit<SavedAddress, 'id'>) => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState(initial ? {
    label: initial.label, street: initial.street, city: initial.city,
    state: initial.state, postalCode: initial.postalCode, country: initial.country,
    phone: initial.phone, isDefault: initial.isDefault,
  } : emptyForm);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.street.trim() || !form.city.trim()) return;
    onSave(form);
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-xl border border-border bg-surface p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-text">{initial ? 'Edit Address' : 'New Address'}</h3>
        <button type="button" onClick={onCancel} className="rounded-lg p-1.5 text-text-muted hover:bg-surface-hover">
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label className="mb-1.5 block text-sm font-medium text-text-secondary">Label</label>
          <input value={form.label} onChange={(e) => setForm({ ...form, label: e.target.value })} placeholder="Home, Office, etc."
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
        </div>
        <div>
          <label className="mb-1.5 block text-sm font-medium text-text-secondary">Phone</label>
          <input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} placeholder="+880 1712-345678"
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
        </div>
      </div>
      <div>
        <label className="mb-1.5 block text-sm font-medium text-text-secondary">Street Address *</label>
        <input required value={form.street} onChange={(e) => setForm({ ...form, street: e.target.value })} placeholder="House 12, Road 5, Dhanmondi"
          className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div>
          <label className="mb-1.5 block text-sm font-medium text-text-secondary">City *</label>
          <input required value={form.city} onChange={(e) => setForm({ ...form, city: e.target.value })} placeholder="Dhaka"
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
        </div>
        <div>
          <label className="mb-1.5 block text-sm font-medium text-text-secondary">State / Division</label>
          <input value={form.state} onChange={(e) => setForm({ ...form, state: e.target.value })} placeholder="Dhaka Division"
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
        </div>
        <div>
          <label className="mb-1.5 block text-sm font-medium text-text-secondary">Postal Code</label>
          <input value={form.postalCode} onChange={(e) => setForm({ ...form, postalCode: e.target.value })} placeholder="1205"
            className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
        </div>
      </div>
      <label className="flex items-center gap-2 text-sm text-text-secondary cursor-pointer">
        <input type="checkbox" checked={form.isDefault} onChange={(e) => setForm({ ...form, isDefault: e.target.checked })}
          className="h-4 w-4 rounded border-border text-primary focus:ring-primary" />
        Set as default address
      </label>
      <div className="flex gap-3 pt-2">
        <button type="submit"
          className="rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-white hover:bg-primary-dark transition-colors">
          {initial ? 'Update Address' : 'Save Address'}
        </button>
        <button type="button" onClick={onCancel}
          className="rounded-lg border border-border px-4 py-2.5 text-sm font-medium text-text-secondary hover:bg-surface-hover transition-colors">
          Cancel
        </button>
      </div>
    </form>
  );
}

function AddressContent() {
  const { addresses, addAddress, updateAddress, removeAddress, setDefault } = useAddressStore();
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => { setMounted(true); }, []);

  if (!mounted) return null;

  const editingAddress = editingId ? addresses.find((a) => a.id === editingId) : undefined;

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6 lg:px-8">
      <Link href="/account" className="inline-flex items-center gap-1.5 text-sm text-text-secondary hover:text-text transition-colors mb-6">
        <ArrowLeft className="h-4 w-4" /> Back to Account
      </Link>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-text">My Addresses</h1>
          <p className="mt-1 text-sm text-text-secondary">{addresses.length} saved address{addresses.length !== 1 ? 'es' : ''}</p>
        </div>
        {!showForm && !editingId && (
          <button onClick={() => setShowForm(true)}
            className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-dark transition-colors">
            <Plus className="h-4 w-4" /> Add Address
          </button>
        )}
      </div>

      {showForm && (
        <div className="mb-6">
          <AddressForm onSave={(data) => { addAddress(data); setShowForm(false); }} onCancel={() => setShowForm(false)} />
        </div>
      )}

      {editingId && editingAddress && (
        <div className="mb-6">
          <AddressForm initial={editingAddress} onSave={(data) => { updateAddress(editingId, data); if (data.isDefault) setDefault(editingId); setEditingId(null); }} onCancel={() => setEditingId(null)} />
        </div>
      )}

      {addresses.length === 0 && !showForm ? (
        <div className="rounded-2xl border border-border bg-surface py-16 text-center">
          <MapPin className="mx-auto h-12 w-12 text-text-muted" />
          <p className="mt-4 text-lg font-medium text-text">No addresses saved</p>
          <p className="mt-1 text-sm text-text-secondary">Add an address for faster checkout</p>
          <button onClick={() => setShowForm(true)}
            className="mt-6 rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-white hover:bg-primary-dark transition-colors">
            Add Your First Address
          </button>
        </div>
      ) : (
        <div className="space-y-4">
          {addresses.map((addr) => (
            <div key={addr.id} className={cn('rounded-xl border bg-surface p-5 transition-colors', addr.isDefault ? 'border-primary' : 'border-border')}>
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-3">
                  <MapPin className="h-5 w-5 text-text-muted mt-0.5 flex-shrink-0" />
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-semibold text-text">{addr.label || 'Address'}</p>
                      {addr.isDefault && (
                        <span className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                          <Star className="h-3 w-3" /> Default
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-sm text-text-secondary">{addr.street}</p>
                    <p className="text-sm text-text-secondary">{addr.city}{addr.state ? `, ${addr.state}` : ''} {addr.postalCode}</p>
                    <p className="text-sm text-text-secondary">{addr.country}</p>
                    {addr.phone && <p className="mt-1 text-xs text-text-muted">{addr.phone}</p>}
                  </div>
                </div>
                <div className="flex items-center gap-1">
                  {!addr.isDefault && (
                    <button onClick={() => setDefault(addr.id)} title="Set as default"
                      className="rounded-lg p-2 text-text-muted hover:bg-surface-hover hover:text-primary transition-colors">
                      <Star className="h-4 w-4" />
                    </button>
                  )}
                  <button onClick={() => setEditingId(addr.id)} title="Edit"
                    className="rounded-lg p-2 text-text-muted hover:bg-surface-hover hover:text-text transition-colors">
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button onClick={() => removeAddress(addr.id)} title="Delete"
                    className="rounded-lg p-2 text-text-muted hover:bg-surface-hover hover:text-red-500 transition-colors">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function AddressesPage() {
  return (
    <AuthGuard requiredRole="customer">
      <AddressContent />
    </AuthGuard>
  );
}

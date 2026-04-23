'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface SavedAddress {
  id: string;
  label: string;
  street: string;
  city: string;
  state: string;
  postalCode: string;
  country: string;
  phone: string;
  isDefault: boolean;
}

interface AddressState {
  addresses: SavedAddress[];
  addAddress: (address: Omit<SavedAddress, 'id'>) => void;
  updateAddress: (id: string, data: Partial<SavedAddress>) => void;
  removeAddress: (id: string) => void;
  setDefault: (id: string) => void;
  getDefault: () => SavedAddress | undefined;
}

export const useAddressStore = create<AddressState>()(
  persist(
    (set, get) => ({
      addresses: [],

      addAddress: (data) => {
        const address: SavedAddress = { ...data, id: `addr-${Date.now()}` };
        if (data.isDefault || get().addresses.length === 0) {
          set((state) => ({
            addresses: [
              ...state.addresses.map((a) => ({ ...a, isDefault: false })),
              { ...address, isDefault: true },
            ],
          }));
        } else {
          set((state) => ({ addresses: [...state.addresses, address] }));
        }
      },

      updateAddress: (id, data) => {
        set((state) => ({
          addresses: state.addresses.map((a) => (a.id === id ? { ...a, ...data } : a)),
        }));
      },

      removeAddress: (id) => {
        const removing = get().addresses.find((a) => a.id === id);
        set((state) => {
          const remaining = state.addresses.filter((a) => a.id !== id);
          if (removing?.isDefault && remaining.length > 0) {
            remaining[0].isDefault = true;
          }
          return { addresses: remaining };
        });
      },

      setDefault: (id) => {
        set((state) => ({
          addresses: state.addresses.map((a) => ({ ...a, isDefault: a.id === id })),
        }));
      },

      getDefault: () => get().addresses.find((a) => a.isDefault),
    }),
    { name: 'addresses-storage' },
  ),
);

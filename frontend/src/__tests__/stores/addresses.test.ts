import { useAddressStore } from '@/stores/addresses';
import { act } from '@testing-library/react';

const homeAddress = {
  label: 'Home',
  street: '123 Mirpur Road',
  city: 'Dhaka',
  state: 'Dhaka',
  postalCode: '1216',
  country: 'Bangladesh',
  phone: '+880-1700-000000',
  isDefault: false,
};

const officeAddress = {
  label: 'Office',
  street: '456 Gulshan Ave',
  city: 'Dhaka',
  state: 'Dhaka',
  postalCode: '1212',
  country: 'Bangladesh',
  phone: '+880-1800-000000',
  isDefault: false,
};

beforeEach(() => {
  act(() => {
    useAddressStore.setState({ addresses: [] });
  });
});

describe('Address Store', () => {
  test('starts with empty addresses', () => {
    expect(useAddressStore.getState().addresses).toHaveLength(0);
  });

  test('adds an address with generated id', () => {
    act(() => {
      useAddressStore.getState().addAddress(homeAddress);
    });

    const addrs = useAddressStore.getState().addresses;
    expect(addrs).toHaveLength(1);
    expect(addrs[0].label).toBe('Home');
    expect(addrs[0].id).toMatch(/^addr-/);
  });

  test('first address is automatically set as default', () => {
    act(() => {
      useAddressStore.getState().addAddress({ ...homeAddress, isDefault: false });
    });

    expect(useAddressStore.getState().addresses[0].isDefault).toBe(true);
  });

  test('adding with isDefault clears other defaults', () => {
    act(() => {
      useAddressStore.getState().addAddress(homeAddress);
      useAddressStore.getState().addAddress({ ...officeAddress, isDefault: true });
    });

    const addrs = useAddressStore.getState().addresses;
    expect(addrs).toHaveLength(2);
    // First should no longer be default
    expect(addrs[0].isDefault).toBe(false);
    // Second should be default
    expect(addrs[1].isDefault).toBe(true);
  });

  test('second address without isDefault keeps first as default', () => {
    act(() => {
      useAddressStore.getState().addAddress(homeAddress);
      useAddressStore.getState().addAddress(officeAddress);
    });

    const addrs = useAddressStore.getState().addresses;
    expect(addrs[0].isDefault).toBe(true);
    expect(addrs[1].isDefault).toBe(false);
  });

  test('updates an address', () => {
    act(() => {
      useAddressStore.getState().addAddress(homeAddress);
    });

    const id = useAddressStore.getState().addresses[0].id;

    act(() => {
      useAddressStore.getState().updateAddress(id, { street: '789 New Road', city: 'Chittagong' });
    });

    const updated = useAddressStore.getState().addresses[0];
    expect(updated.street).toBe('789 New Road');
    expect(updated.city).toBe('Chittagong');
    expect(updated.label).toBe('Home'); // unchanged fields preserved
  });

  test('removes an address', () => {
    // Seed with unique IDs to avoid Date.now() collision
    act(() => {
      useAddressStore.setState({
        addresses: [
          { ...homeAddress, id: 'addr-home', isDefault: true },
          { ...officeAddress, id: 'addr-office', isDefault: false },
        ],
      });
    });

    act(() => {
      useAddressStore.getState().removeAddress('addr-home');
    });

    expect(useAddressStore.getState().addresses).toHaveLength(1);
    expect(useAddressStore.getState().addresses[0].label).toBe('Office');
  });

  test('removing default promotes next address', () => {
    act(() => {
      useAddressStore.setState({
        addresses: [
          { ...homeAddress, id: 'addr-home', isDefault: true },
          { ...officeAddress, id: 'addr-office', isDefault: false },
        ],
      });
    });

    act(() => {
      useAddressStore.getState().removeAddress('addr-home');
    });

    const remaining = useAddressStore.getState().addresses;
    expect(remaining).toHaveLength(1);
    expect(remaining[0].isDefault).toBe(true);
    expect(remaining[0].label).toBe('Office');
  });

  test('setDefault changes the default address', () => {
    act(() => {
      useAddressStore.setState({
        addresses: [
          { ...homeAddress, id: 'addr-home', isDefault: true },
          { ...officeAddress, id: 'addr-office', isDefault: false },
        ],
      });
    });

    act(() => {
      useAddressStore.getState().setDefault('addr-office');
    });

    const addrs = useAddressStore.getState().addresses;
    expect(addrs[0].isDefault).toBe(false);
    expect(addrs[1].isDefault).toBe(true);
  });

  test('getDefault returns the default address', () => {
    act(() => {
      useAddressStore.getState().addAddress(homeAddress);
      useAddressStore.getState().addAddress(officeAddress);
    });

    const defaultAddr = useAddressStore.getState().getDefault();
    expect(defaultAddr?.label).toBe('Home');
  });

  test('getDefault returns undefined when no addresses', () => {
    expect(useAddressStore.getState().getDefault()).toBeUndefined();
  });
});

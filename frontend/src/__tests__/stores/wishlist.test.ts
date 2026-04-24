import { useWishlistStore } from '@/stores/wishlist';
import { act } from '@testing-library/react';

const sampleItem = {
  productId: 'p1',
  name: 'Test Product',
  slug: 'test-product',
  price: 1500,
  image: '/test.jpg',
};

const sampleItem2 = {
  productId: 'p2',
  name: 'Another Product',
  slug: 'another-product',
  price: 2500,
};

beforeEach(() => {
  act(() => {
    useWishlistStore.getState().clear();
  });
});

describe('Wishlist Store', () => {
  test('starts with empty wishlist', () => {
    expect(useWishlistStore.getState().items).toHaveLength(0);
  });

  test('adds an item', () => {
    act(() => {
      useWishlistStore.getState().addItem(sampleItem);
    });

    const items = useWishlistStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0].productId).toBe('p1');
    expect(items[0].name).toBe('Test Product');
    expect(items[0].addedAt).toBeTruthy();
  });

  test('prevents duplicate items', () => {
    act(() => {
      useWishlistStore.getState().addItem(sampleItem);
      useWishlistStore.getState().addItem(sampleItem);
    });

    expect(useWishlistStore.getState().items).toHaveLength(1);
  });

  test('removes an item', () => {
    act(() => {
      useWishlistStore.getState().addItem(sampleItem);
      useWishlistStore.getState().addItem(sampleItem2);
      useWishlistStore.getState().removeItem('p1');
    });

    const items = useWishlistStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0].productId).toBe('p2');
  });

  test('toggleItem adds when not present', () => {
    act(() => {
      useWishlistStore.getState().toggleItem(sampleItem);
    });

    expect(useWishlistStore.getState().items).toHaveLength(1);
  });

  test('toggleItem removes when present', () => {
    act(() => {
      useWishlistStore.getState().addItem(sampleItem);
      useWishlistStore.getState().toggleItem(sampleItem);
    });

    expect(useWishlistStore.getState().items).toHaveLength(0);
  });

  test('isInWishlist returns correct boolean', () => {
    act(() => {
      useWishlistStore.getState().addItem(sampleItem);
    });

    expect(useWishlistStore.getState().isInWishlist('p1')).toBe(true);
    expect(useWishlistStore.getState().isInWishlist('p999')).toBe(false);
  });

  test('clear empties the wishlist', () => {
    act(() => {
      useWishlistStore.getState().addItem(sampleItem);
      useWishlistStore.getState().addItem(sampleItem2);
      useWishlistStore.getState().clear();
    });

    expect(useWishlistStore.getState().items).toHaveLength(0);
  });
});

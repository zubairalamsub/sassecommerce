import { useCartStore, CartItem } from "@/stores/cart";
import { act } from "@testing-library/react";

const sampleItem: CartItem = {
  productId: "p1",
  name: "Test Shirt",
  sku: "TSH-001",
  price: 500,
  quantity: 1,
  image: "/test.jpg",
};

const sampleItemVariant: CartItem = {
  productId: "p1",
  variantId: "v-red",
  name: "Test Shirt - Red",
  sku: "TSH-001-R",
  price: 550,
  quantity: 2,
};

beforeEach(() => {
  act(() => {
    useCartStore.getState().clearCart();
  });
});

describe("Cart Store", () => {
  test("starts with empty cart", () => {
    const state = useCartStore.getState();
    expect(state.items).toHaveLength(0);
    expect(state.total()).toBe(0);
    expect(state.itemCount()).toBe(0);
  });

  test("adds an item to cart", () => {
    act(() => {
      useCartStore.getState().addItem(sampleItem);
    });

    const state = useCartStore.getState();
    expect(state.items).toHaveLength(1);
    expect(state.items[0].productId).toBe("p1");
    expect(state.items[0].quantity).toBe(1);
  });

  test("increments quantity when adding same product", () => {
    act(() => {
      useCartStore.getState().addItem(sampleItem);
      useCartStore.getState().addItem({ ...sampleItem, quantity: 3 });
    });

    const state = useCartStore.getState();
    expect(state.items).toHaveLength(1);
    expect(state.items[0].quantity).toBe(4);
  });

  test("treats different variants as separate items", () => {
    act(() => {
      useCartStore.getState().addItem(sampleItem);
      useCartStore.getState().addItem(sampleItemVariant);
    });

    const state = useCartStore.getState();
    expect(state.items).toHaveLength(2);
  });

  test("removes an item from cart", () => {
    act(() => {
      useCartStore.getState().addItem(sampleItem);
      useCartStore.getState().addItem(sampleItemVariant);
      useCartStore.getState().removeItem("p1", undefined);
    });

    const state = useCartStore.getState();
    expect(state.items).toHaveLength(1);
    expect(state.items[0].variantId).toBe("v-red");
  });

  test("updates quantity of an item", () => {
    act(() => {
      useCartStore.getState().addItem(sampleItem);
      useCartStore.getState().updateQuantity("p1", 5);
    });

    expect(useCartStore.getState().items[0].quantity).toBe(5);
  });

  test("removes item when quantity set to 0", () => {
    act(() => {
      useCartStore.getState().addItem(sampleItem);
      useCartStore.getState().updateQuantity("p1", 0);
    });

    expect(useCartStore.getState().items).toHaveLength(0);
  });

  test("calculates total correctly", () => {
    act(() => {
      useCartStore.getState().addItem(sampleItem); // 500 x 1
      useCartStore.getState().addItem(sampleItemVariant); // 550 x 2
    });

    expect(useCartStore.getState().total()).toBe(1600);
  });

  test("calculates item count correctly", () => {
    act(() => {
      useCartStore.getState().addItem(sampleItem); // qty 1
      useCartStore.getState().addItem(sampleItemVariant); // qty 2
    });

    expect(useCartStore.getState().itemCount()).toBe(3);
  });

  test("clears entire cart", () => {
    act(() => {
      useCartStore.getState().addItem(sampleItem);
      useCartStore.getState().addItem(sampleItemVariant);
      useCartStore.getState().clearCart();
    });

    const state = useCartStore.getState();
    expect(state.items).toHaveLength(0);
    expect(state.total()).toBe(0);
    expect(state.itemCount()).toBe(0);
  });
});

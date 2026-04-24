import { useToastStore } from "@/stores/toast";
import { act } from "@testing-library/react";

beforeEach(() => {
  act(() => {
    // Clear all toasts
    const toasts = useToastStore.getState().toasts;
    toasts.forEach((t) => useToastStore.getState().removeToast(t.id));
  });
  jest.useFakeTimers();
});

afterEach(() => {
  jest.useRealTimers();
});

describe("Toast Store", () => {
  test("starts with no toasts", () => {
    expect(useToastStore.getState().toasts).toHaveLength(0);
  });

  test("adds a toast", () => {
    act(() => {
      useToastStore.getState().addToast("success", "Item added to cart");
    });

    const toasts = useToastStore.getState().toasts;
    expect(toasts).toHaveLength(1);
    expect(toasts[0].type).toBe("success");
    expect(toasts[0].message).toBe("Item added to cart");
  });

  test("adds multiple toasts", () => {
    act(() => {
      useToastStore.getState().addToast("success", "Success!");
      useToastStore.getState().addToast("error", "Error!");
      useToastStore.getState().addToast("warning", "Warning!");
    });

    expect(useToastStore.getState().toasts).toHaveLength(3);
  });

  test("removes a toast by id", () => {
    act(() => {
      useToastStore.getState().addToast("info", "Test message");
    });

    const id = useToastStore.getState().toasts[0].id;

    act(() => {
      useToastStore.getState().removeToast(id);
    });

    expect(useToastStore.getState().toasts).toHaveLength(0);
  });

  test("auto-removes toast after duration", () => {
    act(() => {
      useToastStore.getState().addToast("success", "Temporary", 2000);
    });

    expect(useToastStore.getState().toasts).toHaveLength(1);

    act(() => {
      jest.advanceTimersByTime(2000);
    });

    expect(useToastStore.getState().toasts).toHaveLength(0);
  });

  test("toast with duration 0 does not auto-remove", () => {
    act(() => {
      useToastStore.getState().addToast("error", "Persistent error", 0);
    });

    act(() => {
      jest.advanceTimersByTime(10000);
    });

    expect(useToastStore.getState().toasts).toHaveLength(1);
    expect(useToastStore.getState().toasts[0].message).toBe("Persistent error");
  });

  test("each toast gets a unique id", () => {
    act(() => {
      useToastStore.getState().addToast("success", "One");
      useToastStore.getState().addToast("success", "Two");
    });

    const toasts = useToastStore.getState().toasts;
    expect(toasts[0].id).not.toBe(toasts[1].id);
  });
});

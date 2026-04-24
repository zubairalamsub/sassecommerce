import { useAuthStore, demoLogin, DEMO_USERS } from "@/stores/auth";
import { act } from "@testing-library/react";

beforeEach(() => {
  act(() => {
    useAuthStore.getState().logout();
  });
});

describe("Auth Store", () => {
  test("starts unauthenticated", () => {
    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.token).toBeNull();
    expect(state.isAuthenticated()).toBe(false);
  });

  test("setAuth sets user, token, and tenantId", () => {
    const user = DEMO_USERS["admin@fashion.com.bd"].user;
    act(() => {
      useAuthStore.getState().setAuth(user, "test-token", "tenant_saajan");
    });

    const state = useAuthStore.getState();
    expect(state.user?.email).toBe("admin@fashion.com.bd");
    expect(state.token).toBe("test-token");
    expect(state.tenantId).toBe("tenant_saajan");
    expect(state.isAuthenticated()).toBe(true);
  });

  test("logout clears all auth state", () => {
    const user = DEMO_USERS["admin@fashion.com.bd"].user;
    act(() => {
      useAuthStore.getState().setAuth(user, "test-token", "tenant_saajan");
      useAuthStore.getState().logout();
    });

    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.token).toBeNull();
    expect(state.tenantId).toBeNull();
    expect(state.isAuthenticated()).toBe(false);
  });

  test("hasRole checks role hierarchy correctly", () => {
    const admin = DEMO_USERS["admin@fashion.com.bd"].user;
    act(() => {
      useAuthStore.getState().setAuth(admin, "token", "t1");
    });

    const state = useAuthStore.getState();
    // Admin (80) >= customer (40) → true
    expect(state.hasRole("customer")).toBe(true);
    // Admin (80) >= moderator (60) → true
    expect(state.hasRole("moderator")).toBe(true);
    // Admin (80) >= admin (80) → true
    expect(state.hasRole("admin")).toBe(true);
    // Admin (80) >= super_admin (100) → false
    expect(state.hasRole("super_admin")).toBe(false);
  });

  test("isSuperAdmin returns true only for super_admin", () => {
    const superAdmin = DEMO_USERS["super@saajan.com.bd"].user;
    act(() => {
      useAuthStore.getState().setAuth(superAdmin, "token", null);
    });
    expect(useAuthStore.getState().isSuperAdmin()).toBe(true);

    const admin = DEMO_USERS["admin@fashion.com.bd"].user;
    act(() => {
      useAuthStore.getState().setAuth(admin, "token", "t1");
    });
    expect(useAuthStore.getState().isSuperAdmin()).toBe(false);
  });

  test("isTenantAdmin returns true for admin and super_admin", () => {
    const admin = DEMO_USERS["admin@fashion.com.bd"].user;
    act(() => {
      useAuthStore.getState().setAuth(admin, "token", "t1");
    });
    expect(useAuthStore.getState().isTenantAdmin()).toBe(true);

    const customer = DEMO_USERS["rahim@example.com"].user;
    act(() => {
      useAuthStore.getState().setAuth(customer, "token", "t1");
    });
    expect(useAuthStore.getState().isTenantAdmin()).toBe(false);
  });

  test("isStaff returns true for moderator and above", () => {
    const mod = DEMO_USERS["staff@fashion.com.bd"].user;
    act(() => {
      useAuthStore.getState().setAuth(mod, "token", "t1");
    });
    expect(useAuthStore.getState().isStaff()).toBe(true);

    const customer = DEMO_USERS["rahim@example.com"].user;
    act(() => {
      useAuthStore.getState().setAuth(customer, "token", "t1");
    });
    expect(useAuthStore.getState().isStaff()).toBe(false);
  });

  test("isCustomer returns true only for customer role", () => {
    const customer = DEMO_USERS["rahim@example.com"].user;
    act(() => {
      useAuthStore.getState().setAuth(customer, "token", "t1");
    });
    expect(useAuthStore.getState().isCustomer()).toBe(true);

    const admin = DEMO_USERS["admin@fashion.com.bd"].user;
    act(() => {
      useAuthStore.getState().setAuth(admin, "token", "t1");
    });
    expect(useAuthStore.getState().isCustomer()).toBe(false);
  });

  test("hasRole returns false when not authenticated", () => {
    expect(useAuthStore.getState().hasRole("customer")).toBe(false);
  });
});

describe("demoLogin", () => {
  test("returns user and token for valid demo credentials", () => {
    const result = demoLogin("admin@fashion.com.bd", "admin123");
    expect(result).not.toBeNull();
    expect(result!.user.email).toBe("admin@fashion.com.bd");
    expect(result!.user.role).toBe("admin");
    expect(result!.token).toBe("demo-admin-token-t1");
  });

  test("returns null for wrong password", () => {
    const result = demoLogin("admin@fashion.com.bd", "wrongpass");
    expect(result).toBeNull();
  });

  test("returns null for unknown email", () => {
    const result = demoLogin("unknown@test.com", "password");
    expect(result).toBeNull();
  });
});

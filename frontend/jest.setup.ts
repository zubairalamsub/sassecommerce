import "@testing-library/jest-dom";

// jsdom doesn't ship `fetch`. Default it to a safe 404 stub so tests that
// don't care about the network don't blow up; tests that DO exercise fetch
// override with their own mock.
if (typeof (globalThis as { fetch?: unknown }).fetch === "undefined") {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (globalThis as any).fetch = jest.fn(async () => ({
    ok: false,
    status: 404,
    statusText: "Not Found",
    headers: new Map([["Content-Type", "application/json"]]),
    json: async () => ({ error: "test stub" }),
    text: async () => '{"error":"test stub"}',
  }));
}

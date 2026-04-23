import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { useAuth } from "./useAuth";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useAuth", () => {
  const originalFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("fetches current user", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({
        username: "admin",
        email: "admin@test.com",
        display_name: "Admin",
        admin: true,
      }),
    })) as unknown as typeof fetch;

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.username).toBe("admin");
    expect(result.current.data?.admin).toBe(true);
  });

  it("handles auth failure", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: false,
      status: 401,
      json: async () => ({}),
    })) as unknown as typeof fetch;

    // Prevent the OIDC redirect
    delete (window as Record<string, unknown>).location;
    (window as Record<string, unknown>).location = { href: "" } as Location;

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

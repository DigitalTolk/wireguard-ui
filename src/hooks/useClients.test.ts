import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { useClients, useCreateClient, useDeleteClient, useSetClientStatus } from "./useClients";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useClients", () => {
  const originalFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("fetches client list", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => [
        { Client: { id: "c1", name: "Client 1" }, QRCode: "" },
      ],
    })) as unknown as typeof fetch;

    const { result } = renderHook(() => useClients(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data![0].Client.name).toBe("Client 1");
  });

  it("creates a client", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 201,
      json: async () => ({ id: "new1", name: "New" }),
    })) as unknown as typeof fetch;

    const { result } = renderHook(() => useCreateClient(), { wrapper: createWrapper() });

    result.current.mutate({ name: "New", allocated_ips: ["10.0.0.2/32"], allowed_ips: ["0.0.0.0/0"] });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("deletes a client", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 204,
      json: async () => undefined,
    })) as unknown as typeof fetch;

    const { result } = renderHook(() => useDeleteClient(), { wrapper: createWrapper() });

    result.current.mutate("c1");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("sets client status", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({ id: "c1", enabled: false }),
    })) as unknown as typeof fetch;

    const { result } = renderHook(() => useSetClientStatus(), { wrapper: createWrapper() });

    result.current.mutate({ id: "c1", enabled: false });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

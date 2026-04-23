import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiGet, apiPost, apiPut, apiPatch, apiDelete, ApiError } from "./api-client";

describe("api-client", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    // prevent actual OIDC redirect
    delete (window as Record<string, unknown>).location;
    (window as Record<string, unknown>).location = { href: "" } as Location;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("apiGet sends GET request with correct headers", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({ data: "test" }),
    })) as unknown as typeof fetch;

    const result = await apiGet<{ data: string }>("/test");
    expect(result.data).toBe("test");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/test"),
      expect.objectContaining({ method: "GET" })
    );
  });

  it("apiPost sends POST with JSON body", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({ created: true }),
    })) as unknown as typeof fetch;

    const result = await apiPost<{ created: boolean }>("/items", { name: "x" });
    expect(result.created).toBe(true);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/items"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ name: "x" }),
      })
    );
  });

  it("apiPut sends PUT with JSON body", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({ updated: true }),
    })) as unknown as typeof fetch;

    await apiPut("/items/1", { name: "y" });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/items/1"),
      expect.objectContaining({ method: "PUT" })
    );
  });

  it("apiPatch sends PATCH request", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({}),
    })) as unknown as typeof fetch;

    await apiPatch("/items/1/status", { enabled: true });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/items/1/status"),
      expect.objectContaining({ method: "PATCH" })
    );
  });

  it("apiDelete sends DELETE request", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 204,
      json: async () => undefined,
    })) as unknown as typeof fetch;

    await apiDelete("/items/1");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/items/1"),
      expect.objectContaining({ method: "DELETE" })
    );
  });

  it("throws ApiError on non-OK response", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: false,
      status: 400,
      json: async () => ({
        error: { code: "BAD_REQUEST", message: "Invalid input" },
      }),
    })) as unknown as typeof fetch;

    await expect(apiGet("/bad")).rejects.toThrow(ApiError);
    await expect(apiGet("/bad")).rejects.toThrow("Invalid input");
  });

  it("throws ApiError with fallback message on non-JSON error response", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: false,
      status: 500,
      json: async () => {
        throw new Error("not json");
      },
    })) as unknown as typeof fetch;

    await expect(apiGet("/err")).rejects.toThrow("Request failed with status 500");
  });

  it("redirects to OIDC login on 401", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: false,
      status: 401,
      json: async () => ({}),
    })) as unknown as typeof fetch;

    await expect(apiGet("/protected")).rejects.toThrow("Not authenticated");
    expect(window.location.href).toContain("oidc/login");
  });

  it("apiPost without body sends no body", async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({}),
    })) as unknown as typeof fetch;

    await apiPost("/action");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ body: undefined })
    );
  });
});

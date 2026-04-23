import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import App from "./App";

// Mock matchMedia for ThemeProvider
window.matchMedia = vi.fn().mockReturnValue({
  matches: false,
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
});

// Mock fetch for the auth/me call and clients list
globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
  const url = typeof input === "string" ? input : input.toString();
  if (url.includes("/auth/me")) {
    return {
      ok: true,
      status: 200,
      json: async () => ({
        username: "admin",
        email: "admin@test.com",
        display_name: "Admin",
        admin: true,
      }),
    } as Response;
  }
  if (url.includes("/subnet-ranges")) {
    return { ok: true, status: 200, json: async () => [] } as Response;
  }
  if (url.includes("/clients")) {
    return { ok: true, status: 200, json: async () => [] } as Response;
  }
  return { ok: true, status: 200, json: async () => ({}) } as Response;
}) as unknown as typeof fetch;

describe("App", () => {
  it("renders the app shell with sidebar", async () => {
    render(<App />);
    await waitFor(() => {
      expect(screen.getAllByText("WireGuard UI").length).toBeGreaterThan(0);
    });
  });

  it("shows navigation links", async () => {
    render(<App />);
    await waitFor(() => {
      expect(screen.getAllByText("Clients").length).toBeGreaterThan(0);
      expect(screen.getAllByText("Status").length).toBeGreaterThan(0);
      expect(screen.getAllByText("About").length).toBeGreaterThan(0);
    });
  });

  it("shows admin nav links for admin users", async () => {
    render(<App />);
    await waitFor(() => {
      expect(screen.getByText("Server")).toBeInTheDocument();
      expect(screen.getByText("Settings")).toBeInTheDocument();
      expect(screen.getByText("Users")).toBeInTheDocument();
      expect(screen.getByText("Audit Logs")).toBeInTheDocument();
    });
  });
});

import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { AuditPage } from "./AuditPage";

describe("AuditPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("renders audit log heading", async () => {
    cleanup = mockFetch({
      "/audit-logs": { data: [], total: 0, page: 1, per_page: 50 },
    });
    renderWithProviders(<AuditPage />);
    await waitFor(() => {
      expect(screen.getByText("Audit Logs")).toBeInTheDocument();
    });
  });

  it("shows empty state when no logs", async () => {
    cleanup = mockFetch({
      "/audit-logs": { data: [], total: 0, page: 1, per_page: 50 },
    });
    renderWithProviders(<AuditPage />);
    await waitFor(() => {
      expect(screen.getByText("No audit logs found")).toBeInTheDocument();
    });
  });

  it("renders audit log entries", async () => {
    cleanup = mockFetch({
      "/audit-logs": {
        data: [
          {
            id: 1,
            timestamp: "2026-01-01T12:00:00Z",
            actor: "admin",
            action: "create",
            resource_type: "client",
            resource_id: "c1",
            details: "",
            ip_address: "127.0.0.1",
          },
        ],
        total: 1,
        page: 1,
        per_page: 50,
      },
    });

    renderWithProviders(<AuditPage />);
    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
      expect(screen.getByText("create")).toBeInTheDocument();
      expect(screen.getByText("client")).toBeInTheDocument();
      expect(screen.getByText("127.0.0.1")).toBeInTheDocument();
    });
  });

  it("shows export button", async () => {
    cleanup = mockFetch({
      "/audit-logs": { data: [], total: 0, page: 1, per_page: 50 },
    });
    renderWithProviders(<AuditPage />);
    await waitFor(() => {
      expect(screen.getByText("Export to Excel")).toBeInTheDocument();
    });
  });

  it("shows pagination info", async () => {
    cleanup = mockFetch({
      "/audit-logs": { data: [], total: 0, page: 1, per_page: 50 },
    });
    renderWithProviders(<AuditPage />);
    await waitFor(() => {
      expect(screen.getByText(/Page 1 of 1/)).toBeInTheDocument();
    });
  });
});

import { describe, it, expect, afterEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { AuditPage } from "./AuditPage";

const mockResponses = {
  "/audit-logs/filters": { actors: ["admin", "user1"], actions: ["client.create", "user.delete"] },
  "/audit-logs": { data: [], total: 0, page: 1, per_page: 50 },
};

const logEntry = {
  id: 1,
  timestamp: "2026-01-15T10:30:00Z",
  actor: "admin",
  action: "client.create",
  resource_type: "client",
  resource_id: "abc123",
  details: '{"name":"Alice","email":"alice@example.com"}',
  ip_address: "192.168.1.100",
};

describe("AuditPage interactions", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows export button and clicks it", async () => {
    const user = userEvent.setup();
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("Export to Excel")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Export to Excel"));
    expect(openSpy).toHaveBeenCalledWith(
      expect.stringContaining("/audit-logs/export"),
      "_blank"
    );
    openSpy.mockRestore();
  });

  it("types in search and presses enter", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Name, email, or ID...")).toBeInTheDocument();
    });

    await user.type(screen.getByPlaceholderText("Name, email, or ID..."), "alice{Enter}");
  });

  it("clicks search button", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Name, email, or ID...")).toBeInTheDocument();
    });

    const searchBtns = screen.getAllByLabelText("Search");
    await user.click(searchBtns[0]);
  });

  it("shows date from/to inputs", async () => {
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("Date From")).toBeInTheDocument();
      expect(screen.getByText("Date To")).toBeInTheDocument();
    });
  });

  it("shows actor and action filter labels", async () => {
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      // Use getAllByText since table header "Action" and filter label "Action" may both be present
      expect(screen.getAllByText("Actor").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("Action").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("renders logs with all columns", async () => {
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [logEntry],
        total: 1,
        page: 1,
        per_page: 50,
      },
    });

    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
      expect(screen.getByText("client.create")).toBeInTheDocument();
      expect(screen.getByText("client")).toBeInTheDocument();
      expect(screen.getByText("192.168.1.100")).toBeInTheDocument();
    });
  });

  it("shows resource with name only when no email", async () => {
    const logNoEmail = {
      ...logEntry,
      id: 2,
      details: '{"name":"Bob"}',
    };
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [logNoEmail],
        total: 1,
        page: 1,
        per_page: 50,
      },
    });

    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText(/Bob.*abc123/)).toBeInTheDocument();
    });
  });

  it("shows resource_id when no details", async () => {
    const logNoDetails = {
      ...logEntry,
      id: 3,
      details: "{}",
    };
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [logNoDetails],
        total: 1,
        page: 1,
        per_page: 50,
      },
    });

    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("abc123")).toBeInTheDocument();
    });
  });

  it("shows pagination controls", async () => {
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [],
        total: 100,
        page: 1,
        per_page: 50,
      },
    });

    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText(/Page 1 of 2/)).toBeInTheDocument();
      expect(screen.getByText("Previous")).toBeInTheDocument();
      expect(screen.getByText("Next")).toBeInTheDocument();
    });
  });

  it("disables previous on page 1", async () => {
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      const prevButton = screen.getByText("Previous").closest("button");
      expect(prevButton).toBeDisabled();
    });
  });

  it("shows table headers", async () => {
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("Timestamp")).toBeInTheDocument();
      expect(screen.getByText("IP Address")).toBeInTheDocument();
      expect(screen.getByText("Resource Type")).toBeInTheDocument();
      expect(screen.getByText("Resource")).toBeInTheDocument();
    });
  });

  it("changes date from filter", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("Date From")).toBeInTheDocument();
    });

    const fromInput = screen.getByLabelText("Date From");
    await user.clear(fromInput);
    await user.type(fromInput, "2026-01-01");
  });

  it("handles invalid details JSON gracefully", async () => {
    const logBadJSON = {
      ...logEntry,
      id: 4,
      details: "not-json",
    };
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [logBadJSON],
        total: 1,
        page: 1,
        per_page: 50,
      },
    });

    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      // should fallback to showing resource_id
      expect(screen.getByText("abc123")).toBeInTheDocument();
    });
  });

  it("clicks Next pagination button", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [],
        total: 100,
        page: 1,
        per_page: 50,
      },
    });
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("Next")).toBeInTheDocument();
    });

    const nextButton = screen.getByText("Next").closest("button")!;
    expect(nextButton).not.toBeDisabled();
    await user.click(nextButton);
  });

  it("disables Next on last page", async () => {
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [],
        total: 10,
        page: 1,
        per_page: 50,
      },
    });
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      const nextButton = screen.getByText("Next").closest("button");
      expect(nextButton).toBeDisabled();
    });
  });

  it("changes date to filter", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("Date To")).toBeInTheDocument();
    });

    const toInput = screen.getByLabelText("Date To");
    await user.clear(toInput);
    await user.type(toInput, "2026-12-31");
  });

  it("shows resource with email matching name", async () => {
    const logEmailOnly = {
      ...logEntry,
      id: 5,
      details: '{"email":"alice@example.com"}',
    };
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [logEmailOnly],
        total: 1,
        page: 1,
        per_page: 50,
      },
    });

    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      // name = email, so format is: name (resource_id)
      expect(screen.getByText(/alice@example.com.*abc123/)).toBeInTheDocument();
    });
  });

  it("exports with filters applied", async () => {
    const user = userEvent.setup();
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Name, email, or ID...")).toBeInTheDocument();
    });

    // Clear existing text, type fresh, then press Enter to apply filter
    const searchInput = screen.getByPlaceholderText("Name, email, or ID...");
    await user.clear(searchInput);
    await user.type(searchInput, "myfilter{Enter}");

    // Now export
    await user.click(screen.getByText("Export to Excel"));

    expect(openSpy).toHaveBeenCalledWith(
      expect.stringContaining("search=myfilter"),
      "_blank"
    );
    openSpy.mockRestore();
  });

  it("shows total count in pagination info", async () => {
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [logEntry],
        total: 42,
        page: 1,
        per_page: 50,
      },
    });
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText(/42 total/)).toBeInTheDocument();
    });
  });

  it("renders Activity Log card title", async () => {
    cleanup = mockFetch(mockResponses);
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("Activity Log")).toBeInTheDocument();
    });
  });

  it("clicks Previous button when on page 2", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      ...mockResponses,
      "/audit-logs": {
        data: [],
        total: 100,
        page: 2,
        per_page: 50,
      },
    });

    // Navigate to page=2
    window.history.pushState({}, "", "?page=2");
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByText("Previous")).toBeInTheDocument();
    });

    const prevButton = screen.getByText("Previous").closest("button")!;
    expect(prevButton).not.toBeDisabled();
    await user.click(prevButton);

    // Clean up URL
    window.history.pushState({}, "", "/");
  });

  it("clears a filter by setting it to empty value", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch(mockResponses);

    // Start with a search filter applied
    window.history.pushState({}, "", "?search=alice");
    renderWithProviders(<AuditPage />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Name, email, or ID...")).toBeInTheDocument();
    });

    // Clear the search and press Enter
    const searchInput = screen.getByPlaceholderText("Name, email, or ID...");
    await user.clear(searchInput);
    await user.type(searchInput, "{Enter}");

    // Clean up URL
    window.history.pushState({}, "", "/");
  });
});

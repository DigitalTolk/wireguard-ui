import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { UsersPage } from "./UsersPage";

describe("UsersPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows heading", async () => {
    cleanup = mockFetch({ "/users": [] });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("Users")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    cleanup = mockFetch({ "/users": [] });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("No users have logged in yet")).toBeInTheDocument();
    });
  });

  it("renders user list", async () => {
    cleanup = mockFetch({
      "/users": [
        { username: "admin", email: "admin@company.com", display_name: "Admin User", updated_at: "2026-04-22T12:00:00Z" },
        { username: "jdoe", email: "jdoe@company.com", display_name: "Jane Doe", updated_at: "2026-04-21T10:00:00Z" },
      ],
    });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("Admin User")).toBeInTheDocument();
      expect(screen.getByText("Jane Doe")).toBeInTheDocument();
      expect(screen.getByText("admin@company.com")).toBeInTheDocument();
      expect(screen.getByText("jdoe@company.com")).toBeInTheDocument();
    });
  });

  it("shows SSO explanation text", async () => {
    cleanup = mockFetch({ "/users": [] });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText(/managed through your SSO provider/)).toBeInTheDocument();
    });
  });
});

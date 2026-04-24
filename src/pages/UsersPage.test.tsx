import { describe, it, expect, afterEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { UsersPage } from "./UsersPage";

const adminMe = { username: "admin", email: "admin@company.com", display_name: "Admin User", admin: true };

describe("UsersPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows heading", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/users": [] });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("Users")).toBeInTheDocument();
    });
  });

  it("shows empty state with correct colSpan message", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/users": [] });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      const cell = screen.getByText("No users have logged in yet");
      expect(cell).toBeInTheDocument();
      // Verify colSpan is 5 on the empty state cell
      expect(cell.closest("td")).toHaveAttribute("colspan", "5");
    });
  });

  it("renders user list", async () => {
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/users": [
        { username: "admin", email: "admin@company.com", display_name: "Admin User", admin: true, updated_at: "2026-04-22T12:00:00Z" },
        { username: "jdoe", email: "jdoe@company.com", display_name: "Jane Doe", admin: false, updated_at: "2026-04-21T10:00:00Z" },
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
    cleanup = mockFetch({ "/auth/me": adminMe, "/users": [] });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText(/managed through your SSO provider/)).toBeInTheDocument();
    });
  });

  it("shows Admin badge for admin users and User badge for non-admin users", async () => {
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/users": [
        { username: "admin", email: "admin@company.com", display_name: "Admin User", admin: true, updated_at: "2026-04-22T12:00:00Z" },
        { username: "jdoe", email: "jdoe@company.com", display_name: "Jane Doe", admin: false, updated_at: "2026-04-21T10:00:00Z" },
      ],
    });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("Admin")).toBeInTheDocument();
      expect(screen.getByText("User")).toBeInTheDocument();
    });
  });

  it("hides admin toggle for the current user", async () => {
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/users": [
        { username: "admin", email: "admin@company.com", display_name: "Admin User", admin: true, updated_at: "2026-04-22T12:00:00Z" },
        { username: "jdoe", email: "jdoe@company.com", display_name: "Jane Doe", admin: false, updated_at: "2026-04-21T10:00:00Z" },
      ],
    });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Toggle admin for jdoe")).toBeInTheDocument();
    });
    // Toggle should not be rendered for the logged-in user
    expect(screen.queryByLabelText("Toggle admin for admin")).not.toBeInTheDocument();
    // Toggle for other users should work normally
    expect(screen.getByLabelText("Toggle admin for jdoe")).not.toBeChecked();
  });

  it("triggers toggleAdmin mutation when switch is clicked", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/users": [
        { username: "jdoe", email: "jdoe@company.com", display_name: "Jane Doe", admin: false, updated_at: "2026-04-21T10:00:00Z" },
      ],
      "/users/jdoe/admin": {},
    });
    renderWithProviders(<UsersPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Toggle admin for jdoe")).toBeInTheDocument();
    });

    const toggle = screen.getByLabelText("Toggle admin for jdoe");
    await user.click(toggle);

    // Verify the fetch was called with PATCH to the admin endpoint
    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/users/jdoe/admin"),
        expect.objectContaining({ method: "PATCH" })
      );
    });
  });

  it("shows toast error when toggleAdmin mutation fails", async () => {
    const user = userEvent.setup();
    // Set up fetch to return an error for the admin toggle
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/auth/me")) {
        return {
          ok: true,
          status: 200,
          json: async () => adminMe,
          text: async () => JSON.stringify(adminMe),
          headers: new Headers(),
        } as Response;
      }
      if (url.includes("/users") && !url.includes("/admin")) {
        return {
          ok: true,
          status: 200,
          json: async () => [
            { username: "jdoe", email: "jdoe@company.com", display_name: "Jane Doe", admin: false, updated_at: "2026-04-21T10:00:00Z" },
          ],
          text: async () => "[]",
          headers: new Headers(),
        } as Response;
      }
      if (url.includes("/admin")) {
        return {
          ok: false,
          status: 500,
          json: async () => ({ error: { code: "INTERNAL", message: "Toggle failed" } }),
          text: async () => "error",
          headers: new Headers(),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    });
    cleanup = () => { globalThis.fetch = originalFetch; };

    renderWithProviders(<UsersPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Toggle admin for jdoe")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Toggle admin for jdoe"));

    // Verify the PATCH call was attempted
    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/users/jdoe/admin"),
        expect.objectContaining({ method: "PATCH" })
      );
    });
  });

  it("shows dash for missing display_name and email", async () => {
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/users": [
        { username: "noinfo", email: "", display_name: "", admin: false, updated_at: "" },
      ],
    });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("noinfo")).toBeInTheDocument();
    });
    // display_name and email should show "-"
    const dashes = screen.getAllByText("-");
    expect(dashes.length).toBeGreaterThanOrEqual(2);
  });

  it("shows dash when updated_at is missing", async () => {
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/users": [
        { username: "newuser", email: "new@co.com", display_name: "New User", admin: false, updated_at: "" },
      ],
    });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("New User")).toBeInTheDocument();
    });
    // updated_at empty means "-" is shown in the Last Login column
    const dashes = screen.getAllByText("-");
    expect(dashes.length).toBeGreaterThanOrEqual(1);
  });

  it("shows table headers including Role column", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/users": [] });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("Username")).toBeInTheDocument();
      expect(screen.getByText("Display Name")).toBeInTheDocument();
      expect(screen.getByText("Email")).toBeInTheDocument();
      expect(screen.getByText("Role")).toBeInTheDocument();
      expect(screen.getByText("Last Login")).toBeInTheDocument();
    });
  });
});

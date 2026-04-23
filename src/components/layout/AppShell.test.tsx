import { describe, it, expect, afterEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { AppShell } from "./AppShell";

const adminMe = {
  username: "admin",
  email: "admin@test.com",
  display_name: "Admin User",
  admin: true,
};

const regularMe = {
  username: "user1",
  email: "user1@test.com",
  display_name: "Regular User",
  admin: false,
};

describe("AppShell", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("renders the WireGuard UI title", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getAllByText("WireGuard UI").length).toBeGreaterThan(0);
    });
  });

  it("shows admin nav items for admin users", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByText("Server")).toBeInTheDocument();
      expect(screen.getByText("Settings")).toBeInTheDocument();
      expect(screen.getByText("Users")).toBeInTheDocument();
      expect(screen.getByText("Audit Logs")).toBeInTheDocument();
      expect(screen.getByText("Wake-on-LAN")).toBeInTheDocument();
      expect(screen.getByText("Status")).toBeInTheDocument();
    });
  });

  it("hides admin nav items for non-admin users", async () => {
    cleanup = mockFetch({ "/auth/me": regularMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByText("Clients")).toBeInTheDocument();
      expect(screen.getByText("About")).toBeInTheDocument();
    });
    expect(screen.queryByText("Server")).not.toBeInTheDocument();
    expect(screen.queryByText("Settings")).not.toBeInTheDocument();
    expect(screen.queryByText("Users")).not.toBeInTheDocument();
    expect(screen.queryByText("Audit Logs")).not.toBeInTheDocument();
    expect(screen.queryByText("Wake-on-LAN")).not.toBeInTheDocument();
    expect(screen.queryByText("Status")).not.toBeInTheDocument();
  });

  it("shows common nav items for all users", async () => {
    cleanup = mockFetch({ "/auth/me": regularMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByText("Clients")).toBeInTheDocument();
      expect(screen.getByText("About")).toBeInTheDocument();
    });
  });

  it("displays admin user display_name and email", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByText("Admin User")).toBeInTheDocument();
      expect(screen.getByText("admin@test.com")).toBeInTheDocument();
    });
  });

  it("displays non-admin user email without Admin badge", async () => {
    cleanup = mockFetch({ "/auth/me": regularMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByText("Regular User")).toBeInTheDocument();
      expect(screen.getByText("user1@test.com")).toBeInTheDocument();
    });
    expect(screen.queryByText("Admin")).not.toBeInTheDocument();
  });

  it("shows Admin badge for admin users", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByText("Admin")).toBeInTheDocument();
    });
  });

  it("falls back to username when display_name is empty", async () => {
    const noDisplayName = { ...regularMe, display_name: "" };
    cleanup = mockFetch({ "/auth/me": noDisplayName });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByText("user1")).toBeInTheDocument();
    });
  });

  it("shows mobile menu toggle button", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByLabelText("Open menu")).toBeInTheDocument();
    });
  });

  it("toggles mobile sidebar open and closed", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByLabelText("Open menu")).toBeInTheDocument();
    });

    // Open menu
    await user.click(screen.getByLabelText("Open menu"));
    expect(screen.getByLabelText("Close menu")).toBeInTheDocument();

    // Close menu
    await user.click(screen.getByLabelText("Close menu"));
    expect(screen.getByLabelText("Open menu")).toBeInTheDocument();
  });

  it("shows logout button", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByLabelText("Log out")).toBeInTheDocument();
    });
  });

  it("calls logout API on logout button click", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/auth/me": adminMe, "/auth/logout": {} });

    // Mock location.href setter
    const hrefSetter = vi.fn();
    Object.defineProperty(window, "location", {
      value: { ...window.location, href: "" },
      writable: true,
    });
    Object.defineProperty(window.location, "href", {
      set: hrefSetter,
      get: () => "",
    });

    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByLabelText("Log out")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Log out"));

    await waitFor(() => {
      expect(hrefSetter).toHaveBeenCalledWith("./api/v1/auth/oidc/login");
    });
  });

  it("has main navigation landmark", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByRole("navigation", { name: "Main navigation" })).toBeInTheDocument();
    });
  });

  it("has main content landmark", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByRole("main")).toBeInTheDocument();
    });
  });

  it("clicking a nav link closes the mobile sidebar", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByLabelText("Open menu")).toBeInTheDocument();
    });

    // Open sidebar
    await user.click(screen.getByLabelText("Open menu"));
    expect(screen.getByLabelText("Close menu")).toBeInTheDocument();

    // Click a nav link (About is always visible)
    await user.click(screen.getByText("About"));

    // Sidebar should close
    await waitFor(() => {
      expect(screen.getByLabelText("Open menu")).toBeInTheDocument();
    });
  });

  it("clicking the overlay closes the mobile sidebar", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/auth/me": adminMe });
    renderWithProviders(<AppShell />);
    await waitFor(() => {
      expect(screen.getByLabelText("Open menu")).toBeInTheDocument();
    });

    // Open sidebar
    await user.click(screen.getByLabelText("Open menu"));
    expect(screen.getByLabelText("Close menu")).toBeInTheDocument();

    // Click the overlay (it has aria-hidden="true")
    const overlay = document.querySelector(".fixed.inset-0.z-40");
    expect(overlay).not.toBeNull();
    await user.click(overlay!);

    // Sidebar should close
    await waitFor(() => {
      expect(screen.getByLabelText("Open menu")).toBeInTheDocument();
    });
  });
});

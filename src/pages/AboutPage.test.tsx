import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { AboutPage } from "./AboutPage";

describe("AboutPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("renders the about heading", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "v1.0.0",
        git_commit: "abc123",
        client_defaults: {},
      },
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      expect(screen.getByText("About")).toBeInTheDocument();
    });
  });

  it("shows version and commit info", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "v1.0.0",
        git_commit: "abc123",
        client_defaults: {},
      },
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("v1.0.0")).toBeInTheDocument();
      expect(screen.getByDisplayValue("abc123")).toBeInTheDocument();
    });
  });

  it("shows fork attribution", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "dev",
        git_commit: "N/A",
        client_defaults: {},
      },
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      expect(screen.getByText("Khanh Ngo")).toBeInTheDocument();
      expect(screen.getByText(/Fork of/)).toBeInTheDocument();
      expect(screen.getByText("DigitalTolk/wireguard-ui")).toBeInTheDocument();
    });
  });

  it("shows latest release info when available", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "v1.0.0",
        git_commit: "abc123",
        client_defaults: {},
      },
      "api.github.com/repos/DigitalTolk/wireguard-ui/releases/latest": {
        tag_name: "v1.1.0",
        published_at: "2026-04-20T00:00:00Z",
      },
      "api.github.com/repos/DigitalTolk/wireguard-ui/contributors": [
        { login: "user1", avatar_url: "https://example.com/avatar1.png", html_url: "https://github.com/user1", contributions: 10 },
      ],
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("v1.1.0")).toBeInTheDocument();
    });
    // Should also show the "Update available" badge since v1.0.0 !== v1.1.0
    expect(screen.getByText("Update available")).toBeInTheDocument();
  });

  it("shows contributor avatars when loaded", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "v1.0.0",
        git_commit: "abc123",
        client_defaults: {},
      },
      "api.github.com/repos/DigitalTolk/wireguard-ui/contributors": [
        { login: "user1", avatar_url: "https://example.com/avatar1.png", html_url: "https://github.com/user1", contributions: 10 },
        { login: "user2", avatar_url: "https://example.com/avatar2.png", html_url: "https://github.com/user2", contributions: 5 },
      ],
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      const img1 = screen.getByAltText("user1");
      expect(img1).toBeInTheDocument();
      expect(img1).toHaveAttribute("src", "https://example.com/avatar1.png");
      const img2 = screen.getByAltText("user2");
      expect(img2).toBeInTheDocument();
    });
  });

  it("does not show Update available badge when version matches release", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "v1.0.0",
        git_commit: "abc123",
        client_defaults: {},
      },
      "api.github.com/repos/DigitalTolk/wireguard-ui/releases/latest": {
        tag_name: "v1.0.0",
        published_at: "2026-04-20T00:00:00Z",
      },
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      // Both Current Version and Latest Release show v1.0.0
      const inputs = screen.getAllByDisplayValue("v1.0.0");
      expect(inputs.length).toBeGreaterThanOrEqual(2);
    });
    expect(screen.queryByText("Update available")).not.toBeInTheDocument();
  });

  it("does not show Update available badge when version is development", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "development",
        git_commit: "abc123",
        client_defaults: {},
      },
      "api.github.com/repos/DigitalTolk/wireguard-ui/releases/latest": {
        tag_name: "v1.1.0",
        published_at: "2026-04-20T00:00:00Z",
      },
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("development")).toBeInTheDocument();
    });
    expect(screen.queryByText("Update available")).not.toBeInTheDocument();
  });

  it("shows N/A for published_at when not available in release", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "v1.0.0",
        git_commit: "abc123",
        client_defaults: {},
      },
      "api.github.com/repos/DigitalTolk/wireguard-ui/releases/latest": {
        tag_name: "v1.1.0",
        published_at: null,
      },
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("v1.1.0")).toBeInTheDocument();
      expect(screen.getByDisplayValue("N/A")).toBeInTheDocument();
    });
  });

  it("shows skeleton when release data is not yet loaded and no contributors", async () => {
    // When fetch for GitHub returns non-ok, latestRelease should be null
    // and contributors should be empty, showing skeleton
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "v1.0.0",
        git_commit: "abc123",
        client_defaults: {},
      },
      // No GitHub responses - they'll 404, which triggers null/[] return
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("v1.0.0")).toBeInTheDocument();
    });
    // Should show "Latest Release" label with skeleton, not input
    expect(screen.getByText("Latest Release")).toBeInTheDocument();
  });

  it("shows copyright with project link", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "v1.0.0",
        git_commit: "abc123",
        client_defaults: {},
      },
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      expect(screen.getByText(/All rights reserved/)).toBeInTheDocument();
    });
  });

  it("handles version match with v prefix on tag but not on app_version", async () => {
    cleanup = mockFetch({
      "/auth/info": {
        base_path: "",
        app_version: "1.0.0",
        git_commit: "abc123",
        client_defaults: {},
      },
      "api.github.com/repos/DigitalTolk/wireguard-ui/releases/latest": {
        tag_name: "v1.0.0",
        published_at: "2026-04-20T00:00:00Z",
      },
    });
    renderWithProviders(<AboutPage />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("1.0.0")).toBeInTheDocument();
    });
    // tag_name "v1.0.0" should match version "1.0.0" via the `v${version}` check
    expect(screen.queryByText("Update available")).not.toBeInTheDocument();
  });
});

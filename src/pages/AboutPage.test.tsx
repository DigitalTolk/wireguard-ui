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
});

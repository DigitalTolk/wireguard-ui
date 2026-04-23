import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { ClientsPage } from "./ClientsPage";

describe("ClientsPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows client list heading", async () => {
    cleanup = mockFetch({ "/clients": [], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);
    await waitFor(() => {
      expect(screen.getByText("WireGuard Clients")).toBeInTheDocument();
    });
  });

  it("shows empty state when no clients", async () => {
    cleanup = mockFetch({ "/clients": [], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText(/No clients configured yet/)).toBeInTheDocument();
    });
  });

  it("renders clients from API", async () => {
    cleanup = mockFetch({
      "/clients": [
        {
          Client: {
            id: "c1",
            name: "Test Client",
            email: "test@example.com",
            enabled: true,
            allocated_ips: ["10.0.0.2/32"],
            allowed_ips: ["0.0.0.0/0"],
            additional_notes: "",
            created_at: "2024-01-01T00:00:00Z",
            updated_at: "2024-01-01T00:00:00Z",
          },
          QRCode: "",
        },
      ],
      "/subnet-ranges": [],
    });

    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Test Client")).toBeInTheDocument();
    });
  });

  it("shows enabled badge for enabled clients", async () => {
    cleanup = mockFetch({
      "/clients": [
        {
          Client: {
            id: "c1",
            name: "Active",
            email: "",
            enabled: true,
            allocated_ips: [],
            allowed_ips: [],
            additional_notes: "",
            created_at: "2024-01-01T00:00:00Z",
            updated_at: "2024-01-01T00:00:00Z",
          },
          QRCode: "",
        },
      ],
      "/subnet-ranges": [],
    });

    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Enabled")).toBeInTheDocument();
    });
  });
});

import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { StatusPage } from "./StatusPage";

describe("StatusPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows heading", async () => {
    cleanup = mockFetch({ "/status": [] });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Server Status")).toBeInTheDocument();
    });
  });

  it("shows no interfaces message when empty", async () => {
    cleanup = mockFetch({ "/status": [] });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("No WireGuard interfaces found")).toBeInTheDocument();
    });
  });

  it("renders device with peers", async () => {
    cleanup = mockFetch({
      "/status": [
        {
          name: "wg0",
          peers: [
            {
              name: "Client1",
              email: "c1@test.com",
              public_key: "pk1abcdef1234567890",
              received_bytes: 1024,
              transmit_bytes: 2048,
              last_handshake_time: new Date().toISOString(),
              last_handshake_rel: 60000000000,
              connected: true,
              allocated_ip: "10.0.0.2/32",
              endpoint: "1.2.3.4:51820",
            },
          ],
        },
      ],
    });

    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("wg0")).toBeInTheDocument();
      expect(screen.getByText("Client1")).toBeInTheDocument();
      expect(screen.getByText("1.2.3.4:51820")).toBeInTheDocument();
    });
  });

  it("shows sortable column headers", async () => {
    cleanup = mockFetch({ "/status": [{ name: "wg0", peers: [] }] });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Name")).toBeInTheDocument();
      expect(screen.getByText("Handshake")).toBeInTheDocument();
      expect(screen.getByText("Endpoint")).toBeInTheDocument();
    });
  });
});

import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { SettingsPage } from "./SettingsPage";

describe("SettingsPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows heading", async () => {
    cleanup = mockFetch({
      "/settings": {
        endpoint_address: "vpn.example.com",
        dns_servers: ["1.1.1.1"],
        mtu: 1450,
        persistent_keepalive: 15,
        firewall_mark: "0xca6c",
        table: "auto",
        config_file_path: "/etc/wireguard/wg0.conf",
      },
    });
    renderWithProviders(<SettingsPage />);
    await waitFor(() => {
      expect(screen.getByText("Global Settings")).toBeInTheDocument();
    });
  });

  it("renders settings values", async () => {
    cleanup = mockFetch({
      "/settings": {
        endpoint_address: "vpn.example.com",
        dns_servers: ["1.1.1.1", "8.8.8.8"],
        mtu: 1450,
        persistent_keepalive: 15,
        firewall_mark: "0xca6c",
        table: "auto",
        config_file_path: "/etc/wireguard/wg0.conf",
      },
    });

    renderWithProviders(<SettingsPage />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("vpn.example.com")).toBeInTheDocument();
      expect(screen.getByDisplayValue("1.1.1.1, 8.8.8.8")).toBeInTheDocument();
      expect(screen.getByDisplayValue("1450")).toBeInTheDocument();
    });
  });
});

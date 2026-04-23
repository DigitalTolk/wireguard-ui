import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { SettingsPage } from "./SettingsPage";

const defaultSettings = {
  endpoint_address: "vpn.example.com",
  dns_servers: ["1.1.1.1"],
  mtu: 1450,
  persistent_keepalive: 15,
  firewall_mark: "0xca6c",
  table: "auto",
  config_file_path: "/etc/wireguard/wg0.conf",
};

describe("SettingsPage interactions", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("renders all fields with values", async () => {
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("vpn.example.com")).toBeInTheDocument();
      expect(screen.getByDisplayValue("1.1.1.1")).toBeInTheDocument();
      expect(screen.getByDisplayValue("1450")).toBeInTheDocument();
      expect(screen.getByDisplayValue("15")).toBeInTheDocument();
      expect(screen.getByDisplayValue("0xca6c")).toBeInTheDocument();
      expect(screen.getByDisplayValue("auto")).toBeInTheDocument();
      expect(screen.getByDisplayValue("/etc/wireguard/wg0.conf")).toBeInTheDocument();
    });
  });

  it("shows save button", async () => {
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByText("Save Settings")).toBeInTheDocument();
    });
  });

  it("modifies endpoint and clicks save", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("vpn.example.com")).toBeInTheDocument();
    });

    const endpointInput = screen.getByDisplayValue("vpn.example.com");
    await user.clear(endpointInput);
    await user.type(endpointInput, "newvpn.example.com");

    await user.click(screen.getByText("Save Settings"));
  });

  it("shows help text for fields", async () => {
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByText(/Public hostname or IP address/)).toBeInTheDocument();
      expect(screen.getByText(/Comma-separated list of DNS/)).toBeInTheDocument();
      expect(screen.getByText(/Maximum Transmission Unit/)).toBeInTheDocument();
    });
  });

  it("shows card headings", async () => {
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByText("Network")).toBeInTheDocument();
      expect(screen.getByText("Advanced")).toBeInTheDocument();
    });
  });

  it("modifies DNS field", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("1.1.1.1")).toBeInTheDocument();
    });

    const dnsInput = screen.getByDisplayValue("1.1.1.1");
    await user.clear(dnsInput);
    await user.type(dnsInput, "8.8.8.8, 8.8.4.4");
  });

  it("modifies MTU field", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("1450")).toBeInTheDocument();
    });

    const mtuInput = screen.getByDisplayValue("1450");
    await user.clear(mtuInput);
    await user.type(mtuInput, "1420");
  });

  it("modifies config file path", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("/etc/wireguard/wg0.conf")).toBeInTheDocument();
    });

    const configInput = screen.getByDisplayValue("/etc/wireguard/wg0.conf");
    await user.clear(configInput);
    await user.type(configInput, "/etc/wireguard/wg1.conf");
  });
});

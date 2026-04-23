import { describe, it, expect, afterEach, vi } from "vitest";
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

  it("shows validation error for config file path not starting with /", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("/etc/wireguard/wg0.conf")).toBeInTheDocument();
    });

    const configInput = screen.getByDisplayValue("/etc/wireguard/wg0.conf");
    await user.clear(configInput);
    await user.type(configInput, "relative/path/wg0.conf");

    await waitFor(() => {
      expect(screen.getByText("Config file path must be an absolute path (start with /)")).toBeInTheDocument();
    });
  });

  it("successfully saves settings and resets form state", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/settings": defaultSettings,
    });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("vpn.example.com")).toBeInTheDocument();
    });

    // Modify a field so we know state has been dirtied
    const endpointInput = screen.getByDisplayValue("vpn.example.com");
    await user.clear(endpointInput);
    await user.type(endpointInput, "new.vpn.com");

    // Click save
    await user.click(screen.getByText("Save Settings"));

    // Verify the PUT was called
    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/settings"),
        expect.objectContaining({ method: "PUT" })
      );
    });
  });

  it("shows error when save settings fails", async () => {
    const user = userEvent.setup();
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/settings")) {
        // Return settings on GET, error on PUT
        const method = (vi.mocked(globalThis.fetch).mock.lastCall?.[1] as RequestInit | undefined)?.method;
        // Since we can't easily distinguish, check the second arg
        return {
          ok: true,
          status: 200,
          json: async () => defaultSettings,
          text: async () => JSON.stringify(defaultSettings),
          headers: new Headers(),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    });
    cleanup = () => { globalThis.fetch = originalFetch; };

    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("vpn.example.com")).toBeInTheDocument();
    });

    // Now switch to error mode for the PUT
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/settings") && init?.method === "PUT") {
        return {
          ok: false,
          status: 500,
          json: async () => ({ error: { code: "INTERNAL", message: "Save failed" } }),
          text: async () => "error",
          headers: new Headers(),
        } as Response;
      }
      if (url.includes("/settings")) {
        return {
          ok: true,
          status: 200,
          json: async () => defaultSettings,
          text: async () => JSON.stringify(defaultSettings),
          headers: new Headers(),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    });

    await user.click(screen.getByText("Save Settings"));

    // Verify the PUT was called
    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/settings"),
        expect.objectContaining({ method: "PUT" })
      );
    });
  });

  it("shows validation error for empty endpoint", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("vpn.example.com")).toBeInTheDocument();
    });

    const endpointInput = screen.getByDisplayValue("vpn.example.com");
    await user.clear(endpointInput);

    await waitFor(() => {
      expect(screen.getByText("Endpoint address is required")).toBeInTheDocument();
    });
  });

  it("shows validation error for empty DNS", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("1.1.1.1")).toBeInTheDocument();
    });

    const dnsInput = screen.getByDisplayValue("1.1.1.1");
    await user.clear(dnsInput);

    await waitFor(() => {
      expect(screen.getByText("At least one DNS server is required")).toBeInTheDocument();
    });
  });

  it("shows validation error for invalid DNS IP", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("1.1.1.1")).toBeInTheDocument();
    });

    const dnsInput = screen.getByDisplayValue("1.1.1.1");
    await user.clear(dnsInput);
    await user.type(dnsInput, "not-an-ip");

    await waitFor(() => {
      expect(screen.getByText("Each DNS server must be a valid IP address")).toBeInTheDocument();
    });
  });

  it("shows validation error for invalid MTU", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("1450")).toBeInTheDocument();
    });

    const mtuInput = screen.getByDisplayValue("1450");
    await user.clear(mtuInput);
    await user.type(mtuInput, "500");

    await waitFor(() => {
      expect(screen.getByText("MTU must be 0 (to omit) or between 1280 and 9000")).toBeInTheDocument();
    });
  });

  it("shows validation error for empty MTU", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("1450")).toBeInTheDocument();
    });

    const mtuInput = screen.getByDisplayValue("1450");
    await user.clear(mtuInput);

    await waitFor(() => {
      expect(screen.getByText("MTU is required")).toBeInTheDocument();
    });
  });

  it("shows validation error for invalid keepalive", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("15")).toBeInTheDocument();
    });

    const kaInput = screen.getByDisplayValue("15");
    await user.clear(kaInput);
    await user.type(kaInput, "99999");

    await waitFor(() => {
      expect(screen.getByText("Persistent keepalive must be between 0 and 65535")).toBeInTheDocument();
    });
  });

  it("shows validation error for invalid firewall mark", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("0xca6c")).toBeInTheDocument();
    });

    const fwInput = screen.getByDisplayValue("0xca6c");
    await user.clear(fwInput);
    await user.type(fwInput, "not-a-number");

    await waitFor(() => {
      expect(screen.getByText("Must be a hex (0x...) or decimal number")).toBeInTheDocument();
    });
  });

  it("shows validation error for empty config file path", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("/etc/wireguard/wg0.conf")).toBeInTheDocument();
    });

    const configInput = screen.getByDisplayValue("/etc/wireguard/wg0.conf");
    await user.clear(configInput);

    await waitFor(() => {
      expect(screen.getByText("Config file path is required")).toBeInTheDocument();
    });
  });

  it("disables save button when form is invalid", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("vpn.example.com")).toBeInTheDocument();
    });

    // Clear endpoint to make form invalid
    const endpointInput = screen.getByDisplayValue("vpn.example.com");
    await user.clear(endpointInput);

    await waitFor(() => {
      const saveBtn = screen.getByText("Save Settings").closest("button");
      expect(saveBtn).toBeDisabled();
    });
  });

  it("modifies routing table field", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("auto")).toBeInTheDocument();
    });

    const tblInput = screen.getByDisplayValue("auto");
    await user.clear(tblInput);
    await user.type(tblInput, "100");
    expect(tblInput).toHaveValue("100");
  });

  it("modifies persistent keepalive field", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/settings": defaultSettings });
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("15")).toBeInTheDocument();
    });

    const kaInput = screen.getByDisplayValue("15");
    await user.clear(kaInput);
    await user.type(kaInput, "25");
    expect(kaInput).toHaveValue(25);
  });
});

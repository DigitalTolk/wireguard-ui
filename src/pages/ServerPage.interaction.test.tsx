import { describe, it, expect, vi, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { ServerPage } from "./ServerPage";

const serverData = {
  Interface: { addresses: ["10.0.0.1/24"], listen_port: 51820 },
  KeyPair: { public_key: "serverpub123", private_key: "serverpriv" },
};

describe("ServerPage interactions", () => {
  let cleanup: () => void;
  afterEach(() => { cleanup?.(); });

  it("clicks regenerate keypair with confirmation", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    cleanup = mockFetch({
      "/server": serverData,
      "/server/keypair": { public_key: "newpub", private_key: "newpriv" },
    });

    renderWithProviders(<ServerPage />);
    await waitFor(() => {
      expect(screen.getByText("Regenerate")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Regenerate"));
    expect(window.confirm).toHaveBeenCalled();
  });

  it("clicks save interface", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/server": serverData,
      "/server/interface": serverData.Interface,
    });

    renderWithProviders(<ServerPage />);
    await waitFor(() => {
      expect(screen.getByText("Save")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Save"));
  });

  it("displays server addresses", async () => {
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("10.0.0.1/24")).toBeInTheDocument();
      expect(screen.getByDisplayValue("51820")).toBeInTheDocument();
    });
  });

  it("cancels regenerate keypair when user declines confirm", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(false);
    cleanup = mockFetch({ "/server": serverData });

    renderWithProviders(<ServerPage />);
    await waitFor(() => {
      expect(screen.getByText("Regenerate")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Regenerate"));
    expect(window.confirm).toHaveBeenCalled();
  });

  it("shows validation error for empty addresses", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("10.0.0.1/24")).toBeInTheDocument();
    });

    const addrInput = screen.getByLabelText("Server addresses");
    await user.clear(addrInput);

    await waitFor(() => {
      expect(screen.getByText("At least one address is required")).toBeInTheDocument();
    });
  });

  it("shows validation error for invalid CIDR addresses", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("10.0.0.1/24")).toBeInTheDocument();
    });

    const addrInput = screen.getByLabelText("Server addresses");
    await user.clear(addrInput);
    await user.type(addrInput, "not-a-cidr");

    await waitFor(() => {
      expect(screen.getByText("Each address must be valid CIDR (e.g. 10.252.1.0/24)")).toBeInTheDocument();
    });
  });

  it("shows validation error for empty listen port", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("51820")).toBeInTheDocument();
    });

    const portInput = screen.getByLabelText("Listen port");
    await user.clear(portInput);

    await waitFor(() => {
      expect(screen.getByText("Listen port is required")).toBeInTheDocument();
    });
  });

  it("shows validation error for out-of-range port", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("51820")).toBeInTheDocument();
    });

    const portInput = screen.getByLabelText("Listen port");
    await user.clear(portInput);
    await user.type(portInput, "99999");

    await waitFor(() => {
      expect(screen.getByText("Port must be between 1 and 65535")).toBeInTheDocument();
    });
  });

  it("disables Save button when form is invalid", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("10.0.0.1/24")).toBeInTheDocument();
    });

    // Clear addresses to make form invalid
    const addrInput = screen.getByLabelText("Server addresses");
    await user.clear(addrInput);

    await waitFor(() => {
      const saveButton = screen.getByText("Save").closest("button");
      expect(saveButton).toBeDisabled();
    });
  });

  it("shows Keypair card with public key", async () => {
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByText("Keypair")).toBeInTheDocument();
      expect(screen.getByLabelText("Server public key")).toBeInTheDocument();
      expect(screen.getByDisplayValue("serverpub123")).toBeInTheDocument();
    });
  });

  it("public key input is read-only", async () => {
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      const pubKeyInput = screen.getByLabelText("Server public key");
      expect(pubKeyInput).toHaveAttribute("readonly");
    });
  });

  it("shows Post-Up, Pre-Down, Post-Down script fields", async () => {
    const serverWithScripts = {
      Interface: {
        addresses: ["10.0.0.1/24"],
        listen_port: 51820,
        post_up: "iptables -A FORWARD",
        pre_down: "echo predown",
        post_down: "iptables -D FORWARD",
      },
      KeyPair: { public_key: "pub123", private_key: "priv" },
    };
    cleanup = mockFetch({ "/server": serverWithScripts });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByText("Post-Up Script")).toBeInTheDocument();
      expect(screen.getByText("Pre-Down Script")).toBeInTheDocument();
      expect(screen.getByText("Post-Down Script")).toBeInTheDocument();
      expect(screen.getByDisplayValue("iptables -A FORWARD")).toBeInTheDocument();
      expect(screen.getByDisplayValue("echo predown")).toBeInTheDocument();
      expect(screen.getByDisplayValue("iptables -D FORWARD")).toBeInTheDocument();
    });
  });

  it("edits Post-Up Script field", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByText("Post-Up Script")).toBeInTheDocument();
    });

    const postUpInput = screen.getByPlaceholderText("iptables -A FORWARD ...");
    await user.type(postUpInput, "echo hello");

    expect(postUpInput).toHaveValue("echo hello");
  });

  it("edits Pre-Down Script field", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByText("Pre-Down Script")).toBeInTheDocument();
    });

    const preDownInput = screen.getByPlaceholderText("Optional pre-down script");
    await user.type(preDownInput, "echo predown");

    expect(preDownInput).toHaveValue("echo predown");
  });

  it("edits Post-Down Script field", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByText("Post-Down Script")).toBeInTheDocument();
    });

    const postDownInput = screen.getByPlaceholderText("iptables -D FORWARD ...");
    await user.type(postDownInput, "echo postdown");

    expect(postDownInput).toHaveValue("echo postdown");
  });

  it("handles save interface error", async () => {
    const user = userEvent.setup();
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/server/interface") && init?.method === "PUT") {
        return {
          ok: false,
          status: 500,
          json: async () => ({ error: { code: "INTERNAL", message: "Save failed" } }),
          text: async () => "error",
          headers: new Headers(),
        } as Response;
      }
      if (url.includes("/server")) {
        return {
          ok: true,
          status: 200,
          json: async () => serverData,
          text: async () => JSON.stringify(serverData),
          headers: new Headers(),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    });
    cleanup = () => { globalThis.fetch = originalFetch; };

    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByText("Save")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/server/interface"),
        expect.objectContaining({ method: "PUT" })
      );
    });
  });

  it("handles regenerate keypair error", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/server/keypair") && init?.method === "POST") {
        return {
          ok: false,
          status: 500,
          json: async () => ({ error: { code: "INTERNAL", message: "Keypair regen failed" } }),
          text: async () => "error",
          headers: new Headers(),
        } as Response;
      }
      if (url.includes("/server")) {
        return {
          ok: true,
          status: 200,
          json: async () => serverData,
          text: async () => JSON.stringify(serverData),
          headers: new Headers(),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    });
    cleanup = () => { globalThis.fetch = originalFetch; };

    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByText("Regenerate")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Regenerate"));

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/server/keypair"),
        expect.objectContaining({ method: "POST" })
      );
    });
  });

  it("edits listen port", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("51820")).toBeInTheDocument();
    });

    const portInput = screen.getByLabelText("Listen port");
    await user.clear(portInput);
    await user.type(portInput, "51821");
    expect(portInput).toHaveValue(51821);
  });

  it("edits addresses field", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("10.0.0.1/24")).toBeInTheDocument();
    });

    const addrInput = screen.getByLabelText("Server addresses");
    await user.clear(addrInput);
    await user.type(addrInput, "10.0.0.2/24");
    expect(addrInput).toHaveValue("10.0.0.2/24");
  });
});

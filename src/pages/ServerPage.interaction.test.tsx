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
});

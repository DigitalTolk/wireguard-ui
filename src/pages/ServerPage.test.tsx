import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { ServerPage } from "./ServerPage";

describe("ServerPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows heading and save button", async () => {
    cleanup = mockFetch({
      "/server": {
        Interface: { addresses: ["10.0.0.1/24"], listen_port: 51820 },
        KeyPair: { public_key: "pubkey123", private_key: "priv" },
      },
    });
    renderWithProviders(<ServerPage />);
    await waitFor(() => {
      expect(screen.getByText("Server Configuration")).toBeInTheDocument();
      expect(screen.getByText("Save")).toBeInTheDocument();
    });
  });

  it("renders server config values", async () => {
    cleanup = mockFetch({
      "/server": {
        Interface: { addresses: ["10.252.1.0/24"], listen_port: 51820 },
        KeyPair: { public_key: "serverpubkey123", private_key: "priv" },
      },
    });

    renderWithProviders(<ServerPage />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("serverpubkey123")).toBeInTheDocument();
    });
  });
});

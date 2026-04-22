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

  it("clicks apply config", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/server": serverData,
      "/server/apply-config": { message: "ok" },
    });

    renderWithProviders(<ServerPage />);
    await waitFor(() => {
      expect(screen.getByText("Apply Config")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Apply Config"));
  });

  it("displays server addresses", async () => {
    cleanup = mockFetch({ "/server": serverData });
    renderWithProviders(<ServerPage />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("10.0.0.1/24")).toBeInTheDocument();
      expect(screen.getByDisplayValue("51820")).toBeInTheDocument();
    });
  });
});

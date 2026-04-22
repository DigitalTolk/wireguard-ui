import { describe, it, expect, vi, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { ClientsPage } from "./ClientsPage";

const sampleClient = {
  Client: {
    id: "c1",
    name: "Test Client",
    email: "test@example.com",
    enabled: true,
    allocated_ips: ["10.0.0.2/32"],
    allowed_ips: ["0.0.0.0/0"],
    additional_notes: "some notes",
    public_key: "pk1",
    private_key: "priv1",
  },
  QRCode: "data:image/png;base64,abc123",
};

describe("ClientsPage interactions", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("toggles client status", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/clients": [sampleClient],
      "/clients/c1/status": { ...sampleClient.Client, enabled: false },
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Test Client")).toBeInTheDocument();
    });

    const toggle = screen.getByRole("switch");
    await user.click(toggle);
  });

  it("opens QR code dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/clients": [sampleClient] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Test Client")).toBeInTheDocument();
    });

    const qrButton = screen.getByLabelText("Show QR code for Test Client");
    await user.click(qrButton);

    await waitFor(() => {
      expect(screen.getByText("Test Client - QR Code")).toBeInTheDocument();
    });
  });

  it("shows download button", async () => {
    cleanup = mockFetch({ "/clients": [sampleClient] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Download config for Test Client")).toBeInTheDocument();
    });
  });

  it("shows delete button and confirms", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(false);
    cleanup = mockFetch({ "/clients": [sampleClient] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Delete Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Delete Test Client"));
    expect(window.confirm).toHaveBeenCalled();
  });

  it("displays additional notes", async () => {
    cleanup = mockFetch({ "/clients": [sampleClient] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Notes: some notes")).toBeInTheDocument();
    });
  });

  it("shows disabled badge for disabled clients", async () => {
    const disabledClient = {
      ...sampleClient,
      Client: { ...sampleClient.Client, enabled: false },
    };
    cleanup = mockFetch({ "/clients": [disabledClient] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Disabled")).toBeInTheDocument();
    });
  });

  it("shows client count badge", async () => {
    cleanup = mockFetch({ "/clients": [sampleClient] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("1 clients")).toBeInTheDocument();
    });
  });
});

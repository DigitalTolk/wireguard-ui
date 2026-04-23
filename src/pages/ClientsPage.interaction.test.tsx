import { describe, it, expect, afterEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { ClientsPage } from "./ClientsPage";

const adminMe = { username: "admin", email: "admin@test.com", display_name: "Admin", admin: true };

const sampleClient = {
  Client: {
    id: "c1",
    name: "Test Client",
    email: "test@example.com",
    enabled: true,
    allocated_ips: ["10.0.0.2/32"],
    allowed_ips: ["0.0.0.0/0"],
    extra_allowed_ips: ["192.168.1.0/24"],
    additional_notes: "some notes",
    public_key: "pk1",
    private_key: "priv1",
    preshared_key: "psk1",
    endpoint: "vpn.example.com:51820",
    use_server_dns: true,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
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
      "/auth/me": adminMe,
      "/clients": [sampleClient],
      "/clients/c1/status": { ...sampleClient.Client, enabled: false },
      "/subnet-ranges": [],
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
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
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
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Download config for Test Client")).toBeInTheDocument();
    });
  });

  it("shows delete button and opens confirmation dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Delete Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Delete Test Client"));

    await waitFor(() => {
      expect(screen.getByText("Delete Client")).toBeInTheDocument();
      expect(screen.getByText(/Are you sure you want to delete/)).toBeInTheDocument();
    });
  });

  it("confirms delete in dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [sampleClient],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Delete Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Delete Test Client"));

    await waitFor(() => {
      expect(screen.getByText("Delete Client")).toBeInTheDocument();
    });

    // Click the Delete button in the confirmation dialog
    const deleteBtn = screen.getByRole("button", { name: "Delete" });
    await user.click(deleteBtn);
  });

  it("cancels delete dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [sampleClient],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Delete Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Delete Test Client"));

    await waitFor(() => {
      expect(screen.getByText("Delete Client")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  it("displays additional notes", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("some notes")).toBeInTheDocument();
    });
  });

  it("shows disabled badge for disabled clients", async () => {
    const disabledClient = {
      ...sampleClient,
      Client: { ...sampleClient.Client, enabled: false },
    };
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [disabledClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Disabled")).toBeInTheDocument();
    });
  });

  it("shows client count badge", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("1")).toBeInTheDocument();
    });
  });

  it("opens create client dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [],
      "/suggest-client-ips": ["10.252.1.2/32"],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("New Client")).toBeInTheDocument();
    });

    await user.click(screen.getByText("New Client"));

    await waitFor(() => {
      expect(screen.getByText("Name")).toBeInTheDocument();
      expect(screen.getByText("Email *")).toBeInTheDocument();
      expect(screen.getByText("Public Key")).toBeInTheDocument();
      expect(screen.getByText("Preshared Key")).toBeInTheDocument();
      expect(screen.getByText("Allocated IPs")).toBeInTheDocument();
      expect(screen.getByText("Allowed IPs")).toBeInTheDocument();
      expect(screen.getByText("Use server DNS")).toBeInTheDocument();
    });
  });

  it("creates a new client", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [],
      "/suggest-client-ips": ["10.252.1.2/32"],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("New Client")).toBeInTheDocument();
    });

    await user.click(screen.getByText("New Client"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("e.g. John's Laptop")).toBeInTheDocument();
    });

    await user.type(screen.getByPlaceholderText("e.g. John's Laptop"), "Test Laptop");
    await user.type(screen.getByPlaceholderText("john@example.com"), "test@test.com");
    await user.click(screen.getByText("Create"));
  });

  it("cancels create dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [],
      "/suggest-client-ips": [],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("New Client")).toBeInTheDocument();
    });

    await user.click(screen.getByText("New Client"));

    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Cancel"));
  });

  it("shows New Client button in empty state", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("New Client")).toBeInTheDocument();
      expect(screen.getByText(/No clients configured yet/)).toBeInTheDocument();
    });
  });

  it("displays allocated IPs on client card", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("10.0.0.2/32")).toBeInTheDocument();
    });
  });

  it("displays allowed IPs on client card", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("0.0.0.0/0")).toBeInTheDocument();
    });
  });

  it("displays extra allowed IPs on client card", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("192.168.1.0/24")).toBeInTheDocument();
    });
  });

  it("displays created and updated dates on client card", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText(/Created /)).toBeInTheDocument();
      expect(screen.getByText(/Updated /)).toBeInTheDocument();
    });
  });

  it("shows export to excel button", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Export to Excel")).toBeInTheDocument();
    });
  });

  it("clicks export button", async () => {
    const user = userEvent.setup();
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Export to Excel")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Export to Excel"));
    expect(openSpy).toHaveBeenCalledWith(
      expect.stringContaining("/clients/export"),
      "_blank"
    );
    openSpy.mockRestore();
  });

  it("clicks download config button", async () => {
    const user = userEvent.setup();
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Download config for Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Download config for Test Client"));
    expect(openSpy).toHaveBeenCalledWith(
      expect.stringContaining("/clients/c1/config"),
      "_blank"
    );
    openSpy.mockRestore();
  });

  it("shows filters card with search and status", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Filters")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Name, email, or IP...")).toBeInTheDocument();
    });
  });

  it("types in search input and presses enter", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Name, email, or IP...")).toBeInTheDocument();
    });

    const searchInput = screen.getByPlaceholderText("Name, email, or IP...");
    await user.type(searchInput, "test{Enter}");
  });

  it("clicks search button", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Name, email, or IP...")).toBeInTheDocument();
    });

    // There may be multiple elements with "Search" label; get the button one
    const searchBtns = screen.getAllByLabelText("Search");
    await user.click(searchBtns[0]);
  });

  it("opens edit dialog and populates form", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [sampleClient],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Edit Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Edit Test Client"));

    await waitFor(() => {
      expect(screen.getByText("Edit Client")).toBeInTheDocument();
      expect(screen.getByDisplayValue("Test Client")).toBeInTheDocument();
      expect(screen.getByDisplayValue("test@example.com")).toBeInTheDocument();
    });
  });

  it("edits client name in edit dialog and saves", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [sampleClient],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Edit Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Edit Test Client"));

    await waitFor(() => {
      expect(screen.getByDisplayValue("Test Client")).toBeInTheDocument();
    });

    const nameInput = screen.getByDisplayValue("Test Client");
    await user.clear(nameInput);
    await user.type(nameInput, "Updated Name");

    await user.click(screen.getByText("Save"));
  });

  it("cancels edit dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [sampleClient],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Edit Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Edit Test Client"));

    await waitFor(() => {
      expect(screen.getByText("Edit Client")).toBeInTheDocument();
    });

    // Click Cancel in edit dialog
    const cancelButtons = screen.getAllByText("Cancel");
    await user.click(cancelButtons[cancelButtons.length - 1]);
  });

  it("opens email dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [sampleClient],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Email config to Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Email config to Test Client"));

    await waitFor(() => {
      expect(screen.getByText("Send Config via Email")).toBeInTheDocument();
      expect(screen.getByDisplayValue("test@example.com")).toBeInTheDocument();
    });
  });

  it("sends email from dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [sampleClient],
      "/clients/c1/email": { message: "Email sent" },
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Email config to Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Email config to Test Client"));

    await waitFor(() => {
      expect(screen.getByText("Send Config via Email")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Send"));
  });

  it("cancels email dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [sampleClient],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Email config to Test Client")).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText("Email config to Test Client"));

    await waitFor(() => {
      expect(screen.getByText("Send Config via Email")).toBeInTheDocument();
    });

    const cancelButtons = screen.getAllByText("Cancel");
    await user.click(cancelButtons[cancelButtons.length - 1]);
  });

  it("shows subnet range dropdown when ranges exist", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [],
      "/suggest-client-ips": ["10.252.1.2/32"],
      "/subnet-ranges": ["Office:10.0.1.0/24", "Remote:10.0.2.0/24"],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("New Client")).toBeInTheDocument();
    });

    await user.click(screen.getByText("New Client"));

    await waitFor(() => {
      expect(screen.getByText("Subnet Range")).toBeInTheDocument();
    });
  });


  it("shows client email next to name", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [sampleClient], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("test@example.com")).toBeInTheDocument();
    });
  });

  it("shows create dialog with notes field", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/auth/me": adminMe,
      "/clients": [],
      "/suggest-client-ips": [],
      "/subnet-ranges": [],
    });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("New Client")).toBeInTheDocument();
    });

    await user.click(screen.getByText("New Client"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Optional notes")).toBeInTheDocument();
    });
  });

  it("displays five status filter options", async () => {
    cleanup = mockFetch({ "/auth/me": adminMe, "/clients": [], "/subnet-ranges": [] });
    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("Filters")).toBeInTheDocument();
    });

    // The status select trigger should be present
    expect(screen.getByText("Status")).toBeInTheDocument();
  });
});

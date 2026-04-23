import { describe, it, expect, vi, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { WolPage } from "./WolPage";

const host = { MacAddress: "AA-BB-CC-DD-EE-FF", Name: "Server1", LatestUsed: null };
const hostUsed = { MacAddress: "11:22:33:44:55:66", Name: "Server2", LatestUsed: "2026-03-15T08:00:00Z" };

describe("WolPage interactions", () => {
  let cleanup: () => void;
  afterEach(() => { cleanup?.(); });

  it("sends wake packet", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/wol-hosts": [host],
      "/wake": { ...host, LatestUsed: new Date().toISOString() },
    });

    renderWithProviders(<WolPage />);
    await waitFor(() => expect(screen.getByText("Server1")).toBeInTheDocument());

    await user.click(screen.getByLabelText("Wake Server1"));
  });

  it("deletes host with confirmation", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    cleanup = mockFetch({
      "/wol-hosts": [host],
    });

    renderWithProviders(<WolPage />);
    await waitFor(() => expect(screen.getByText("Server1")).toBeInTheDocument());

    await user.click(screen.getByLabelText("Delete Server1"));
    expect(window.confirm).toHaveBeenCalled();
  });

  it("cancels host deletion", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(false);
    cleanup = mockFetch({
      "/wol-hosts": [host],
    });

    renderWithProviders(<WolPage />);
    await waitFor(() => expect(screen.getByText("Server1")).toBeInTheDocument());

    await user.click(screen.getByLabelText("Delete Server1"));
    expect(window.confirm).toHaveBeenCalled();
  });

  it("shows Never for unused hosts", async () => {
    cleanup = mockFetch({ "/wol-hosts": [host] });
    renderWithProviders(<WolPage />);
    await waitFor(() => expect(screen.getByText("Never")).toBeInTheDocument());
  });

  it("shows formatted date for used hosts", async () => {
    cleanup = mockFetch({ "/wol-hosts": [hostUsed] });
    renderWithProviders(<WolPage />);
    await waitFor(() => {
      expect(screen.getByText("Server2")).toBeInTheDocument();
      // The date should be formatted, not "Never"
      expect(screen.queryByText("Never")).not.toBeInTheDocument();
    });
  });

  it("opens create host dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/wol-hosts": [] });
    renderWithProviders(<WolPage />);

    await waitFor(() => expect(screen.getByText("New Host")).toBeInTheDocument());

    await user.click(screen.getByText("New Host"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("e.g. File Server")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("AA:BB:CC:DD:EE:FF")).toBeInTheDocument();
    });
  });

  it("fills and submits create host form", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/wol-hosts": [],
    });
    renderWithProviders(<WolPage />);

    await waitFor(() => expect(screen.getByText("New Host")).toBeInTheDocument());
    await user.click(screen.getByText("New Host"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("e.g. File Server")).toBeInTheDocument();
    });

    await user.type(screen.getByPlaceholderText("e.g. File Server"), "Test Server");
    await user.type(screen.getByPlaceholderText("AA:BB:CC:DD:EE:FF"), "11:22:33:44:55:66");

    await user.click(screen.getByText("Create"));
  });

  it("cancels create host dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/wol-hosts": [] });
    renderWithProviders(<WolPage />);

    await waitFor(() => expect(screen.getByText("New Host")).toBeInTheDocument());
    await user.click(screen.getByText("New Host"));

    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Cancel"));
  });

  it("shows table headers", async () => {
    cleanup = mockFetch({ "/wol-hosts": [] });
    renderWithProviders(<WolPage />);

    await waitFor(() => {
      expect(screen.getByText("Name")).toBeInTheDocument();
      expect(screen.getByText("MAC Address")).toBeInTheDocument();
      expect(screen.getByText("Last Used")).toBeInTheDocument();
      expect(screen.getByText("Actions")).toBeInTheDocument();
    });
  });

  it("shows multiple hosts", async () => {
    cleanup = mockFetch({ "/wol-hosts": [host, hostUsed] });
    renderWithProviders(<WolPage />);

    await waitFor(() => {
      expect(screen.getByText("Server1")).toBeInTheDocument();
      expect(screen.getByText("Server2")).toBeInTheDocument();
    });
  });

  it("handles wake host error", async () => {
    const user = userEvent.setup();
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/wake") && init?.method === "POST") {
        return {
          ok: false,
          status: 500,
          json: async () => ({ error: { code: "INTERNAL", message: "Wake failed" } }),
          text: async () => "error",
          headers: new Headers(),
        } as Response;
      }
      if (url.includes("/wol-hosts")) {
        return {
          ok: true,
          status: 200,
          json: async () => [host],
          text: async () => JSON.stringify([host]),
          headers: new Headers(),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    });
    cleanup = () => { globalThis.fetch = originalFetch; };

    renderWithProviders(<WolPage />);
    await waitFor(() => expect(screen.getByText("Server1")).toBeInTheDocument());

    await user.click(screen.getByLabelText("Wake Server1"));

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/wake"),
        expect.objectContaining({ method: "POST" })
      );
    });
  });

  it("handles delete host error", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/wol-hosts/") && init?.method === "DELETE") {
        return {
          ok: false,
          status: 500,
          json: async () => ({ error: { code: "INTERNAL", message: "Delete failed" } }),
          text: async () => "error",
          headers: new Headers(),
        } as Response;
      }
      if (url.includes("/wol-hosts")) {
        return {
          ok: true,
          status: 200,
          json: async () => [host],
          text: async () => JSON.stringify([host]),
          headers: new Headers(),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    });
    cleanup = () => { globalThis.fetch = originalFetch; };

    renderWithProviders(<WolPage />);
    await waitFor(() => expect(screen.getByText("Server1")).toBeInTheDocument());

    await user.click(screen.getByLabelText("Delete Server1"));
    expect(window.confirm).toHaveBeenCalled();

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/wol-hosts/"),
        expect.objectContaining({ method: "DELETE" })
      );
    });
  });

  it("handles create host error", async () => {
    const user = userEvent.setup();
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/wol-hosts") && init?.method === "POST") {
        return {
          ok: false,
          status: 500,
          json: async () => ({ error: { code: "INTERNAL", message: "Create failed" } }),
          text: async () => "error",
          headers: new Headers(),
        } as Response;
      }
      if (url.includes("/wol-hosts")) {
        return {
          ok: true,
          status: 200,
          json: async () => [],
          text: async () => "[]",
          headers: new Headers(),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    });
    cleanup = () => { globalThis.fetch = originalFetch; };

    renderWithProviders(<WolPage />);
    await waitFor(() => expect(screen.getByText("New Host")).toBeInTheDocument());

    await user.click(screen.getByText("New Host"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("e.g. File Server")).toBeInTheDocument();
    });

    await user.type(screen.getByPlaceholderText("e.g. File Server"), "Test Server");
    await user.type(screen.getByPlaceholderText("AA:BB:CC:DD:EE:FF"), "11:22:33:44:55:66");

    await user.click(screen.getByText("Create"));

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/wol-hosts"),
        expect.objectContaining({ method: "POST" })
      );
    });
  });

  it("shows validation error for empty name in create dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/wol-hosts": [] });
    renderWithProviders(<WolPage />);

    await waitFor(() => expect(screen.getByText("New Host")).toBeInTheDocument());

    await user.click(screen.getByText("New Host"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("AA:BB:CC:DD:EE:FF")).toBeInTheDocument();
    });

    // Type only MAC, leave name empty
    await user.type(screen.getByPlaceholderText("AA:BB:CC:DD:EE:FF"), "11:22:33:44:55:66");

    // The create button should be disabled because name is missing
    const createBtn = screen.getByText("Create").closest("button");
    expect(createBtn).toBeDisabled();
  });

  it("shows validation error for invalid MAC format in create dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/wol-hosts": [] });
    renderWithProviders(<WolPage />);

    await waitFor(() => expect(screen.getByText("New Host")).toBeInTheDocument());

    await user.click(screen.getByText("New Host"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("e.g. File Server")).toBeInTheDocument();
    });

    await user.type(screen.getByPlaceholderText("e.g. File Server"), "Test");
    await user.type(screen.getByPlaceholderText("AA:BB:CC:DD:EE:FF"), "invalid-mac");

    await waitFor(() => {
      expect(screen.getByText(/Invalid MAC format/)).toBeInTheDocument();
    });
  });

  it("shows validation errors for empty name and MAC in create dialog", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({ "/wol-hosts": [] });
    renderWithProviders(<WolPage />);

    await waitFor(() => expect(screen.getByText("New Host")).toBeInTheDocument());

    await user.click(screen.getByText("New Host"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("e.g. File Server")).toBeInTheDocument();
    });

    // Both fields empty - create should be disabled
    const createBtn = screen.getByText("Create").closest("button");
    expect(createBtn).toBeDisabled();

    // Verify the errors exist
    expect(screen.getByText("Name is required")).toBeInTheDocument();
    expect(screen.getByText("MAC address is required")).toBeInTheDocument();
  });
});

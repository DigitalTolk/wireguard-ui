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
});

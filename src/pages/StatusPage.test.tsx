import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { StatusPage } from "./StatusPage";

const connectedPeer = {
  name: "Client1",
  email: "c1@test.com",
  public_key: "pk1abcdef1234567890",
  received_bytes: 1024,
  transmit_bytes: 2048,
  last_handshake_time: new Date().toISOString(),
  last_handshake_rel: 60_000_000_000, // 60 seconds in nanos
  connected: true,
  allocated_ip: "10.0.0.2/32",
  endpoint: "1.2.3.4:51820",
};

const disconnectedPeer = {
  name: "Client2",
  email: "c2@test.com",
  public_key: "pk2abcdef1234567890",
  received_bytes: 0,
  transmit_bytes: 0,
  last_handshake_time: "",
  last_handshake_rel: 0,
  connected: false,
  allocated_ip: "10.0.0.3/32",
  endpoint: "",
};

describe("StatusPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows heading", async () => {
    cleanup = mockFetch({ "/status": [] });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Server Status")).toBeInTheDocument();
    });
  });

  it("shows no interfaces message when empty", async () => {
    cleanup = mockFetch({ "/status": [] });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("No WireGuard interfaces found")).toBeInTheDocument();
    });
  });

  it("renders device with peers", async () => {
    cleanup = mockFetch({
      "/status": [
        {
          name: "wg0",
          peers: [connectedPeer],
        },
      ],
    });

    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("wg0")).toBeInTheDocument();
      expect(screen.getByText("Client1")).toBeInTheDocument();
      expect(screen.getByText("1.2.3.4:51820")).toBeInTheDocument();
    });
  });

  it("shows sortable column headers", async () => {
    cleanup = mockFetch({ "/status": [{ name: "wg0", peers: [] }] });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Name")).toBeInTheDocument();
      expect(screen.getByText("Handshake")).toBeInTheDocument();
      expect(screen.getByText("Endpoint")).toBeInTheDocument();
    });
  });

  it("shows 'No peers connected' when device has no peers", async () => {
    cleanup = mockFetch({ "/status": [{ name: "wg0", peers: [] }] });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("No peers connected")).toBeInTheDocument();
    });
  });

  it("shows 'No peers connected' when peers is null", async () => {
    cleanup = mockFetch({ "/status": [{ name: "wg0", peers: null }] });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("No peers connected")).toBeInTheDocument();
    });
  });

  it("displays disconnected peer with dash for endpoint", async () => {
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [disconnectedPeer] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Client2")).toBeInTheDocument();
      expect(screen.getByText("-")).toBeInTheDocument();
    });
  });

  it("shows 'Unknown' when peer has no name", async () => {
    const namelessPeer = { ...connectedPeer, name: "" };
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [namelessPeer] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Unknown")).toBeInTheDocument();
    });
  });

  it("displays truncated public key", async () => {
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [connectedPeer] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("pk1abcdef1234567...")).toBeInTheDocument();
    });
  });

  it("formats bytes correctly for various sizes", async () => {
    const largePeer = {
      ...connectedPeer,
      received_bytes: 1_500_000_000, // ~1.4 GB
      transmit_bytes: 2_500_000, // ~2.4 MB
    };
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [largePeer] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("1.4 GB")).toBeInTheDocument();
      expect(screen.getByText("2.4 MB")).toBeInTheDocument();
    });
  });

  it("formats zero bytes as '0 B'", async () => {
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [disconnectedPeer] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      const zeroBytes = screen.getAllByText("0 B");
      expect(zeroBytes.length).toBe(2); // rx and tx both 0
    });
  });

  it("formats handshake as 'Never' for zero/negative nanos", async () => {
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [disconnectedPeer] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Never")).toBeInTheDocument();
    });
  });

  it("formats handshake in seconds", async () => {
    const peerSec = { ...connectedPeer, last_handshake_rel: 30_000_000_000 }; // 30s
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerSec] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("30s ago")).toBeInTheDocument();
    });
  });

  it("formats handshake in minutes", async () => {
    const peerMin = { ...connectedPeer, last_handshake_rel: 300_000_000_000 }; // 5m
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerMin] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("5m ago")).toBeInTheDocument();
    });
  });

  it("formats handshake in hours", async () => {
    const peerHr = { ...connectedPeer, last_handshake_rel: 7_200_000_000_000 }; // 2h
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerHr] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("2h ago")).toBeInTheDocument();
    });
  });

  it("formats handshake in days", async () => {
    const peerDay = { ...connectedPeer, last_handshake_rel: 172_800_000_000_000 }; // 2d
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerDay] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("2d ago")).toBeInTheDocument();
    });
  });

  it("sorts peers by name ascending by default", async () => {
    const peerA = { ...connectedPeer, name: "Alpha", public_key: "pka1234567890123" };
    const peerB = { ...disconnectedPeer, name: "Bravo", public_key: "pkb1234567890123" };
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerB, peerA] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Alpha")).toBeInTheDocument();
      expect(screen.getByText("Bravo")).toBeInTheDocument();
    });
    // Alpha should come before Bravo in the DOM
    const rows = screen.getAllByRole("row");
    const alphaRow = rows.find((r) => within(r).queryByText("Alpha"));
    const bravoRow = rows.find((r) => within(r).queryByText("Bravo"));
    expect(rows.indexOf(alphaRow!)).toBeLessThan(rows.indexOf(bravoRow!));
  });

  it("toggles sort direction when clicking same column header", async () => {
    const user = userEvent.setup();
    const peerA = { ...connectedPeer, name: "Alpha", public_key: "pka1234567890123" };
    const peerB = { ...disconnectedPeer, name: "Bravo", public_key: "pkb1234567890123" };
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerA, peerB] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Alpha")).toBeInTheDocument();
    });

    // Click Name header to toggle to desc
    await user.click(screen.getByText("Name"));

    // Now Bravo should come before Alpha
    const rows = screen.getAllByRole("row");
    const alphaRow = rows.find((r) => within(r).queryByText("Alpha"));
    const bravoRow = rows.find((r) => within(r).queryByText("Bravo"));
    expect(rows.indexOf(bravoRow!)).toBeLessThan(rows.indexOf(alphaRow!));
  });

  it("sorts by a different column when clicking a new header", async () => {
    const user = userEvent.setup();
    const peerLowRx = { ...connectedPeer, name: "LowRx", public_key: "pkl1234567890123", received_bytes: 100 };
    const peerHighRx = { ...connectedPeer, name: "HighRx", public_key: "pkh1234567890123", received_bytes: 999999 };
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerHighRx, peerLowRx] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("LowRx")).toBeInTheDocument();
    });

    // Click Rx header to sort by received_bytes asc
    await user.click(screen.getByText("Rx"));

    const rows = screen.getAllByRole("row");
    const lowRow = rows.find((r) => within(r).queryByText("LowRx"));
    const highRow = rows.find((r) => within(r).queryByText("HighRx"));
    expect(rows.indexOf(lowRow!)).toBeLessThan(rows.indexOf(highRow!));
  });

  it("sorts by Tx column", async () => {
    const user = userEvent.setup();
    const peerLowTx = { ...connectedPeer, name: "LowTx", public_key: "pkl1234567890123", transmit_bytes: 50 };
    const peerHighTx = { ...connectedPeer, name: "HighTx", public_key: "pkh1234567890123", transmit_bytes: 999999 };
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerHighTx, peerLowTx] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("LowTx")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Tx"));

    const rows = screen.getAllByRole("row");
    const lowRow = rows.find((r) => within(r).queryByText("LowTx"));
    const highRow = rows.find((r) => within(r).queryByText("HighTx"));
    expect(rows.indexOf(lowRow!)).toBeLessThan(rows.indexOf(highRow!));
  });

  it("sorts by Endpoint column", async () => {
    const user = userEvent.setup();
    const peerA = { ...connectedPeer, name: "PeerA", public_key: "pka1234567890123", endpoint: "a.example.com:51820" };
    const peerB = { ...connectedPeer, name: "PeerB", public_key: "pkb1234567890123", endpoint: "z.example.com:51820" };
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerB, peerA] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("PeerA")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Endpoint"));

    const rows = screen.getAllByRole("row");
    const aRow = rows.find((r) => within(r).queryByText("PeerA"));
    const bRow = rows.find((r) => within(r).queryByText("PeerB"));
    expect(rows.indexOf(aRow!)).toBeLessThan(rows.indexOf(bRow!));
  });

  it("sorts by Handshake column", async () => {
    const user = userEvent.setup();
    const peerRecent = { ...connectedPeer, name: "Recent", public_key: "pkr1234567890123", last_handshake_rel: 5_000_000_000 };
    const peerOld = { ...connectedPeer, name: "Old", public_key: "pko1234567890123", last_handshake_rel: 999_000_000_000 };
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerOld, peerRecent] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Recent")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Handshake"));

    const rows = screen.getAllByRole("row");
    const recentRow = rows.find((r) => within(r).queryByText("Recent"));
    const oldRow = rows.find((r) => within(r).queryByText("Old"));
    expect(rows.indexOf(recentRow!)).toBeLessThan(rows.indexOf(oldRow!));
  });

  it("sorts by connected status column", async () => {
    const user = userEvent.setup();
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [disconnectedPeer, connectedPeer] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("Client1")).toBeInTheDocument();
      expect(screen.getByText("Client2")).toBeInTheDocument();
    });

    // The connected column header has no text label, it just has SortIcon
    // We need to find and click the second TableHead (the one for connected)
    const headerRow = screen.getAllByRole("row")[0];
    const headerCells = within(headerRow).getAllByRole("columnheader");
    // connected is the 2nd column header
    await user.click(headerCells[1]);
  });

  it("renders multiple devices", async () => {
    cleanup = mockFetch({
      "/status": [
        { name: "wg0", peers: [connectedPeer] },
        { name: "wg1", peers: [disconnectedPeer] },
      ],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("wg0")).toBeInTheDocument();
      expect(screen.getByText("wg1")).toBeInTheDocument();
    });
  });

  it("formats KB correctly", async () => {
    const peerKB = {
      ...connectedPeer,
      received_bytes: 5120, // 5 KB
      transmit_bytes: 512000, // 500 KB
    };
    cleanup = mockFetch({
      "/status": [{ name: "wg0", peers: [peerKB] }],
    });
    renderWithProviders(<StatusPage />);
    await waitFor(() => {
      expect(screen.getByText("5.0 KB")).toBeInTheDocument();
      expect(screen.getByText("500.0 KB")).toBeInTheDocument();
    });
  });
});

import { describe, it, expect, vi, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { WolPage } from "./WolPage";

const host = { MacAddress: "AA-BB-CC-DD-EE-FF", Name: "Server1", LatestUsed: null };

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

  it("shows Never for unused hosts", async () => {
    cleanup = mockFetch({ "/wol-hosts": [host] });
    renderWithProviders(<WolPage />);
    await waitFor(() => expect(screen.getByText("Never")).toBeInTheDocument());
  });
});

import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { WolPage } from "./WolPage";

describe("WolPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows heading", async () => {
    cleanup = mockFetch({ "/wol-hosts": [] });
    renderWithProviders(<WolPage />);
    await waitFor(() => {
      expect(screen.getByText("Wake-on-LAN")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    cleanup = mockFetch({ "/wol-hosts": [] });
    renderWithProviders(<WolPage />);
    await waitFor(() => {
      expect(screen.getByText("No Wake-on-LAN hosts configured")).toBeInTheDocument();
    });
  });

  it("renders hosts from API", async () => {
    cleanup = mockFetch({
      "/wol-hosts": [
        { MacAddress: "AA-BB-CC-DD-EE-FF", Name: "Server1", LatestUsed: null },
      ],
    });

    renderWithProviders(<WolPage />);
    await waitFor(() => {
      expect(screen.getByText("Server1")).toBeInTheDocument();
      expect(screen.getByText("AA-BB-CC-DD-EE-FF")).toBeInTheDocument();
    });
  });
});

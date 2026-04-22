import { describe, it, expect, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { UsersPage } from "./UsersPage";

describe("UsersPage", () => {
  let cleanup: () => void;

  afterEach(() => {
    cleanup?.();
  });

  it("shows heading", async () => {
    cleanup = mockFetch({ "/users": [] });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("Users")).toBeInTheDocument();
    });
  });

  it("renders user list", async () => {
    cleanup = mockFetch({
      "/users": [
        { username: "admin", email: "admin@test.com", admin: true },
        { username: "user1", email: "user@test.com", admin: false },
      ],
    });

    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
      expect(screen.getByText("user1")).toBeInTheDocument();
    });
  });
});

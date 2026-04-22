import { describe, it, expect, vi, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockFetch } from "@/test/test-utils";
import { UsersPage } from "./UsersPage";

describe("UsersPage interactions", () => {
  let cleanup: () => void;
  afterEach(() => { cleanup?.(); });

  it("shows admin badge for admin users", async () => {
    cleanup = mockFetch({
      "/users": [
        { username: "admin", email: "admin@test.com", admin: true },
      ],
    });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("Admin")).toBeInTheDocument();
    });
  });

  it("shows User badge for non-admin users", async () => {
    cleanup = mockFetch({
      "/users": [
        { username: "regular", email: "user@test.com", admin: false },
      ],
    });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("User")).toBeInTheDocument();
    });
  });

  it("clicks delete with confirmation", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(false);
    cleanup = mockFetch({
      "/users": [
        { username: "deluser", email: "del@test.com", admin: false },
      ],
    });
    renderWithProviders(<UsersPage />);
    await waitFor(() => expect(screen.getByText("deluser")).toBeInTheDocument());

    await user.click(screen.getByLabelText('Delete user deluser'));
    expect(window.confirm).toHaveBeenCalled();
  });

  it("shows email or dash", async () => {
    cleanup = mockFetch({
      "/users": [
        { username: "noemail", email: "", admin: false },
      ],
    });
    renderWithProviders(<UsersPage />);
    await waitFor(() => {
      expect(screen.getByText("-")).toBeInTheDocument();
    });
  });
});

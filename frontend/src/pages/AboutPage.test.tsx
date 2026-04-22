import { describe, it, expect } from "vitest";
import { renderWithProviders } from "@/test/test-utils";
import { AboutPage } from "./AboutPage";

describe("AboutPage", () => {
  it("renders the about heading", () => {
    const { getByText } = renderWithProviders(<AboutPage />);
    expect(getByText("About")).toBeInTheDocument();
  });

  it("renders the project link", () => {
    const { getByText } = renderWithProviders(<AboutPage />);
    expect(getByText("wireguard-ui")).toBeInTheDocument();
  });
});

import { describe, it, expect } from "vitest";
import { renderWithProviders } from "@/test/test-utils";
import { AuditPage } from "./AuditPage";

describe("AuditPage", () => {
  it("renders audit log heading", () => {
    const { getByText } = renderWithProviders(<AuditPage />);
    expect(getByText("Audit Logs")).toBeInTheDocument();
  });

  it("shows placeholder message", () => {
    const { getByText } = renderWithProviders(<AuditPage />);
    expect(getByText(/audit.*available/i)).toBeInTheDocument();
  });
});

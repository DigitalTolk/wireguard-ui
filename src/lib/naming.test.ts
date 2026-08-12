import { describe, it, expect } from "vitest";
import { applyNamePattern } from "./naming";

describe("applyNamePattern", () => {
  it("transforms with the documented example", () => {
    expect(
      applyNamePattern(
        "first.last@example.com",
        "^([A-Za-z0-9]+)\\.([A-Za-z0-9]+)@.+$",
        "abc-$1$2-def",
      ),
    ).toBe("abc-firstlast-def");
  });

  it("returns empty string when pattern is empty", () => {
    expect(applyNamePattern("anything@example.com", "", "$1")).toBe("");
  });

  it("returns empty string for an invalid regex (no throw)", () => {
    expect(applyNamePattern("user@example.com", "([unclosed", "$1")).toBe("");
  });

  it("returns empty string when the pattern does not match", () => {
    expect(applyNamePattern("noatsign", "^(.+)@(.+)$", "$1-$2")).toBe("");
  });

  it("supports a single capture group", () => {
    expect(applyNamePattern("alice@example.com", "^([a-z]+)@.+$", "user-$1")).toBe("user-alice");
  });
});

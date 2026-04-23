import { describe, it, expect, vi, afterEach } from "vitest";
import { render } from "@testing-library/react";
import { ThemeProvider } from "./ThemeProvider";

describe("ThemeProvider", () => {
  afterEach(() => {
    document.documentElement.classList.remove("dark");
  });

  it("renders children", () => {
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });

    const { getByText } = render(
      <ThemeProvider>
        <span>child content</span>
      </ThemeProvider>
    );
    expect(getByText("child content")).toBeInTheDocument();
  });

  it("adds dark class when system prefers dark", () => {
    const matchMedia = vi.fn().mockReturnValue({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });
    window.matchMedia = matchMedia;

    render(
      <ThemeProvider>
        <span>dark</span>
      </ThemeProvider>
    );

    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("does not add dark class when system prefers light", () => {
    const matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });
    window.matchMedia = matchMedia;

    render(
      <ThemeProvider>
        <span>light</span>
      </ThemeProvider>
    );

    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("listens for changes and cleans up", () => {
    const addEventListener = vi.fn();
    const removeEventListener = vi.fn();
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener,
      removeEventListener,
    });

    const { unmount } = render(
      <ThemeProvider>
        <span>test</span>
      </ThemeProvider>
    );

    expect(addEventListener).toHaveBeenCalledWith("change", expect.any(Function));
    unmount();
    expect(removeEventListener).toHaveBeenCalledWith("change", expect.any(Function));
  });
});

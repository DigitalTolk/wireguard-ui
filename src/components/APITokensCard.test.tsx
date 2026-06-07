import { describe, it, expect, afterEach, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/test-utils";
import { APITokensCard } from "./APITokensCard";

interface Route {
  match: (url: string, method: string) => boolean;
  respond: (
    url: string,
    init: RequestInit | undefined,
  ) => Response | Promise<Response>;
}

function jsonRes(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
    text: async () => JSON.stringify(body),
    headers: new Headers(),
  } as Response;
}

function installFetch(routes: Route[]) {
  const orig = globalThis.fetch;
  globalThis.fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === "string" ? input : input.toString();
    const method = (init?.method ?? "GET").toUpperCase();
    for (const r of routes) {
      if (r.match(url, method)) return r.respond(url, init);
    }
    return jsonRes({ error: { code: "NOT_FOUND", message: "no route" } }, 404);
  });
  return () => {
    globalThis.fetch = orig;
  };
}

const sampleToken = {
  id: "tok-1",
  name: "deploy-bot",
  created_by: "admin",
  created_at: "2024-01-01T00:00:00Z",
  last_used_at: null,
  revoked_at: null,
};

describe("APITokensCard", () => {
  let cleanup: () => void;
  afterEach(() => cleanup?.());

  it("lists existing tokens", async () => {
    cleanup = installFetch([
      {
        match: (u, m) => m === "GET" && u.endsWith("/api-tokens"),
        respond: () => jsonRes([sampleToken]),
      },
    ]);
    renderWithProviders(<APITokensCard />);
    await waitFor(() => {
      expect(screen.getByText("deploy-bot")).toBeInTheDocument();
    });
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("shows empty state when no tokens exist", async () => {
    cleanup = installFetch([
      {
        match: (u, m) => m === "GET" && u.endsWith("/api-tokens"),
        respond: () => jsonRes([]),
      },
    ]);
    renderWithProviders(<APITokensCard />);
    await waitFor(() => {
      expect(screen.getByText("No tokens yet.")).toBeInTheDocument();
    });
  });

  it("creates a token and shows the plaintext exactly once", async () => {
    const user = userEvent.setup();
    let listCalls = 0;
    cleanup = installFetch([
      {
        match: (u, m) => m === "GET" && u.endsWith("/api-tokens"),
        respond: () => {
          listCalls += 1;
          // First call returns empty; after creation the list refresh adds the new one.
          return listCalls === 1 ? jsonRes([]) : jsonRes([sampleToken]);
        },
      },
      {
        match: (u, m) => m === "POST" && u.endsWith("/api-tokens"),
        respond: () =>
          jsonRes(
            { ...sampleToken, token: "wgui_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" },
            201,
          ),
      },
    ]);
    renderWithProviders(<APITokensCard />);

    await user.click(screen.getByText("New Token"));
    const nameInput = await screen.findByLabelText("Name");
    fireEvent.change(nameInput, { target: { value: "deploy-bot" } });
    await user.click(screen.getByRole("button", { name: "Create" }));

    // Plaintext appears in a once-only dialog
    await waitFor(() => {
      expect(screen.getByText("Token created")).toBeInTheDocument();
    });
    const plaintextInput = screen.getByLabelText("Token") as HTMLInputElement;
    expect(plaintextInput.value).toMatch(/^wgui_/);
  });

  it("does not surface the plaintext after dismissal", async () => {
    const user = userEvent.setup();
    let listCalls = 0;
    cleanup = installFetch([
      {
        match: (u, m) => m === "GET" && u.endsWith("/api-tokens"),
        respond: () => {
          listCalls += 1;
          return listCalls === 1 ? jsonRes([]) : jsonRes([sampleToken]);
        },
      },
      {
        match: (u, m) => m === "POST" && u.endsWith("/api-tokens"),
        respond: () =>
          jsonRes(
            { ...sampleToken, token: "wgui_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" },
            201,
          ),
      },
    ]);
    renderWithProviders(<APITokensCard />);

    await user.click(screen.getByText("New Token"));
    fireEvent.change(await screen.findByLabelText("Name"), { target: { value: "deploy-bot" } });
    await user.click(screen.getByRole("button", { name: "Create" }));

    const dismiss = await screen.findByRole("button", { name: /saved it/i });
    await user.click(dismiss);

    // After dismissal the plaintext input must be gone — no other route
    // brings it back, even after the list refresh.
    await waitFor(() => {
      expect(screen.queryByLabelText("Token")).not.toBeInTheDocument();
    });
  });

  it("revokes a token after confirmation", async () => {
    const user = userEvent.setup();
    let revoked = false;
    cleanup = installFetch([
      {
        match: (u, m) => m === "GET" && u.endsWith("/api-tokens"),
        respond: () => {
          if (revoked) {
            return jsonRes([{ ...sampleToken, revoked_at: "2024-01-02T00:00:00Z" }]);
          }
          return jsonRes([sampleToken]);
        },
      },
      {
        match: (u, m) => m === "DELETE" && u.includes(`/api-tokens/${sampleToken.id}`),
        respond: () => {
          revoked = true;
          return jsonRes({}, 204);
        },
      },
    ]);
    renderWithProviders(<APITokensCard />);

    await user.click(await screen.findByLabelText("Revoke deploy-bot"));
    await user.click(await screen.findByRole("button", { name: "Revoke" }));

    await waitFor(() => {
      expect(screen.getByText("Revoked")).toBeInTheDocument();
    });
  });

  it("does not revoke when the confirm dialog is cancelled", async () => {
    const user = userEvent.setup();
    const revokeMock = vi.fn();
    cleanup = installFetch([
      {
        match: (u, m) => m === "GET" && u.endsWith("/api-tokens"),
        respond: () => jsonRes([sampleToken]),
      },
      {
        match: (u, m) => m === "DELETE" && u.includes(`/api-tokens/`),
        respond: () => {
          revokeMock();
          return jsonRes({}, 204);
        },
      },
    ]);
    renderWithProviders(<APITokensCard />);

    await user.click(await screen.findByLabelText("Revoke deploy-bot"));
    await user.click(await screen.findByRole("button", { name: "Cancel" }));
    expect(revokeMock).not.toHaveBeenCalled();
  });
});

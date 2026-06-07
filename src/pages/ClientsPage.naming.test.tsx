import { describe, it, expect, afterEach, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/test-utils";
import { ClientsPage } from "./ClientsPage";

const adminMe = {
  username: "admin",
  email: "admin@test.com",
  display_name: "Admin",
  admin: true,
};

const defaultsResponse = {
  base_path: "",
  app_version: "test",
  git_commit: "abc",
  client_defaults: {
    AllowedIps: ["0.0.0.0/0"],
    ExtraAllowedIps: [],
    UseServerDNS: true,
  },
};

interface RouteHandler {
  match: (url: string, method: string) => boolean;
  respond: (
    url: string,
    init: RequestInit | undefined,
  ) => Response | Promise<Response>;
}

function jsonRes(data: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => data,
    text: async () => JSON.stringify(data),
    headers: new Headers(),
  } as Response;
}

function installFetch(routes: RouteHandler[]) {
  const orig = globalThis.fetch;
  globalThis.fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === "string" ? input : input.toString();
    const method = (init?.method ?? "GET").toUpperCase();
    for (const r of routes) {
      if (r.match(url, method)) return r.respond(url, init);
    }
    return jsonRes({}, 404);
  });
  return () => {
    globalThis.fetch = orig;
  };
}

function settingsRoute(extra: Record<string, string> = {}): RouteHandler {
  return {
    match: (u, m) => m === "GET" && u.endsWith("/settings"),
    respond: () =>
      jsonRes({
        endpoint_address: "vpn",
        dns_servers: ["1.1.1.1"],
        mtu: 1450,
        persistent_keepalive: 15,
        firewall_mark: "0xca6c",
        table: "auto",
        config_file_path: "/etc/wireguard/wg0.conf",
        client_name_pattern: "",
        client_name_replacement: "",
        email_filename_pattern: "",
        email_filename_replacement: "",
        ...extra,
        updated_at: "2024-01-01",
      }),
  };
}

const baseRoutes: RouteHandler[] = [
  { match: (u, m) => m === "GET" && u.includes("/auth/me"), respond: () => jsonRes(adminMe) },
  { match: (u, m) => m === "GET" && u.includes("/auth/info"), respond: () => jsonRes(defaultsResponse) },
  { match: (u, m) => m === "GET" && u.includes("/subnet-ranges"), respond: () => jsonRes([]) },
  { match: (u, m) => m === "GET" && u.includes("/suggest-client-ips"), respond: () => jsonRes(["10.252.1.2/32"]) },
  { match: (u, m) => m === "GET" && u.includes("/clients"), respond: () => jsonRes([]) },
];

describe("ClientsPage — client name auto-fill from email", () => {
  let cleanup: () => void;
  afterEach(() => cleanup?.());

  it("prefills name when email is set first and pattern matches", async () => {
    const user = userEvent.setup();
    cleanup = installFetch([
      settingsRoute({
        client_name_pattern: "^([A-Za-z0-9]+)\\.([A-Za-z0-9]+)@.+$",
        client_name_replacement: "abc-$1$2-def",
      }),
      ...baseRoutes,
    ]);

    renderWithProviders(<ClientsPage />);

    await waitFor(() => {
      expect(screen.getByText("New Client")).toBeInTheDocument();
    });
    await user.click(screen.getByText("New Client"));
    const emailInput = await screen.findByPlaceholderText("john@example.com");

    fireEvent.change(emailInput, { target: { value: "first.last@example.com" } });

    const nameInput = screen.getByPlaceholderText("e.g. John's Laptop") as HTMLInputElement;
    await waitFor(() => {
      expect(nameInput.value).toBe("abc-firstlast-def");
    });
  });

  it("does not prefill name when name is already typed first", async () => {
    const user = userEvent.setup();
    cleanup = installFetch([
      settingsRoute({
        client_name_pattern: "^([A-Za-z0-9]+)\\.([A-Za-z0-9]+)@.+$",
        client_name_replacement: "$1$2",
      }),
      ...baseRoutes,
    ]);

    renderWithProviders(<ClientsPage />);
    await waitFor(() => {
      expect(screen.getByText("New Client")).toBeInTheDocument();
    });
    await user.click(screen.getByText("New Client"));
    const nameInput = (await screen.findByPlaceholderText("e.g. John's Laptop")) as HTMLInputElement;
    fireEvent.change(nameInput, { target: { value: "manual-name" } });

    const emailInput = screen.getByPlaceholderText("john@example.com");
    fireEvent.change(emailInput, { target: { value: "first.last@example.com" } });

    expect(nameInput.value).toBe("manual-name");
  });

  it("does not auto-fill when pattern is empty", async () => {
    const user = userEvent.setup();
    cleanup = installFetch([settingsRoute(), ...baseRoutes]);

    renderWithProviders(<ClientsPage />);
    await waitFor(() => {
      expect(screen.getByText("New Client")).toBeInTheDocument();
    });
    await user.click(screen.getByText("New Client"));
    const emailInput = await screen.findByPlaceholderText("john@example.com");
    fireEvent.change(emailInput, { target: { value: "first.last@example.com" } });

    const nameInput = screen.getByPlaceholderText("e.g. John's Laptop") as HTMLInputElement;
    expect(nameInput.value).toBe("");
  });
});

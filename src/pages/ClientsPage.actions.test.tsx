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
  // Match GET /clients and /clients?...query only — not /clients/{id}/...
  { match: (u, m) => m === "GET" && /\/clients(\?|$)/.test(u), respond: () => jsonRes([]) },
];

function createdClient(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: "newclient",
    name: "Test Laptop",
    email: "alice@example.com",
    enabled: true,
    allocated_ips: ["10.252.1.2/32"],
    allowed_ips: ["0.0.0.0/0"],
    extra_allowed_ips: [],
    subnet_ranges: [],
    endpoint: "",
    additional_notes: "",
    public_key: "pub",
    private_key: "priv",
    preshared_key: "psk",
    use_server_dns: true,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  };
}

// Intercept anchor.click() so we can assert which downloads were triggered.
function captureAnchorClicks() {
  const clicks: Array<{ href: string; download: string }> = [];
  const spy = vi
    .spyOn(HTMLAnchorElement.prototype, "click")
    .mockImplementation(function (this: HTMLAnchorElement) {
      clicks.push({ href: this.href, download: this.download });
    });
  return {
    clicks,
    restore: () => spy.mockRestore(),
  };
}

async function openCreateAndSubmit(
  user: ReturnType<typeof userEvent.setup>,
  name: string,
  email: string,
) {
  await user.click(screen.getByText("New Client"));
  const nameInput = await screen.findByPlaceholderText("e.g. John's Laptop");
  fireEvent.change(nameInput, { target: { value: name } });
  fireEvent.change(screen.getByPlaceholderText("john@example.com"), {
    target: { value: email },
  });
  await user.click(screen.getByRole("button", { name: "Create" }));
}

describe("ClientsPage — post-create dialog button actions", () => {
  let cleanup: () => void;
  // eslint-disable-next-line prefer-const
  let anchors: ReturnType<typeof captureAnchorClicks> | undefined = undefined;
  afterEach(() => {
    cleanup?.();
    anchors?.restore();
  });

  it("Download opens the single-client config URL", async () => {
    const user = userEvent.setup();
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: () => jsonRes(createdClient(), 201),
      },
    ]);
    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Client"));
    await openCreateAndSubmit(user, "Test Laptop", "alice@example.com");

    await waitFor(() => screen.getByText("Client created"));
    await user.click(screen.getByRole("button", { name: /Download/ }));

    expect(openSpy).toHaveBeenCalledTimes(1);
    expect(String(openSpy.mock.calls[0][0])).toContain("/clients/newclient/config");
    openSpy.mockRestore();
  });

  it("Email opens the email dialog pre-filled with the client's email", async () => {
    const user = userEvent.setup();
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: () => jsonRes(createdClient(), 201),
      },
    ]);
    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Client"));
    await openCreateAndSubmit(user, "Test Laptop", "alice@example.com");

    await waitFor(() => screen.getByText("Client created"));
    await user.click(screen.getByRole("button", { name: /Email/ }));

    await waitFor(() => {
      expect(screen.getByText("Send Config via Email")).toBeInTheDocument();
    });
    const recipient = screen.getByPlaceholderText("recipient@example.com") as HTMLInputElement;
    expect(recipient.value).toBe("alice@example.com");
  });

  it("QR code opens the QR dialog for the new client", async () => {
    const user = userEvent.setup();
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: () => jsonRes(createdClient(), 201),
      },
      {
        match: (u, m) => m === "GET" && u.includes("/clients/newclient/qrcode"),
        respond: () => jsonRes({ qr_code: "data:image/png;base64,xyz" }),
      },
    ]);
    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Client"));
    await openCreateAndSubmit(user, "Test Laptop", "alice@example.com");

    await waitFor(() => screen.getByText("Client created"));
    await user.click(screen.getByRole("button", { name: /QR code/ }));

    await waitFor(() => {
      expect(screen.getByText("Test Laptop - QR Code")).toBeInTheDocument();
    });
  });
});

describe("ClientsPage — bulk create derived names + dedup", () => {
  let cleanup: () => void;
  afterEach(() => cleanup?.());

  it("dedupes duplicate emails and uses the pattern for the name", async () => {
    const user = userEvent.setup();
    const posted: Array<{ name: string; email: string }> = [];
    cleanup = installFetch([
      settingsRoute({
        client_name_pattern: "^([A-Za-z0-9]+)\\.([A-Za-z0-9]+)@.+$",
        client_name_replacement: "$1$2",
      }),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: (_u, init) => {
          const body = init?.body ? JSON.parse(String(init.body)) : {};
          posted.push({ name: body.name, email: body.email });
          return jsonRes(createdClient({ id: `id-${posted.length}`, name: body.name, email: body.email }), 201);
        },
      },
    ]);

    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Clients"));
    await user.click(screen.getByText("New Clients"));
    const textarea = await screen.findByLabelText("Emails");
    fireEvent.change(textarea, {
      target: {
        value: "alice.one@x.com\nbob.two@x.com\nalice.one@x.com\n\n",
      },
    });
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => screen.getByText("Import complete"));
    // The duplicate and empty lines must be ignored — only 2 POSTs.
    expect(posted).toHaveLength(2);
    expect(posted[0]).toEqual({ name: "aliceone", email: "alice.one@x.com" });
    expect(posted[1]).toEqual({ name: "bobtwo", email: "bob.two@x.com" });
  });

  it("falls back to the email local-part when no pattern is set", async () => {
    const user = userEvent.setup();
    const posted: Array<{ name: string; email: string }> = [];
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: (_u, init) => {
          const body = init?.body ? JSON.parse(String(init.body)) : {};
          posted.push({ name: body.name, email: body.email });
          return jsonRes(createdClient({ id: `id-${posted.length}`, name: body.name, email: body.email }), 201);
        },
      },
    ]);

    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Clients"));
    await user.click(screen.getByText("New Clients"));
    const textarea = await screen.findByLabelText("Emails");
    fireEvent.change(textarea, {
      target: { value: "first.last+tag@example.com" },
    });
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => screen.getByText("Import complete"));
    expect(posted).toHaveLength(1);
    // Non-alphanumeric characters are stripped from the local part.
    expect(posted[0].name).toBe("firstlasttag");
  });
});

describe("ClientsPage — bulk batch-action buttons", () => {
  let cleanup: () => void;
  let anchors: ReturnType<typeof captureAnchorClicks> | undefined = undefined;
  afterEach(() => {
    cleanup?.();
    anchors?.restore();
  });

  async function createTwoBulkClients(
    user: ReturnType<typeof userEvent.setup>,
    extraRoutes: RouteHandler[] = [],
  ) {
    let postCount = 0;
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: (_u, init) => {
          postCount += 1;
          const body = init?.body ? JSON.parse(String(init.body)) : {};
          return jsonRes(
            createdClient({
              id: `bulk-${postCount}`,
              name: body.name ?? `n${postCount}`,
              email: body.email ?? `e${postCount}@x.com`,
            }),
            201,
          );
        },
      },
      ...extraRoutes,
    ]);
    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Clients"));
    await user.click(screen.getByText("New Clients"));
    const textarea = await screen.findByLabelText("Emails");
    fireEvent.change(textarea, {
      target: { value: "alpha@x.com\nbeta@x.com" },
    });
    await user.click(screen.getByRole("button", { name: "Create" }));
    await waitFor(() => screen.getByText("Import complete"));
  }

  it("downloads all configs as a single zip", async () => {
    const user = userEvent.setup();
    anchors = captureAnchorClicks();
    let bundleURL: string | null = null;
    await createTwoBulkClients(user, [
      {
        match: (u, m) => m === "GET" && u.includes("/clients/bundle/configs.zip"),
        respond: (u) => {
          bundleURL = u;
          // Return a tiny binary blob — the test only cares that the URL
          // is correct and that the download anchor was clicked once.
          return {
            ok: true,
            status: 200,
            json: async () => ({}),
            text: async () => "",
            blob: async () => new Blob(["zip-bytes"], { type: "application/zip" }),
            headers: new Headers({ "Content-Type": "application/zip" }),
          } as Response;
        },
      },
    ]);
    await user.click(screen.getByRole("button", { name: /All configs/ }));

    await waitFor(() => expect(anchors.clicks.length).toBe(1), { timeout: 2000 });
    expect(bundleURL).toContain("ids=bulk-1%2Cbulk-2");
    expect(anchors.clicks[0].download).toBe("wireguard-configs.zip");
  });

  it("downloads all QR codes as a single zip", async () => {
    const user = userEvent.setup();
    anchors = captureAnchorClicks();
    let bundleURL: string | null = null;
    await createTwoBulkClients(user, [
      {
        match: (u, m) => m === "GET" && u.includes("/clients/bundle/qrcodes.zip"),
        respond: (u) => {
          bundleURL = u;
          return {
            ok: true,
            status: 200,
            json: async () => ({}),
            text: async () => "",
            blob: async () => new Blob(["zip-bytes"], { type: "application/zip" }),
            headers: new Headers({ "Content-Type": "application/zip" }),
          } as Response;
        },
      },
    ]);
    await user.click(screen.getByRole("button", { name: /All QR codes/ }));

    await waitFor(() => expect(anchors.clicks.length).toBe(1), { timeout: 2000 });
    expect(bundleURL).toContain("ids=bulk-1%2Cbulk-2");
    expect(anchors.clicks[0].download).toBe("wireguard-qrcodes.zip");
  });

  it("does not trigger a download when the QR codes bundle fails", async () => {
    const user = userEvent.setup();
    anchors = captureAnchorClicks();
    await createTwoBulkClients(user, [
      {
        match: (u, m) => m === "GET" && u.includes("/clients/bundle/qrcodes.zip"),
        respond: () =>
          jsonRes({ error: { code: "INTERNAL", message: "boom" } }, 500),
      },
    ]);
    await user.click(screen.getByRole("button", { name: /All QR codes/ }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /All QR codes/ })).not.toBeDisabled();
    });
    expect(anchors.clicks).toHaveLength(0);
  });

  it("does not trigger a download when the configs bundle fails", async () => {
    const user = userEvent.setup();
    anchors = captureAnchorClicks();
    await createTwoBulkClients(user, [
      {
        match: (u, m) => m === "GET" && u.includes("/clients/bundle/configs.zip"),
        respond: () =>
          jsonRes({ error: { code: "INTERNAL", message: "boom" } }, 500),
      },
    ]);
    await user.click(screen.getByRole("button", { name: /All configs/ }));

    // Wait for the action to finish (button becomes enabled again) and
    // assert no anchor was clicked.
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /All configs/ })).not.toBeDisabled();
    });
    expect(anchors.clicks).toHaveLength(0);
  });

  it("emails each created client", async () => {
    const user = userEvent.setup();
    const emailed: string[] = [];
    await createTwoBulkClients(user, [
      {
        match: (u, m) => m === "POST" && u.includes("/email"),
        respond: (u, init) => {
          const idMatch = u.match(/\/clients\/([^/]+)\/email/);
          const body = init?.body ? JSON.parse(String(init.body)) : {};
          emailed.push(`${idMatch?.[1]}:${body.email}`);
          return jsonRes({ message: "ok" });
        },
      },
    ]);
    await user.click(screen.getByRole("button", { name: /Email each/ }));

    await waitFor(() => expect(emailed.length).toBe(2), { timeout: 2000 });
    expect(emailed).toContain("bulk-1:alpha@x.com");
    expect(emailed).toContain("bulk-2:beta@x.com");
  });
});

describe("ClientsPage — dialog close/cancel behavior", () => {
  let cleanup: () => void;
  afterEach(() => cleanup?.());

  it("Close on the post-create dialog dismisses it", async () => {
    const user = userEvent.setup();
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: () => jsonRes(createdClient(), 201),
      },
    ]);
    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Client"));
    await openCreateAndSubmit(user, "Test Laptop", "alice@example.com");
    await waitFor(() => screen.getByText("Client created"));

    // The Dialog renders a built-in X-close button labeled "Close" plus our
    // explicit ghost "Close" button — both should dismiss. Click the last
    // (visible footer) one.
    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    await user.click(closeButtons[closeButtons.length - 1]);

    await waitFor(() => {
      expect(screen.queryByText("Client created")).not.toBeInTheDocument();
    });
  });

  it("Cancel on the bulk-create dialog dismisses it without making API calls", async () => {
    const user = userEvent.setup();
    let postCount = 0;
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: () => {
          postCount += 1;
          return jsonRes({}, 201);
        },
      },
    ]);

    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Clients"));
    await user.click(screen.getByText("New Clients"));
    const textarea = await screen.findByLabelText("Emails");
    fireEvent.change(textarea, { target: { value: "discarded@x.com" } });

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    await waitFor(() => {
      expect(screen.queryByLabelText("Emails")).not.toBeInTheDocument();
    });
    expect(postCount).toBe(0);
  });

  it("Close on the bulk results dialog dismisses it", async () => {
    const user = userEvent.setup();
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: (_u, init) => {
          const body = init?.body ? JSON.parse(String(init.body)) : {};
          return jsonRes(
            createdClient({ id: "bulk-1", name: body.name, email: body.email }),
            201,
          );
        },
      },
    ]);

    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Clients"));
    await user.click(screen.getByText("New Clients"));
    fireEvent.change(await screen.findByLabelText("Emails"), {
      target: { value: "alpha@x.com" },
    });
    await user.click(screen.getByRole("button", { name: "Create" }));
    await waitFor(() => screen.getByText("Import complete"));

    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    await user.click(closeButtons[closeButtons.length - 1]);

    await waitFor(() => {
      expect(screen.queryByText("Import complete")).not.toBeInTheDocument();
    });
  });
});

describe("ClientsPage — bulk dialog disabled state", () => {
  let cleanup: () => void;
  afterEach(() => cleanup?.());

  it("disables Cancel + Create while a bulk create is in flight", async () => {
    const user = userEvent.setup();
    // The POST handler defers resolution so we can observe the in-flight UI.
    let resolvePost: (() => void) | null = null;
    const postPromise = new Promise<void>((res) => {
      resolvePost = res;
    });
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: async (_u, init) => {
          await postPromise;
          const body = init?.body ? JSON.parse(String(init.body)) : {};
          return jsonRes(
            createdClient({ id: "bulk-1", name: body.name, email: body.email }),
            201,
          );
        },
      },
    ]);

    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Clients"));
    await user.click(screen.getByText("New Clients"));
    fireEvent.change(await screen.findByLabelText("Emails"), {
      target: { value: "alpha@x.com" },
    });
    await user.click(screen.getByRole("button", { name: "Create" }));

    // The Create button now shows the in-flight label and is disabled, and
    // Cancel is disabled too so the user can't drop the request half-way.
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Creating..." })).toBeDisabled();
    });
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();

    // Let the POST resolve so the dialog cleans up.
    resolvePost!();
    await waitFor(() => screen.getByText("Import complete"));
  });

  it("disables the batch-action buttons while one is running", async () => {
    const user = userEvent.setup();
    let resolveEmail: (() => void) | null = null;
    const emailPromise = new Promise<void>((res) => {
      resolveEmail = res;
    });
    cleanup = installFetch([
      settingsRoute(),
      ...baseRoutes,
      {
        match: (u, m) => m === "POST" && u.endsWith("/clients"),
        respond: (_u, init) => {
          const body = init?.body ? JSON.parse(String(init.body)) : {};
          return jsonRes(
            createdClient({ id: "bulk-1", name: body.name, email: body.email }),
            201,
          );
        },
      },
      {
        match: (u, m) => m === "POST" && u.includes("/email"),
        respond: async () => {
          await emailPromise;
          return jsonRes({ message: "ok" });
        },
      },
    ]);

    renderWithProviders(<ClientsPage />);
    await waitFor(() => screen.getByText("New Clients"));
    await user.click(screen.getByText("New Clients"));
    fireEvent.change(await screen.findByLabelText("Emails"), {
      target: { value: "alpha@x.com" },
    });
    await user.click(screen.getByRole("button", { name: "Create" }));
    await waitFor(() => screen.getByText("Import complete"));

    await user.click(screen.getByRole("button", { name: /Email each/ }));

    // While Email each is in flight the other batch buttons must be disabled.
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /All configs/ })).toBeDisabled();
    });
    expect(screen.getByRole("button", { name: /All QR codes/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: /Email each/ })).toBeDisabled();
    // Two buttons have the accessible name "Close": our explicit footer
    // button (rendered first as a child) and the dialog's built-in X icon
    // (rendered after children). The footer button is what we wire to
    // bulkAction state, so check index 0.
    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    expect(closeButtons[0]).toBeDisabled();

    resolveEmail!();
    await waitFor(() => {
      const buttons = screen.getAllByRole("button", { name: "Close" });
      expect(buttons[0]).not.toBeDisabled();
    });
  });
});

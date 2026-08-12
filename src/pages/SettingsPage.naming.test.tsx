import { describe, it, expect, afterEach, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/test-utils";
import { SettingsPage } from "./SettingsPage";

function jsonRes(data: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => data,
    text: async () => JSON.stringify(data),
    headers: new Headers(),
  } as Response;
}

const baseSettings = {
  endpoint_address: "vpn.example.com",
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
  updated_at: "2024-01-01",
};

function install(routes: Array<{ match: (u: string, m: string) => boolean; respond: (u: string, init?: RequestInit) => Response | Promise<Response> }>) {
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

describe("SettingsPage — Naming Patterns", () => {
  let cleanup: () => void;
  afterEach(() => cleanup?.());

  it("renders the four pattern fields", async () => {
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () => jsonRes(baseSettings),
      },
    ]);
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Client Name Pattern")).toBeInTheDocument();
      expect(screen.getByLabelText("Client Name Replacement")).toBeInTheDocument();
      expect(screen.getByLabelText("Email Filename Pattern")).toBeInTheDocument();
      expect(screen.getByLabelText("Email Filename Replacement")).toBeInTheDocument();
    });
  });

  it("displays the documented example in help text", async () => {
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () => jsonRes(baseSettings),
      },
    ]);
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByText(/abc-firstlast-def/)).toBeInTheDocument();
    });
  });

  it("flags an invalid client-name regex", async () => {
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () => jsonRes(baseSettings),
      },
    ]);
    renderWithProviders(<SettingsPage />);

    const input = await screen.findByLabelText("Client Name Pattern");
    fireEvent.change(input, { target: { value: "([unclosed" } });

    await waitFor(() => {
      expect(screen.getByText("Invalid regular expression")).toBeInTheDocument();
    });
  });

  it("starts in regex mode when both fields are empty", async () => {
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () => jsonRes(baseSettings),
      },
    ]);
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByRole("radio", { name: "Regex pattern" })).toBeInTheDocument();
    });
    // Regex inputs are visible
    expect(screen.getByLabelText("Email Filename Pattern")).toBeInTheDocument();
    // Static input is NOT visible
    expect(screen.queryByLabelText("Static Filename")).not.toBeInTheDocument();
  });

  it("starts in static mode when replacement is set but pattern is empty", async () => {
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () =>
          jsonRes({
            ...baseSettings,
            email_filename_pattern: "",
            email_filename_replacement: "company-vpn",
          }),
      },
    ]);
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Static Filename")).toBeInTheDocument();
    });
    const staticInput = screen.getByLabelText("Static Filename") as HTMLInputElement;
    expect(staticInput.value).toBe("company-vpn");
    // Regex inputs are NOT visible
    expect(screen.queryByLabelText("Email Filename Pattern")).not.toBeInTheDocument();
  });

  it("switching from regex to static hides regex inputs and shows static input", async () => {
    const user = userEvent.setup();
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () => jsonRes(baseSettings),
      },
    ]);
    renderWithProviders(<SettingsPage />);

    const staticRadio = await screen.findByRole("radio", { name: "Static name" });
    await user.click(staticRadio);

    await waitFor(() => {
      expect(screen.getByLabelText("Static Filename")).toBeInTheDocument();
    });
    expect(screen.queryByLabelText("Email Filename Pattern")).not.toBeInTheDocument();
  });

  it("static mode saves with an empty pattern and the static name in replacement", async () => {
    const user = userEvent.setup();
    let putBody: unknown = null;
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () => jsonRes(baseSettings),
      },
      {
        match: (u, m) => m === "PUT" && u.endsWith("/settings"),
        respond: (_u, init) => {
          putBody = init?.body ? JSON.parse(String(init.body)) : null;
          return jsonRes({ ...baseSettings });
        },
      },
    ]);
    renderWithProviders(<SettingsPage />);

    const staticRadio = await screen.findByRole("radio", { name: "Static name" });
    await user.click(staticRadio);

    const staticInput = await screen.findByLabelText("Static Filename");
    fireEvent.change(staticInput, { target: { value: "company-vpn" } });

    await user.click(screen.getByText("Save Settings"));

    await waitFor(() => expect(putBody).not.toBeNull());
    const body = putBody as Record<string, string>;
    expect(body.email_filename_pattern).toBe("");
    expect(body.email_filename_replacement).toBe("company-vpn");
  });

  it("switching from static back to regex hides the static input and shows the regex inputs", async () => {
    const user = userEvent.setup();
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () =>
          jsonRes({
            ...baseSettings,
            email_filename_pattern: "",
            email_filename_replacement: "company-vpn",
          }),
      },
    ]);
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Static Filename")).toBeInTheDocument();
    });
    expect(screen.queryByLabelText("Email Filename Pattern")).not.toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "Regex pattern" }));

    await waitFor(() => {
      expect(screen.getByLabelText("Email Filename Pattern")).toBeInTheDocument();
    });
    expect(screen.queryByLabelText("Static Filename")).not.toBeInTheDocument();
  });

  it("flags an invalid email-filename regex (regex mode only)", async () => {
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () => jsonRes(baseSettings),
      },
    ]);
    renderWithProviders(<SettingsPage />);

    const input = await screen.findByLabelText("Email Filename Pattern");
    fireEvent.change(input, { target: { value: "([unclosed" } });

    await waitFor(() => {
      expect(screen.getByText("Invalid regular expression")).toBeInTheDocument();
    });
  });

  it("static mode tolerates pre-existing invalid regex in storage", async () => {
    // If the user has a broken pattern saved, switching to static should let
    // them save again without the regex validation blocking the Save button.
    const user = userEvent.setup();
    let putBody: unknown = null;
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () => jsonRes({ ...baseSettings, email_filename_pattern: "([unclosed" }),
      },
      {
        match: (u, m) => m === "PUT" && u.endsWith("/settings"),
        respond: (_u, init) => {
          putBody = init?.body ? JSON.parse(String(init.body)) : null;
          return jsonRes({ ...baseSettings });
        },
      },
    ]);
    renderWithProviders(<SettingsPage />);

    // In regex mode the bad pattern is flagged.
    await waitFor(() => {
      expect(screen.getByText("Invalid regular expression")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("radio", { name: "Static name" }));
    const staticInput = await screen.findByLabelText("Static Filename");
    fireEvent.change(staticInput, { target: { value: "static-name" } });

    // Save button must be enabled — the regex error should be gone.
    expect(screen.queryByText("Invalid regular expression")).not.toBeInTheDocument();
    await user.click(screen.getByText("Save Settings"));
    await waitFor(() => expect(putBody).not.toBeNull());
    const body = putBody as Record<string, string>;
    expect(body.email_filename_pattern).toBe("");
    expect(body.email_filename_replacement).toBe("static-name");
  });

  it("sends all four pattern fields when saving", async () => {
    const user = userEvent.setup();
    let putBody: unknown = null;
    cleanup = install([
      {
        match: (u, m) => m === "GET" && u.endsWith("/settings"),
        respond: () => jsonRes(baseSettings),
      },
      {
        match: (u, m) => m === "PUT" && u.endsWith("/settings"),
        respond: (_u, init) => {
          putBody = init?.body ? JSON.parse(String(init.body)) : null;
          return jsonRes({ ...baseSettings });
        },
      },
    ]);
    renderWithProviders(<SettingsPage />);

    const pat = await screen.findByLabelText("Client Name Pattern");
    fireEvent.change(pat, { target: { value: "^([A-Za-z0-9]+)\\.([A-Za-z0-9]+)@.+$" } });

    const rep = screen.getByLabelText("Client Name Replacement");
    fireEvent.change(rep, { target: { value: "$1$2" } });

    const fpat = screen.getByLabelText("Email Filename Pattern");
    fireEvent.change(fpat, { target: { value: "^([A-Za-z0-9]+)@.+$" } });

    const frep = screen.getByLabelText("Email Filename Replacement");
    fireEvent.change(frep, { target: { value: "$1" } });

    await user.click(screen.getByText("Save Settings"));

    await waitFor(() => {
      expect(putBody).not.toBeNull();
    });
    const body = putBody as Record<string, string>;
    expect(body.client_name_pattern).toBe("^([A-Za-z0-9]+)\\.([A-Za-z0-9]+)@.+$");
    expect(body.client_name_replacement).toBe("$1$2");
    expect(body.email_filename_pattern).toBe("^([A-Za-z0-9]+)@.+$");
    expect(body.email_filename_replacement).toBe("$1");
  });
});

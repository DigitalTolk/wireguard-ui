import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPut } from "@/lib/api-client";
import { splitList } from "@/lib/utils";
import { isValidIPList, isValidFirewallMark } from "@/lib/validation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Save } from "lucide-react";
import { toast } from "sonner";
import type { GlobalSetting } from "@/lib/types";

function HelpText({ children }: { children: React.ReactNode }) {
  return <p className="text-muted-foreground">{children}</p>;
}

export function SettingsPage() {
  const qc = useQueryClient();
  const { data: settings, isLoading } = useQuery({
    queryKey: ["settings"],
    queryFn: () => apiGet<GlobalSetting>("/settings"),
  });

  const [endpoint, setEndpoint] = useState<string | null>(null);
  const [dns, setDns] = useState<string | null>(null);
  const [mtu, setMtu] = useState<string | null>(null);
  const [keepalive, setKeepalive] = useState<string | null>(null);
  const [fwmark, setFwmark] = useState<string | null>(null);
  const [tbl, setTbl] = useState<string | null>(null);
  const [configPath, setConfigPath] = useState<string | null>(null);

  const endpointVal = endpoint ?? settings?.endpoint_address ?? "";
  const dnsVal = dns ?? settings?.dns_servers?.join(", ") ?? "";
  const mtuVal = mtu ?? String(settings?.mtu ?? "");
  const keepaliveVal =
    keepalive ?? String(settings?.persistent_keepalive ?? "");
  const fwmarkVal = fwmark ?? settings?.firewall_mark ?? "";
  const tblVal = tbl ?? settings?.table ?? "";
  const configPathVal = configPath ?? settings?.config_file_path ?? "";

  const settingsErrors = useMemo(() => {
    const errors: Record<string, string> = {};
    if (!endpointVal.trim()) {
      errors.endpoint = "Endpoint address is required";
    }
    if (!dnsVal.trim()) {
      errors.dns = "At least one DNS server is required";
    } else if (!isValidIPList(dnsVal)) {
      errors.dns = "Each DNS server must be a valid IP address";
    }
    const mtuNum = Number(mtuVal);
    if (mtuVal.trim() === "") {
      errors.mtu = "MTU is required";
    } else if (
      !Number.isInteger(mtuNum) ||
      (mtuNum !== 0 && (mtuNum < 1280 || mtuNum > 9000))
    ) {
      errors.mtu = "MTU must be 0 (to omit) or between 1280 and 9000";
    }
    const kaNum = Number(keepaliveVal);
    if (
      keepaliveVal.trim() !== "" &&
      (!Number.isInteger(kaNum) || kaNum < 0 || kaNum > 65535)
    ) {
      errors.keepalive = "Persistent keepalive must be between 0 and 65535";
    }
    if (fwmarkVal.trim() && !isValidFirewallMark(fwmarkVal)) {
      errors.fwmark = "Must be a hex (0x...) or decimal number";
    }
    if (!configPathVal.trim()) {
      errors.configPath = "Config file path is required";
    } else if (!configPathVal.trim().startsWith("/")) {
      errors.configPath = "Config file path must be an absolute path (start with /)";
    }
    return errors;
  }, [endpointVal, dnsVal, mtuVal, keepaliveVal, fwmarkVal, configPathVal]);
  const settingsValid = Object.keys(settingsErrors).length === 0;

  const saveSettings = useMutation({
    mutationFn: () =>
      apiPut("/settings", {
        endpoint_address: endpointVal,
        dns_servers: splitList(dnsVal),
        mtu: Number(mtuVal) || 0,
        persistent_keepalive: Number(keepaliveVal) || 0,
        firewall_mark: fwmarkVal,
        table: tblVal,
        config_file_path: configPathVal,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["settings"] });
      setEndpoint(null);
      setDns(null);
      setMtu(null);
      setKeepalive(null);
      setFwmark(null);
      setTbl(null);
      setConfigPath(null);
      toast.success("Settings saved");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Global Settings</h2>
        <Button
          onClick={() => saveSettings.mutate()}
          disabled={!settingsValid || saveSettings.isPending}
        >
          <Save className="mr-2 h-4 w-4" />
          {saveSettings.isPending ? "Saving..." : "Save Settings"}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Network</CardTitle>
        </CardHeader>
        <CardContent className="grid items-start gap-6 sm:grid-cols-2">
          <div className="grid gap-2">
            <Label htmlFor="endpoint">Endpoint Address</Label>
            <Input
              id="endpoint"
              placeholder="vpn.example.com or 203.0.113.1"
              value={endpointVal}
              onChange={(e) => setEndpoint(e.target.value)}
            />
            {settingsErrors.endpoint && (
              <p className="text-destructive">{settingsErrors.endpoint}</p>
            )}
            <HelpText>
              Public hostname or IP address that clients connect to. Can include
              a port, e.g. <code>vpn.example.com:51820</code>.
            </HelpText>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="dns">DNS Servers</Label>
            <Input
              id="dns"
              placeholder="1.1.1.1, 8.8.8.8"
              value={dnsVal}
              onChange={(e) => setDns(e.target.value)}
            />
            {settingsErrors.dns && (
              <p className="text-destructive">{settingsErrors.dns}</p>
            )}
            <HelpText>
              Comma-separated list of DNS server IP addresses pushed to clients.
            </HelpText>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="mtu">MTU</Label>
            <Input
              id="mtu"
              type="number"
              placeholder="1450"
              value={mtuVal}
              onChange={(e) => setMtu(e.target.value)}
            />
            {settingsErrors.mtu && (
              <p className="text-destructive">{settingsErrors.mtu}</p>
            )}
            <HelpText>
              Maximum Transmission Unit size in bytes. Typical values are
              1420–1450. Set to 0 to omit from client configs.
            </HelpText>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="keepalive">Persistent Keepalive</Label>
            <Input
              id="keepalive"
              type="number"
              placeholder="15"
              value={keepaliveVal}
              onChange={(e) => setKeepalive(e.target.value)}
            />
            {settingsErrors.keepalive && (
              <p className="text-destructive">{settingsErrors.keepalive}</p>
            )}
            <HelpText>
              Interval in seconds for keepalive packets. Helps maintain
              connections behind NAT. Set to 0 to disable.
            </HelpText>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Advanced</CardTitle>
        </CardHeader>
        <CardContent className="grid items-start gap-6 sm:grid-cols-2">
          <div className="grid gap-2">
            <Label htmlFor="fwmark">Firewall Mark</Label>
            <Input
              id="fwmark"
              placeholder="0xca6c"
              value={fwmarkVal}
              onChange={(e) => setFwmark(e.target.value)}
            />
            {settingsErrors.fwmark && (
              <p className="text-destructive">{settingsErrors.fwmark}</p>
            )}
            <HelpText>
              Hex value used for policy routing. Default{" "}
              <code>0xca6c</code> (51820 in decimal).
            </HelpText>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="table">Routing Table</Label>
            <Input
              id="table"
              placeholder="auto"
              value={tblVal}
              onChange={(e) => setTbl(e.target.value)}
            />
            <HelpText>
              Routing table for WireGuard routes. Use <code>auto</code> for
              automatic, <code>off</code> to disable, or a numeric table ID.
            </HelpText>
          </div>
          <div className="grid gap-2 sm:col-span-2">
            <Label htmlFor="configpath">Config File Path</Label>
            <Input
              id="configpath"
              placeholder="/etc/wireguard/wg0.conf"
              value={configPathVal}
              onChange={(e) => setConfigPath(e.target.value)}
            />
            {settingsErrors.configPath && (
              <p className="text-destructive">{settingsErrors.configPath}</p>
            )}
            <HelpText>
              Absolute path where the generated WireGuard configuration is
              written. The server must have write access to this path.
            </HelpText>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

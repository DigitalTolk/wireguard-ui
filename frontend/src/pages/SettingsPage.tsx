import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPut } from "@/lib/api-client";
import { splitList } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import type { GlobalSetting } from "@/lib/types";

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
  const keepaliveVal = keepalive ?? String(settings?.persistent_keepalive ?? "");
  const fwmarkVal = fwmark ?? settings?.firewall_mark ?? "";
  const tblVal = tbl ?? settings?.table ?? "";
  const configPathVal = configPath ?? settings?.config_file_path ?? "";

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
      <h2 className="text-2xl font-bold tracking-tight">Global Settings</h2>
      <Card>
        <CardHeader>
          <CardTitle>WireGuard Settings</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-5 sm:grid-cols-2">
          <div className="grid gap-2">
            <Label htmlFor="endpoint">Endpoint Address</Label>
            <Input
              id="endpoint"
              value={endpointVal}
              onChange={(e) => setEndpoint(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="dns">DNS Servers</Label>
            <Input
              id="dns"
              value={dnsVal}
              onChange={(e) => setDns(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="mtu">MTU</Label>
            <Input
              id="mtu"
              type="number"
              value={mtuVal}
              onChange={(e) => setMtu(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="keepalive">Persistent Keepalive</Label>
            <Input
              id="keepalive"
              type="number"
              value={keepaliveVal}
              onChange={(e) => setKeepalive(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="fwmark">Firewall Mark</Label>
            <Input
              id="fwmark"
              value={fwmarkVal}
              onChange={(e) => setFwmark(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="table">Table</Label>
            <Input
              id="table"
              value={tblVal}
              onChange={(e) => setTbl(e.target.value)}
            />
          </div>
          <div className="grid gap-2 sm:col-span-2">
            <Label htmlFor="configpath">Config File Path</Label>
            <Input
              id="configpath"
              value={configPathVal}
              onChange={(e) => setConfigPath(e.target.value)}
            />
          </div>
          <div className="sm:col-span-2 flex justify-end">
            <Button
              onClick={() => saveSettings.mutate()}
              disabled={saveSettings.isPending}
            >
              {saveSettings.isPending ? "Saving..." : "Save Settings"}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

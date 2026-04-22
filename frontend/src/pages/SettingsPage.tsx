import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api-client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import type { GlobalSetting } from "@/lib/types";

export function SettingsPage() {
  const { data: settings, isLoading } = useQuery({
    queryKey: ["settings"],
    queryFn: () => apiGet<GlobalSetting>("/settings"),
  });

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold tracking-tight">Global Settings</h2>
      <Card>
        <CardHeader><CardTitle>WireGuard Settings</CardTitle></CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div>
            <Label htmlFor="endpoint">Endpoint Address</Label>
            <Input id="endpoint" defaultValue={settings?.endpoint_address} readOnly />
          </div>
          <div>
            <Label htmlFor="dns">DNS Servers</Label>
            <Input id="dns" defaultValue={settings?.dns_servers?.join(", ")} readOnly />
          </div>
          <div>
            <Label htmlFor="mtu">MTU</Label>
            <Input id="mtu" defaultValue={settings?.mtu} readOnly />
          </div>
          <div>
            <Label htmlFor="keepalive">Persistent Keepalive</Label>
            <Input id="keepalive" defaultValue={settings?.persistent_keepalive} readOnly />
          </div>
          <div>
            <Label htmlFor="fwmark">Firewall Mark</Label>
            <Input id="fwmark" defaultValue={settings?.firewall_mark} readOnly />
          </div>
          <div>
            <Label htmlFor="table">Table</Label>
            <Input id="table" defaultValue={settings?.table} readOnly />
          </div>
          <div className="sm:col-span-2">
            <Label htmlFor="configpath">Config File Path</Label>
            <Input id="configpath" defaultValue={settings?.config_file_path} readOnly />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

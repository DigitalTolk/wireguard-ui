import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api-client";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import type { DeviceStatus } from "@/lib/types";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

function formatHandshake(nanos: number): string {
  if (!nanos || nanos <= 0) return "Never";
  // last_handshake_rel is a Go time.Duration in nanoseconds
  const seconds = Math.floor(nanos / 1_000_000_000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function StatusPage() {
  const { data: devices, isLoading } = useQuery({
    queryKey: ["status"],
    queryFn: () => apiGet<DeviceStatus[]>("/status"),
    refetchInterval: 5000,
  });

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold tracking-tight">Server Status</h2>
      {devices?.map((device) => (
        <Card key={device.name}>
          <CardHeader>
            <CardTitle>{device.name}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>IP</TableHead>
                  <TableHead>Last Handshake</TableHead>
                  <TableHead>Received</TableHead>
                  <TableHead>Sent</TableHead>
                  <TableHead>Endpoint</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {device.peers?.map((peer) => (
                  <TableRow key={peer.public_key}>
                    <TableCell className="font-medium">{peer.name || "Unknown"}</TableCell>
                    <TableCell>
                      <Badge className="w-28 justify-center" variant={peer.connected ? "default" : "secondary"}>
                        {peer.connected ? "Connected" : "Disconnected"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm">
                      {peer.allocated_ip
                        ? peer.allocated_ip.split(", ").map((ip, i) => (
                            <div key={i}>{ip}</div>
                          ))
                        : "-"}
                    </TableCell>
                    <TableCell className="text-sm">{formatHandshake(peer.last_handshake_rel)}</TableCell>
                    <TableCell className="text-sm">{formatBytes(peer.received_bytes)}</TableCell>
                    <TableCell className="text-sm">{formatBytes(peer.transmit_bytes)}</TableCell>
                    <TableCell className="text-sm">{peer.endpoint || "-"}</TableCell>
                  </TableRow>
                ))}
                {(!device.peers || device.peers.length === 0) && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground">
                      No peers connected
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
            </div>
          </CardContent>
        </Card>
      ))}
      {(!devices || devices.length === 0) && (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            No WireGuard interfaces found
          </CardContent>
        </Card>
      )}
    </div>
  );
}

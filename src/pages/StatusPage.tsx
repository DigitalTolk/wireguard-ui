import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api-client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowDown, ArrowUp, ArrowUpDown, CircleCheck, CircleX } from "lucide-react";
import type { DeviceStatus, PeerStatus } from "@/lib/types";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

function formatHandshake(nanos: number): string {
  if (!nanos || nanos <= 0) return "Never";
  const seconds = Math.floor(nanos / 1_000_000_000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

type SortKey =
  | "name"
  | "connected"
  | "last_handshake_rel"
  | "received_bytes"
  | "transmit_bytes"
  | "endpoint";
type SortDir = "asc" | "desc";

function getSortValue(peer: PeerStatus, key: SortKey): string | number | boolean {
  switch (key) {
    case "name":
      return (peer.name || "").toLowerCase();
    case "connected":
      return peer.connected ? 1 : 0;
    case "last_handshake_rel":
      return peer.last_handshake_rel || Number.MAX_SAFE_INTEGER;
    case "received_bytes":
      return peer.received_bytes;
    case "transmit_bytes":
      return peer.transmit_bytes;
    case "endpoint":
      return peer.endpoint || "";
  }
}

function SortIcon({ column, sortKey, sortDir }: { column: SortKey; sortKey: SortKey; sortDir: SortDir }) {
  if (column !== sortKey) return <ArrowUpDown className="ml-1 inline h-3 w-3 opacity-40" />;
  return sortDir === "asc"
    ? <ArrowUp className="ml-1 inline h-3 w-3" />
    : <ArrowDown className="ml-1 inline h-3 w-3" />;
}

export function StatusPage() {
  const { data: devices, isLoading } = useQuery({
    queryKey: ["status"],
    queryFn: () => apiGet<DeviceStatus[]>("/status"),
    refetchInterval: 5000,
  });

  const [sortKey, setSortKey] = useState<SortKey>("connected");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const sortedDevices = useMemo(() => {
    if (!devices) return [];
    return devices.map((device) => ({
      ...device,
      peers: [...(device.peers || [])].sort((a, b) => {
        const va = getSortValue(a, sortKey);
        const vb = getSortValue(b, sortKey);
        const cmp = va < vb ? -1 : va > vb ? 1 : 0;
        const primary = sortDir === "asc" ? cmp : -cmp;
        if (primary !== 0 || sortKey === "name") return primary;
        // secondary sort by name when primary values are equal
        const na = (a.name || "").toLowerCase();
        const nb = (b.name || "").toLowerCase();
        return na < nb ? -1 : na > nb ? 1 : 0;
      }),
    }));
  }, [devices, sortKey, sortDir]);

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  const headerClass = "cursor-pointer select-none hover:text-foreground";

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold tracking-tight">Server Status</h2>
      {sortedDevices.map((device) => (
        <Card key={device.name}>
          <CardHeader>
            <CardTitle>{device.name}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className={headerClass} onClick={() => toggleSort("name")}>
                      Name <SortIcon column="name" sortKey={sortKey} sortDir={sortDir} />
                    </TableHead>
                    <TableHead className={`${headerClass} w-10`} onClick={() => toggleSort("connected")}>
                      <SortIcon column="connected" sortKey={sortKey} sortDir={sortDir} />
                    </TableHead>
                    <TableHead className={headerClass} onClick={() => toggleSort("endpoint")}>
                      Endpoint <SortIcon column="endpoint" sortKey={sortKey} sortDir={sortDir} />
                    </TableHead>
                    <TableHead className={headerClass} onClick={() => toggleSort("last_handshake_rel")}>
                      Handshake <SortIcon column="last_handshake_rel" sortKey={sortKey} sortDir={sortDir} />
                    </TableHead>
                    <TableHead className={headerClass} onClick={() => toggleSort("received_bytes")}>
                      Rx <SortIcon column="received_bytes" sortKey={sortKey} sortDir={sortDir} />
                    </TableHead>
                    <TableHead className={headerClass} onClick={() => toggleSort("transmit_bytes")}>
                      Tx <SortIcon column="transmit_bytes" sortKey={sortKey} sortDir={sortDir} />
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {device.peers?.map((peer) => (
                    <TableRow key={peer.public_key}>
                      <TableCell>
                        <div className="font-medium">{peer.name || "Unknown"}</div>
                        <code className="text-xs text-muted-foreground">
                          {peer.public_key.substring(0, 16)}...
                        </code>
                      </TableCell>
                      <TableCell>
                        <Tooltip>
                          <TooltipTrigger>
                            {peer.connected ? (
                              <CircleCheck className="h-5 w-5 text-green-500" />
                            ) : (
                              <CircleX className="h-5 w-5 text-muted-foreground" />
                            )}
                          </TooltipTrigger>
                          <TooltipContent>
                            {peer.connected ? "Connected" : "Disconnected"}
                          </TooltipContent>
                        </Tooltip>
                      </TableCell>
                      <TableCell className="font-mono">{peer.endpoint || "-"}</TableCell>
                      <TableCell>{formatHandshake(peer.last_handshake_rel)}</TableCell>
                      <TableCell>{formatBytes(peer.received_bytes)}</TableCell>
                      <TableCell>{formatBytes(peer.transmit_bytes)}</TableCell>
                    </TableRow>
                  ))}
                  {(!device.peers || device.peers.length === 0) && (
                    <TableRow>
                      <TableCell
                        colSpan={6}
                        className="text-center text-muted-foreground"
                      >
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

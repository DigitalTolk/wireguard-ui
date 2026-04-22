import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiDelete } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Power, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { WakeOnLanHost } from "@/lib/types";

export function WolPage() {
  const qc = useQueryClient();
  const { data: hosts, isLoading } = useQuery({
    queryKey: ["wol-hosts"],
    queryFn: () => apiGet<WakeOnLanHost[]>("/wol-hosts"),
  });

  const wakeHost = useMutation({
    mutationFn: (mac: string) => apiPost(`/wol-hosts/${encodeURIComponent(mac)}/wake`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wol-hosts"] });
      toast.success("Magic packet sent");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const deleteHost = useMutation({
    mutationFn: (mac: string) => apiDelete(`/wol-hosts/${encodeURIComponent(mac)}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wol-hosts"] });
      toast.success("Host deleted");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold tracking-tight">Wake-on-LAN</h2>
      <Card>
        <CardHeader><CardTitle>Hosts</CardTitle></CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>MAC Address</TableHead>
                <TableHead>Last Used</TableHead>
                <TableHead className="w-24">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {hosts?.map((host) => (
                <TableRow key={host.MacAddress}>
                  <TableCell className="font-medium">{host.Name}</TableCell>
                  <TableCell className="font-mono text-sm">{host.MacAddress}</TableCell>
                  <TableCell className="text-sm">
                    {host.LatestUsed ? new Date(host.LatestUsed).toLocaleString() : "Never"}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => wakeHost.mutate(host.MacAddress)}
                        aria-label={`Wake ${host.Name}`}
                      >
                        <Power className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => {
                          if (confirm(`Delete "${host.Name}"?`)) {
                            deleteHost.mutate(host.MacAddress);
                          }
                        }}
                        aria-label={`Delete ${host.Name}`}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
              {(!hosts || hosts.length === 0) && (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground">
                    No Wake-on-LAN hosts configured
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

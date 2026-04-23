import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiDelete } from "@/lib/api-client";
import { isValidMAC } from "@/lib/validation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Plus, Power, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { WakeOnLanHost } from "@/lib/types";

export function WolPage() {
  const qc = useQueryClient();
  const { data: hosts, isLoading } = useQuery({
    queryKey: ["wol-hosts"],
    queryFn: () => apiGet<WakeOnLanHost[]>("/wol-hosts"),
  });

  const wakeHost = useMutation({
    mutationFn: (mac: string) =>
      apiPost(`/wol-hosts/${encodeURIComponent(mac)}/wake`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wol-hosts"] });
      toast.success("Magic packet sent");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const deleteHost = useMutation({
    mutationFn: (mac: string) =>
      apiDelete(`/wol-hosts/${encodeURIComponent(mac)}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wol-hosts"] });
      toast.success("Host deleted");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const [showCreate, setShowCreate] = useState(false);
  const [newHost, setNewHost] = useState({ name: "", mac: "" });

  const createHost = useMutation({
    mutationFn: (data: { Name: string; MacAddress: string }) =>
      apiPost<WakeOnLanHost>("/wol-hosts", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wol-hosts"] });
      toast.success("Host created");
      setShowCreate(false);
      setNewHost({ name: "", mac: "" });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const wolErrors = useMemo(() => {
    const errors: Record<string, string> = {};
    if (!newHost.name.trim()) {
      errors.name = "Name is required";
    }
    if (!newHost.mac.trim()) {
      errors.mac = "MAC address is required";
    } else if (!isValidMAC(newHost.mac)) {
      errors.mac = "Invalid MAC format (use AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF)";
    }
    return errors;
  }, [newHost]);
  const wolValid = Object.keys(wolErrors).length === 0;

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Wake-on-LAN</h2>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Host
        </Button>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Hosts</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
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
                  <TableCell className="font-mono text-sm">
                    {host.MacAddress}
                  </TableCell>
                  <TableCell className="text-sm">
                    {host.LatestUsed
                      ? new Date(host.LatestUsed).toLocaleString()
                      : "Never"}
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
                  <TableCell
                    colSpan={4}
                    className="text-center text-muted-foreground"
                  >
                    No Wake-on-LAN hosts configured
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          </div>
        </CardContent>
      </Card>

      {/* Create Host Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>New Host</DialogTitle>
          </DialogHeader>
          <div className="grid gap-5 py-4">
            <div className="grid gap-2">
              <Label htmlFor="wol-name">Name</Label>
              <Input
                id="wol-name"
                placeholder="e.g. File Server"
                value={newHost.name}
                onChange={(e) =>
                  setNewHost((p) => ({ ...p, name: e.target.value }))
                }
              />
              {wolErrors.name && (
                <p className="text-destructive">{wolErrors.name}</p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="wol-mac">MAC Address</Label>
              <Input
                id="wol-mac"
                placeholder="AA:BB:CC:DD:EE:FF"
                value={newHost.mac}
                onChange={(e) =>
                  setNewHost((p) => ({ ...p, mac: e.target.value }))
                }
              />
              {wolErrors.mac && (
                <p className="text-destructive">{wolErrors.mac}</p>
              )}
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={() =>
                createHost.mutate({
                  Name: newHost.name,
                  MacAddress: newHost.mac,
                })
              }
              disabled={!wolValid || createHost.isPending}
            >
              {createHost.isPending ? "Creating..." : "Create"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

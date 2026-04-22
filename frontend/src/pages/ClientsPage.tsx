import { useState } from "react";
import { useClients, useSetClientStatus, useDeleteClient } from "@/hooks/useClients";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Download, QrCode, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { ClientData } from "@/lib/types";

export function ClientsPage() {
  const { data: clients, isLoading } = useClients();
  const setStatus = useSetClientStatus();
  const deleteClient = useDeleteClient();
  const [qrDialog, setQrDialog] = useState<ClientData | null>(null);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-24 w-full" />
        ))}
      </div>
    );
  }

  const handleToggle = (id: string, enabled: boolean) => {
    setStatus.mutate(
      { id, enabled },
      {
        onSuccess: () => toast.success(`Client ${enabled ? "enabled" : "disabled"}`),
        onError: (err) => toast.error(err.message),
      }
    );
  };

  const handleDelete = (id: string, name: string) => {
    if (!confirm(`Delete client "${name}"?`)) return;
    deleteClient.mutate(id, {
      onSuccess: () => toast.success("Client deleted"),
      onError: (err) => toast.error(err.message),
    });
  };

  const handleDownload = (id: string) => {
    window.open(`./api/v1/clients/${id}/config`, "_blank");
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">WireGuard Clients</h2>
        <Badge variant="secondary">{clients?.length ?? 0} clients</Badge>
      </div>

      <div className="grid gap-4">
        {clients?.map((cd) => {
          const client = cd.Client;
          return (
            <Card key={client.id}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-base font-medium">
                  {client.name}
                  {client.email && (
                    <span className="ml-2 text-sm font-normal text-muted-foreground">
                      {client.email}
                    </span>
                  )}
                </CardTitle>
                <div className="flex items-center gap-2">
                  <label className="flex items-center gap-2 text-sm" htmlFor={`toggle-${client.id}`}>
                    <Switch
                      id={`toggle-${client.id}`}
                      checked={client.enabled}
                      onCheckedChange={(checked) => handleToggle(client.id, checked)}
                      aria-label={`${client.enabled ? "Disable" : "Enable"} ${client.name}`}
                    />
                    <Badge variant={client.enabled ? "default" : "secondary"}>
                      {client.enabled ? "Enabled" : "Disabled"}
                    </Badge>
                  </label>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <div className="space-y-1 text-sm text-muted-foreground">
                    <div>IPs: {client.allocated_ips?.join(", ") || "None"}</div>
                    {client.additional_notes && (
                      <div>Notes: {client.additional_notes}</div>
                    )}
                  </div>
                  <div className="flex gap-1">
                    {cd.QRCode && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setQrDialog(cd)}
                        aria-label={`Show QR code for ${client.name}`}
                      >
                        <QrCode className="h-4 w-4" />
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDownload(client.id)}
                      aria-label={`Download config for ${client.name}`}
                    >
                      <Download className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDelete(client.id, client.name)}
                      aria-label={`Delete ${client.name}`}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
        {(!clients || clients.length === 0) && (
          <Card>
            <CardContent className="py-8 text-center text-muted-foreground">
              No clients configured yet.
            </CardContent>
          </Card>
        )}
      </div>

      {/* QR Code Dialog */}
      <Dialog open={!!qrDialog} onOpenChange={() => setQrDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{qrDialog?.Client.name} - QR Code</DialogTitle>
          </DialogHeader>
          {qrDialog?.QRCode && (
            <div className="flex justify-center p-4">
              <img
                src={qrDialog.QRCode}
                alt={`QR code for ${qrDialog.Client.name}`}
                className="max-w-[256px]"
              />
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

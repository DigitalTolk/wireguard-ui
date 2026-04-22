import { useState } from "react";
import {
  useClients,
  useCreateClient,
  useUpdateClient,
  useSetClientStatus,
  useDeleteClient,
} from "@/hooks/useClients";
import { apiGet, apiPost, API_BASE } from "@/lib/api-client";
import { splitList } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Download, Mail, Pencil, Plus, QrCode, Send, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { Client, ClientData } from "@/lib/types";

const emptyCreateForm = {
  name: "",
  email: "",
  telegram_userid: "",
  allocated_ips: [] as string[],
  allowed_ips: ["0.0.0.0/0"],
  extra_allowed_ips: [] as string[],
  use_server_dns: true,
  enabled: true,
  additional_notes: "",
};

export function ClientsPage() {
  const { data: clients, isLoading } = useClients();
  const createClient = useCreateClient();
  const updateClient = useUpdateClient();
  const setStatus = useSetClientStatus();
  const deleteClient = useDeleteClient();

  const [qrDialog, setQrDialog] = useState<ClientData | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newClient, setNewClient] = useState({ ...emptyCreateForm });

  const [editDialog, setEditDialog] = useState<Client | null>(null);
  const [editForm, setEditForm] = useState<Partial<Client>>({});

  const [emailDialog, setEmailDialog] = useState<Client | null>(null);
  const [emailAddress, setEmailAddress] = useState("");
  const [emailSending, setEmailSending] = useState(false);

  const [telegramSending, setTelegramSending] = useState<string | null>(null);

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
        onSuccess: () =>
          toast.success(`Client ${enabled ? "enabled" : "disabled"}`),
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
    window.open(`${API_BASE}/clients/${id}/config`, "_blank");
  };

  const handleOpenCreate = () => {
    setShowCreate(true);
    apiGet<string[]>("/suggest-client-ips")
      .then((ips) => setNewClient((prev) => ({ ...prev, allocated_ips: ips })))
      .catch(() => toast.warning("Could not auto-suggest IPs"));
  };

  const handleCreate = () => {
    createClient.mutate(newClient, {
      onSuccess: () => {
        toast.success("Client created");
        setShowCreate(false);
        setNewClient({ ...emptyCreateForm });
      },
      onError: (err) => toast.error(err.message),
    });
  };

  const handleOpenEdit = (client: Client) => {
    setEditForm({
      name: client.name,
      email: client.email,
      telegram_userid: client.telegram_userid,
      allocated_ips: client.allocated_ips || [],
      allowed_ips: client.allowed_ips || [],
      extra_allowed_ips: client.extra_allowed_ips || [],
      endpoint: client.endpoint,
      additional_notes: client.additional_notes,
      use_server_dns: client.use_server_dns,
      preshared_key: client.preshared_key,
    });
    setEditDialog(client);
  };

  const handleSaveEdit = () => {
    if (!editDialog) return;
    updateClient.mutate(
      { id: editDialog.id, ...editForm },
      {
        onSuccess: () => {
          toast.success("Client updated");
          setEditDialog(null);
        },
        onError: (err) => toast.error(err.message),
      }
    );
  };

  const handleOpenEmail = (client: Client) => {
    setEmailAddress(client.email || "");
    setEmailDialog(client);
  };

  const handleSendEmail = async () => {
    if (!emailDialog) return;
    setEmailSending(true);
    try {
      await apiPost(`/clients/${emailDialog.id}/email`, { email: emailAddress });
      toast.success("Email sent");
      setEmailDialog(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to send email");
    } finally {
      setEmailSending(false);
    }
  };

  const handleSendTelegram = async (client: Client) => {
    setTelegramSending(client.id);
    try {
      await apiPost(`/clients/${client.id}/telegram`);
      toast.success("Telegram message sent");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to send Telegram message");
    } finally {
      setTelegramSending(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold tracking-tight">
            WireGuard Clients
          </h2>
          <Badge variant="secondary">{clients?.length ?? 0}</Badge>
        </div>
        <Button onClick={handleOpenCreate}>
          <Plus className="mr-2 h-4 w-4" />
          New Client
        </Button>
      </div>

      <div className="grid gap-4">
        {clients?.map((cd) => {
          const client = cd.Client;
          return (
            <Card key={client.id}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
                <CardTitle className="text-base font-medium">
                  {client.name}
                  {client.email && (
                    <span className="ml-2 text-sm font-normal text-muted-foreground">
                      {client.email}
                    </span>
                  )}
                </CardTitle>
                <div className="flex items-center gap-3">
                  <label
                    className="flex items-center gap-2 text-sm"
                    htmlFor={`toggle-${client.id}`}
                  >
                    <Switch
                      id={`toggle-${client.id}`}
                      checked={client.enabled}
                      onCheckedChange={(checked) =>
                        handleToggle(client.id, checked)
                      }
                      aria-label={`${client.enabled ? "Disable" : "Enable"} ${client.name}`}
                    />
                    <Badge
                      variant={client.enabled ? "default" : "secondary"}
                    >
                      {client.enabled ? "Enabled" : "Disabled"}
                    </Badge>
                  </label>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <div className="space-y-1 text-sm text-muted-foreground">
                    <div>
                      IPs: {client.allocated_ips?.join(", ") || "None"}
                    </div>
                    {client.additional_notes && (
                      <div>Notes: {client.additional_notes}</div>
                    )}
                  </div>
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleOpenEdit(client)}
                      aria-label={`Edit ${client.name}`}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleOpenEmail(client)}
                      aria-label={`Email config to ${client.name}`}
                    >
                      <Mail className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleSendTelegram(client)}
                      disabled={telegramSending === client.id}
                      aria-label={`Send Telegram to ${client.name}`}
                    >
                      <Send className="h-4 w-4" />
                    </Button>
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
            <CardContent className="py-12 text-center text-muted-foreground">
              No clients configured yet. Click "New Client" to add one.
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

      {/* Create Client Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>New Client</DialogTitle>
          </DialogHeader>
          <div className="grid gap-5 py-4">
            <div className="grid gap-2">
              <Label htmlFor="new-name">Name</Label>
              <Input
                id="new-name"
                placeholder="e.g. John's Laptop"
                value={newClient.name}
                onChange={(e) =>
                  setNewClient((p) => ({ ...p, name: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-email">Email</Label>
              <Input
                id="new-email"
                type="email"
                placeholder="john@example.com"
                value={newClient.email}
                onChange={(e) =>
                  setNewClient((p) => ({ ...p, email: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-telegram">Telegram User ID</Label>
              <Input
                id="new-telegram"
                placeholder="123456789"
                value={newClient.telegram_userid}
                onChange={(e) =>
                  setNewClient((p) => ({ ...p, telegram_userid: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-ips">Allocated IPs</Label>
              <Input
                id="new-ips"
                placeholder="10.252.1.2/32"
                value={newClient.allocated_ips.join(", ")}
                onChange={(e) =>
                  setNewClient((p) => ({
                    ...p,
                    allocated_ips: splitList(e.target.value),
                  }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-allowed">Allowed IPs</Label>
              <Input
                id="new-allowed"
                placeholder="0.0.0.0/0"
                value={newClient.allowed_ips.join(", ")}
                onChange={(e) =>
                  setNewClient((p) => ({
                    ...p,
                    allowed_ips: splitList(e.target.value),
                  }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-notes">Notes</Label>
              <Textarea
                id="new-notes"
                placeholder="Optional notes"
                value={newClient.additional_notes}
                onChange={(e) =>
                  setNewClient((p) => ({
                    ...p,
                    additional_notes: e.target.value,
                  }))
                }
              />
            </div>
            <div className="flex items-center gap-3">
              <Switch
                id="new-dns"
                checked={newClient.use_server_dns}
                onCheckedChange={(v) =>
                  setNewClient((p) => ({ ...p, use_server_dns: v }))
                }
              />
              <Label htmlFor="new-dns">Use server DNS</Label>
            </div>
            <div className="flex items-center gap-3">
              <Switch
                id="new-enabled"
                checked={newClient.enabled}
                onCheckedChange={(v) =>
                  setNewClient((p) => ({ ...p, enabled: v }))
                }
              />
              <Label htmlFor="new-enabled">Enable after creation</Label>
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={!newClient.name || createClient.isPending}
            >
              {createClient.isPending ? "Creating..." : "Create"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Edit Client Dialog */}
      <Dialog open={!!editDialog} onOpenChange={() => setEditDialog(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Edit Client</DialogTitle>
          </DialogHeader>
          <div className="grid gap-5 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">Name</Label>
              <Input
                id="edit-name"
                value={editForm.name ?? ""}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, name: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-email">Email</Label>
              <Input
                id="edit-email"
                type="email"
                value={editForm.email ?? ""}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, email: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-telegram">Telegram User ID</Label>
              <Input
                id="edit-telegram"
                value={editForm.telegram_userid ?? ""}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, telegram_userid: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-ips">Allocated IPs</Label>
              <Input
                id="edit-ips"
                value={editForm.allocated_ips?.join(", ") ?? ""}
                onChange={(e) =>
                  setEditForm((p) => ({
                    ...p,
                    allocated_ips: splitList(e.target.value),
                  }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-allowed">Allowed IPs</Label>
              <Input
                id="edit-allowed"
                value={editForm.allowed_ips?.join(", ") ?? ""}
                onChange={(e) =>
                  setEditForm((p) => ({
                    ...p,
                    allowed_ips: splitList(e.target.value),
                  }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-extra-allowed">Extra Allowed IPs</Label>
              <Input
                id="edit-extra-allowed"
                value={editForm.extra_allowed_ips?.join(", ") ?? ""}
                onChange={(e) =>
                  setEditForm((p) => ({
                    ...p,
                    extra_allowed_ips: splitList(e.target.value),
                  }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-endpoint">Endpoint</Label>
              <Input
                id="edit-endpoint"
                value={editForm.endpoint ?? ""}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, endpoint: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-psk">Preshared Key</Label>
              <Input
                id="edit-psk"
                value={editForm.preshared_key ?? ""}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, preshared_key: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-notes">Notes</Label>
              <Textarea
                id="edit-notes"
                value={editForm.additional_notes ?? ""}
                onChange={(e) =>
                  setEditForm((p) => ({
                    ...p,
                    additional_notes: e.target.value,
                  }))
                }
              />
            </div>
            <div className="flex items-center gap-3">
              <Switch
                id="edit-dns"
                checked={editForm.use_server_dns ?? false}
                onCheckedChange={(v) =>
                  setEditForm((p) => ({ ...p, use_server_dns: v }))
                }
              />
              <Label htmlFor="edit-dns">Use server DNS</Label>
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setEditDialog(null)}>
              Cancel
            </Button>
            <Button
              onClick={handleSaveEdit}
              disabled={updateClient.isPending}
            >
              {updateClient.isPending ? "Saving..." : "Save"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Email Dialog */}
      <Dialog open={!!emailDialog} onOpenChange={() => setEmailDialog(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Send Config via Email</DialogTitle>
          </DialogHeader>
          <div className="grid gap-5 py-4">
            <div className="grid gap-2">
              <Label htmlFor="email-to">Email Address</Label>
              <Input
                id="email-to"
                type="email"
                placeholder="recipient@example.com"
                value={emailAddress}
                onChange={(e) => setEmailAddress(e.target.value)}
              />
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setEmailDialog(null)}>
              Cancel
            </Button>
            <Button
              onClick={handleSendEmail}
              disabled={!emailAddress || emailSending}
            >
              {emailSending ? "Sending..." : "Send"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

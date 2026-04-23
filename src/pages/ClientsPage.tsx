import { useState, useMemo, useCallback, useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  useCreateClient,
  useUpdateClient,
  useSetClientStatus,
  useDeleteClient,
} from "@/hooks/useClients";
import { useAuth } from "@/hooks/useAuth";
import { apiGet, apiPost, API_BASE } from "@/lib/api-client";
import { splitList } from "@/lib/utils";
import {
  isValidCIDR,
  isValidEmail,
  isValidEndpoint,
} from "@/lib/validation";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Download, Mail, Pencil, Plus, QrCode, Search, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { Client, ClientData } from "@/lib/types";

interface EditFormState {
  name: string;
  public_key: string;
  allocated_ips: string;
  allowed_ips: string;
  extra_allowed_ips: string;
  endpoint: string;
  additional_notes: string;
  use_server_dns: boolean;
  preshared_key: string;
}

function validateClientForm(form: {
  name: string;
  email?: string;
  allocated_ips: string;
  allowed_ips: string;
  extra_allowed_ips?: string;
  endpoint?: string;
}, emailRequired: boolean): Record<string, string> {
  const errors: Record<string, string> = {};

  if (!form.name.trim()) errors.name = "Name is required";

  if (emailRequired) {
    if (!form.email?.trim()) errors.email = "Email is required";
    else if (!isValidEmail(form.email)) errors.email = "Invalid email format";
  } else if (form.email?.trim() && !isValidEmail(form.email)) {
    errors.email = "Invalid email format";
  }

  const allocIPs = splitList(form.allocated_ips);
  if (allocIPs.length === 0) errors.allocated_ips = "At least one allocated IP is required";
  else if (!allocIPs.every(isValidCIDR)) errors.allocated_ips = "Each IP must be valid CIDR (e.g. 10.0.0.2/32)";

  const allowIPs = splitList(form.allowed_ips);
  if (allowIPs.length === 0) errors.allowed_ips = "At least one allowed IP is required";
  else if (!allowIPs.every(isValidCIDR)) errors.allowed_ips = "Each IP must be valid CIDR (e.g. 0.0.0.0/0)";

  const extraIPs = splitList(form.extra_allowed_ips ?? "");
  if (extraIPs.length > 0 && !extraIPs.every(isValidCIDR)) {
    errors.extra_allowed_ips = "Each IP must be valid CIDR";
  }

  if (form.endpoint?.trim() && !isValidEndpoint(form.endpoint)) {
    errors.endpoint = "Must be host:port or IP:port";
  }

  return errors;
}

function QrCodeDialog({ client, onClose }: { client: ClientData | null; onClose: () => void }) {
  const { data } = useQuery({
    queryKey: ["client-qr", client?.Client.id],
    queryFn: () => apiGet<{ qr_code: string }>(`/clients/${client!.Client.id}/qrcode`),
    enabled: !!client,
    staleTime: Infinity,
  });

  return (
    <Dialog open={!!client} onOpenChange={() => onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{client?.Client.name} - QR Code</DialogTitle>
        </DialogHeader>
        {data?.qr_code ? (
          <div className="flex justify-center p-4">
            <img src={data.qr_code} alt={`QR code for ${client?.Client.name}`} className="max-w-[256px]" />
          </div>
        ) : (
          <div className="flex justify-center p-8">
            <Skeleton className="h-64 w-64" />
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}


const emptyCreateForm = {
  name: "",
  email: "",
  public_key: "",
  preshared_key: "",
  allocated_ips: "",
  allowed_ips: "0.0.0.0/0",
  extra_allowed_ips: "",
  use_server_dns: true,
  additional_notes: "",
};

export function ClientsPage() {
  const { data: me } = useAuth();
  const isAdminUser = me?.admin ?? false;
  const [searchParams, setSearchParams] = useSearchParams();

  const filterSearch = searchParams.get("search") || "";
  const filterStatus = searchParams.get("status") || "";
  const [searchDirty, setSearchDirty] = useState<string | null>(null);
  const searchInput = searchDirty ?? filterSearch;

  const setFilter = useCallback(
    (key: string, value: string) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (value) {
          next.set(key, value);
        } else {
          next.delete(key);
        }
        return next;
      });
    },
    [setSearchParams]
  );

  const buildApiParams = useCallback(() => {
    const params = new URLSearchParams();
    if (filterSearch) params.set("search", filterSearch);
    if (filterStatus) params.set("status", filterStatus);
    return params.toString();
  }, [filterSearch, filterStatus]);

  const { data: clients, isLoading } = useQuery({
    queryKey: ["clients", filterSearch, filterStatus],
    queryFn: () => {
      const qs = buildApiParams();
      return apiGet<ClientData[]>(`/clients${qs ? `?${qs}` : ""}`);
    },
  });

  const createClient = useCreateClient();
  const updateClient = useUpdateClient();
  const setStatus = useSetClientStatus();
  const deleteClient = useDeleteClient();

  const [qrDialog, setQrDialog] = useState<ClientData | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newClient, setNewClient] = useState({ ...emptyCreateForm });
  const [createTouched, setCreateTouched] = useState(false);
  const [subnetRange, setSubnetRange] = useState("");

  const [editDialog, setEditDialog] = useState<Client | null>(null);
  const [editForm, setEditForm] = useState<EditFormState>({
    name: "", public_key: "", allocated_ips: "", allowed_ips: "",
    extra_allowed_ips: "", endpoint: "", additional_notes: "",
    use_server_dns: true, preshared_key: "",
  });

  const [emailDialog, setEmailDialog] = useState<Client | null>(null);
  const [emailAddress, setEmailAddress] = useState("");
  const [emailSending, setEmailSending] = useState(false);

  const [deleteDialog, setDeleteDialog] = useState<{ id: string; name: string } | null>(null);

  const { data: subnetRanges } = useQuery({
    queryKey: ["subnet-ranges"],
    queryFn: () => apiGet<string[]>("/subnet-ranges"),
    staleTime: Infinity,
  });

  // When subnet range changes in create dialog, suggest IPs
  useEffect(() => {
    if (!showCreate) return;
    const sr = subnetRange || "";
    apiGet<string[]>(`/suggest-client-ips${sr ? `?sr=${sr}` : ""}`)
      .then((ips) => setNewClient((prev) => ({ ...prev, allocated_ips: ips.join(", ") })))
      .catch(() => {});
  }, [subnetRange, showCreate]);

  const createErrors = useMemo(
    () => validateClientForm(newClient, true),
    [newClient]
  );
  const createValid = Object.keys(createErrors).length === 0;

  const editErrors = useMemo(
    () =>
      editDialog
        ? validateClientForm({
            name: editForm.name,
            email: editDialog.email,
            allocated_ips: editForm.allocated_ips,
            allowed_ips: editForm.allowed_ips,
            extra_allowed_ips: editForm.extra_allowed_ips,
            endpoint: editForm.endpoint,
          }, true)
        : {},
    [editDialog, editForm]
  );
  const editValid = Object.keys(editErrors).length === 0;

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
    setDeleteDialog({ id, name });
  };

  const handleConfirmDelete = () => {
    if (!deleteDialog) return;
    deleteClient.mutate(deleteDialog.id, {
      onSuccess: () => {
        toast.success("Client deleted");
        setDeleteDialog(null);
      },
      onError: (err) => toast.error(err.message),
    });
  };

  const handleDownload = (id: string) => {
    window.open(`${API_BASE}/clients/${id}/config`, "_blank");
  };

  const handleOpenCreate = () => {
    setNewClient({ ...emptyCreateForm });
    setCreateTouched(false);
    setSubnetRange("");
    setShowCreate(true);
  };

  const handleCreate = () => {
    const payload = {
      ...newClient,
      allocated_ips: splitList(newClient.allocated_ips),
      allowed_ips: splitList(newClient.allowed_ips),
      extra_allowed_ips: splitList(newClient.extra_allowed_ips),
    };
    createClient.mutate(payload, {
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
      public_key: client.public_key,
      allocated_ips: (client.allocated_ips || []).join(", "),
      allowed_ips: (client.allowed_ips || []).join(", "),
      extra_allowed_ips: (client.extra_allowed_ips || []).join(", "),
      endpoint: client.endpoint,
      additional_notes: client.additional_notes,
      use_server_dns: client.use_server_dns,
      preshared_key: client.preshared_key,
    });
    setEditDialog(client);
  };

  const handleSaveEdit = () => {
    if (!editDialog) return;
    const payload = {
      ...editForm,
      allocated_ips: splitList(editForm.allocated_ips),
      allowed_ips: splitList(editForm.allowed_ips),
      extra_allowed_ips: splitList(editForm.extra_allowed_ips),
    };
    updateClient.mutate(
      { id: editDialog.id, ...payload },
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

  const handleExport = () => {
    window.open(`${API_BASE}/clients/export`, "_blank");
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr) return "-";
    try {
      return new Date(dateStr).toLocaleString();
    } catch {
      return "-";
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold tracking-tight">
            WireGuard Clients
          </h2>
          <Badge variant="secondary">{clients?.length ?? 0}</Badge>
        </div>
        {isAdminUser && (
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleExport}>
              <Download className="mr-2 h-4 w-4" />
              Export to Excel
            </Button>
            <Button onClick={handleOpenCreate}>
              <Plus className="mr-2 h-4 w-4" />
              New Client
            </Button>
          </div>
        )}
      </div>

      {/* Filters (admin only) */}
      {isAdminUser && (
        <Card>
          <CardHeader>
            <CardTitle>Filters</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            <div className="grid gap-2">
              <Label htmlFor="filter-search">Search</Label>
              <div className="flex gap-2">
                <Input
                  id="filter-search"
                  className="min-w-0"
                  placeholder="Name, email, or IP..."
                  value={searchInput}
                  onChange={(e) => setSearchDirty(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") { setFilter("search", searchInput); setSearchDirty(null); }
                  }}
                />
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => { setFilter("search", searchInput); setSearchDirty(null); }}
                  aria-label="Search"
                >
                  <Search className="h-4 w-4" />
                </Button>
              </div>
            </div>
            <div className="grid gap-2">
              <Label>Status</Label>
              <Select
                value={filterStatus || undefined}
                onValueChange={(v: string | null) => setFilter("status", !v || v === "_all" ? "" : v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="All" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="_all">All</SelectItem>
                  <SelectItem value="enabled">Enabled</SelectItem>
                  <SelectItem value="disabled">Disabled</SelectItem>
                  <SelectItem value="connected">Connected</SelectItem>
                  <SelectItem value="disconnected">Disconnected</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4">
        {clients?.map((cd) => {
          const client = cd.Client;
          return (
            <Card key={client.id}>
              <CardContent className="px-5 py-3">
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-base font-semibold">{client.name}</span>
                      <Badge variant={client.enabled ? "default" : "secondary"}>
                        {client.enabled ? "Enabled" : "Disabled"}
                      </Badge>
                    </div>
                    <code className="text-xs text-muted-foreground">{client.public_key}</code>
                    <div className="text-muted-foreground">{client.email}</div>
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    {isAdminUser && (
                      <Switch
                        id={`toggle-${client.id}`}
                        checked={client.enabled}
                        onCheckedChange={(checked) => handleToggle(client.id, checked)}
                        aria-label={`${client.enabled ? "Disable" : "Enable"} ${client.name}`}
                      />
                    )}
                  </div>
                </div>

                <div className="mt-4 grid gap-x-8 gap-y-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
                  <div>
                    <div className="text-xs font-medium text-muted-foreground">Allocated IPs</div>
                    <div>{client.allocated_ips?.join(", ") || "—"}</div>
                  </div>
                  <div>
                    <div className="text-xs font-medium text-muted-foreground">Allowed IPs</div>
                    <div>{client.allowed_ips?.join(", ") || "—"}</div>
                  </div>
                  {client.extra_allowed_ips && client.extra_allowed_ips.length > 0 && client.extra_allowed_ips.some(ip => ip) && (
                    <div>
                      <div className="text-xs font-medium text-muted-foreground">Extra Allowed IPs</div>
                      <div>{client.extra_allowed_ips.join(", ")}</div>
                    </div>
                  )}
                  {client.additional_notes && (
                    <div className="sm:col-span-2 lg:col-span-3">
                      <div className="text-xs font-medium text-muted-foreground">Notes</div>
                      <div>{client.additional_notes}</div>
                    </div>
                  )}
                </div>

                <div className="mt-4 flex flex-col gap-3 border-t pt-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex gap-4 text-xs text-muted-foreground">
                    <span>Created {formatDate(client.created_at)}</span>
                    <span>Updated {formatDate(client.updated_at)}</span>
                  </div>
                  <div className="flex flex-wrap gap-1">
                    {isAdminUser && (
                      <Button variant="ghost" size="icon" onClick={() => handleOpenEdit(client)} aria-label={`Edit ${client.name}`}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                    )}
                    {isAdminUser && (
                      <Button variant="ghost" size="icon" onClick={() => handleOpenEmail(client)} aria-label={`Email config to ${client.name}`}>
                        <Mail className="h-4 w-4" />
                      </Button>
                    )}
                    <Button variant="ghost" size="icon" onClick={() => setQrDialog(cd)} aria-label={`Show QR code for ${client.name}`}>
                      <QrCode className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => handleDownload(client.id)} aria-label={`Download config for ${client.name}`}>
                      <Download className="h-4 w-4" />
                    </Button>
                    {isAdminUser && (
                      <Button variant="ghost" size="icon" onClick={() => handleDelete(client.id, client.name)} aria-label={`Delete ${client.name}`}>
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
        {(!clients || clients.length === 0) && (
          <Card>
            <CardContent className="py-12 text-center text-muted-foreground">
              {isAdminUser
                ? 'No clients configured yet. Click "New Client" to add one.'
                : "No client configurations found for your account."}
            </CardContent>
          </Card>
        )}
      </div>

      {/* QR Code Dialog */}
      <QrCodeDialog client={qrDialog} onClose={() => setQrDialog(null)} />

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteDialog} onOpenChange={() => setDeleteDialog(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Delete Client</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete <strong>{deleteDialog?.name}</strong>? This action cannot be undone. The client will lose access to the VPN immediately after applying the configuration.
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setDeleteDialog(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={deleteClient.isPending}
            >
              {deleteClient.isPending ? "Deleting..." : "Delete"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Create Client Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>New Client</DialogTitle>
          </DialogHeader>
          <div className="grid items-start gap-5 py-4 sm:grid-cols-2" onChange={() => setCreateTouched(true)}>
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
              {createTouched && createErrors.name && (
                <p className="text-destructive">{createErrors.name}</p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-email">Email *</Label>
              <Input
                id="new-email"
                type="email"
                placeholder="john@example.com"
                value={newClient.email}
                onChange={(e) =>
                  setNewClient((p) => ({ ...p, email: e.target.value }))
                }
              />
              {createTouched && createErrors.email && (
                <p className="text-destructive">{createErrors.email}</p>
              )}
            </div>
            {subnetRanges && subnetRanges.length > 0 && (
              <div className="grid gap-2">
                <Label>Subnet Range</Label>
                <Select
                  value={subnetRange || undefined}
                  onValueChange={(v: string | null) => setSubnetRange(!v || v === "_default" ? "" : v)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Default" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="_default">Default</SelectItem>
                    {subnetRanges.map((sr) => (
                      <SelectItem key={sr} value={sr}>
                        {sr}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
            <div className="grid gap-2">
              <Label htmlFor="new-ips">Allocated IPs</Label>
              <Input
                id="new-ips"
                placeholder="e.g. 10.0.0.2/32, 10.0.0.3/32"
                value={newClient.allocated_ips}
                onChange={(e) =>
                  setNewClient((p) => ({
                    ...p,
                    allocated_ips: e.target.value,
                  }))
                }
              />
              {createTouched && createErrors.allocated_ips && (
                <p className="text-destructive">{createErrors.allocated_ips}</p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-allowed">Allowed IPs</Label>
              <Input
                id="new-allowed"
                placeholder="e.g. 10.0.0.2/32, 10.0.0.3/32"
                value={newClient.allowed_ips}
                onChange={(e) =>
                  setNewClient((p) => ({
                    ...p,
                    allowed_ips: e.target.value,
                  }))
                }
              />
              {createTouched && createErrors.allowed_ips && (
                <p className="text-destructive">{createErrors.allowed_ips}</p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-extra-allowed">Extra Allowed IPs</Label>
              <Input
                id="new-extra-allowed"
                placeholder="e.g. 10.0.0.2/32, 10.0.0.3/32"
                value={newClient.extra_allowed_ips}
                onChange={(e) =>
                  setNewClient((p) => ({
                    ...p,
                    extra_allowed_ips: e.target.value,
                  }))
                }
              />
              {createTouched && createErrors.extra_allowed_ips && (
                <p className="text-destructive">{createErrors.extra_allowed_ips}</p>
              )}
            </div>
            <div className="grid gap-2 sm:col-span-2">
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
            <div className="grid gap-2">
              <Label htmlFor="new-pubkey">Public Key</Label>
              <Input
                id="new-pubkey"
                placeholder="Leave blank to auto-generate"
                value={newClient.public_key}
                onChange={(e) =>
                  setNewClient((p) => ({ ...p, public_key: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-psk">Preshared Key</Label>
              <Input
                id="new-psk"
                placeholder="Leave blank to auto-generate, enter - to skip"
                value={newClient.preshared_key}
                onChange={(e) =>
                  setNewClient((p) => ({ ...p, preshared_key: e.target.value }))
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
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={!createValid || createClient.isPending}
            >
              {createClient.isPending ? "Creating..." : "Create"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Edit Client Dialog */}
      <Dialog open={!!editDialog} onOpenChange={() => setEditDialog(null)}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>Edit Client</DialogTitle>
          </DialogHeader>
          <div className="grid items-start gap-5 py-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">Name</Label>
              <Input
                id="edit-name"
                value={editForm.name}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, name: e.target.value }))
                }
              />
              {editErrors.name && (
                <p className="text-destructive">{editErrors.name}</p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-email">Email</Label>
              <Input
                id="edit-email"
                type="email"
                value={editDialog?.email ?? ""}
                disabled
                className="opacity-60"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-ips">Allocated IPs</Label>
              <Input
                id="edit-ips"
                placeholder="e.g. 10.0.0.2/32, 10.0.0.3/32"
                value={editForm.allocated_ips}
                onChange={(e) =>
                  setEditForm((p) => ({
                    ...p,
                    allocated_ips: e.target.value,
                  }))
                }
              />
              {editErrors.allocated_ips && (
                <p className="text-destructive">{editErrors.allocated_ips}</p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-allowed">Allowed IPs</Label>
              <Input
                id="edit-allowed"
                placeholder="e.g. 10.0.0.2/32, 10.0.0.3/32"
                value={editForm.allowed_ips}
                onChange={(e) =>
                  setEditForm((p) => ({
                    ...p,
                    allowed_ips: e.target.value,
                  }))
                }
              />
              {editErrors.allowed_ips && (
                <p className="text-destructive">{editErrors.allowed_ips}</p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-extra-allowed">Extra Allowed IPs</Label>
              <Input
                id="edit-extra-allowed"
                placeholder="e.g. 10.0.0.2/32, 10.0.0.3/32"
                value={editForm.extra_allowed_ips}
                onChange={(e) =>
                  setEditForm((p) => ({
                    ...p,
                    extra_allowed_ips: e.target.value,
                  }))
                }
              />
              {editErrors.extra_allowed_ips && (
                <p className="text-destructive">
                  {editErrors.extra_allowed_ips}
                </p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-endpoint">Endpoint</Label>
              <Input
                id="edit-endpoint"
                value={editForm.endpoint}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, endpoint: e.target.value }))
                }
              />
              {editErrors.endpoint && (
                <p className="text-destructive">{editErrors.endpoint}</p>
              )}
            </div>
            <div className="grid gap-2 sm:col-span-2">
              <Label htmlFor="edit-notes">Notes</Label>
              <Textarea
                id="edit-notes"
                value={editForm.additional_notes}
                onChange={(e) =>
                  setEditForm((p) => ({
                    ...p,
                    additional_notes: e.target.value,
                  }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-pubkey">Public Key</Label>
              <Input
                id="edit-pubkey"
                className="font-mono text-xs"
                value={editForm.public_key}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, public_key: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-psk">Preshared Key</Label>
              <Input
                id="edit-psk"
                className="font-mono text-xs"
                value={editForm.preshared_key}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, preshared_key: e.target.value }))
                }
              />
            </div>
            <div className="flex items-center gap-3 sm:col-span-2">
              <Switch
                id="edit-dns"
                checked={editForm.use_server_dns}
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
              disabled={!editValid || updateClient.isPending}
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

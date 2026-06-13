import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiDelete } from "@/lib/api-client";
import type { APIToken, CreateAPITokenResponse } from "@/lib/types";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Copy, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

function fmtTime(iso?: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString();
}

export function APITokensCard() {
  const qc = useQueryClient();
  const { data: tokens, isLoading } = useQuery({
    queryKey: ["api-tokens"],
    queryFn: () => apiGet<APIToken[]>("/api-tokens"),
  });

  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [plaintext, setPlaintext] = useState<string | null>(null);
  // confirmRevoke holds the token currently being asked-about so we never
  // delete by mistake — revoke is irreversible from the API's perspective.
  const [confirmRevoke, setConfirmRevoke] = useState<APIToken | null>(null);

  const createMutation = useMutation({
    mutationFn: (n: string) =>
      apiPost<CreateAPITokenResponse>("/api-tokens", { name: n }),
    onSuccess: (resp) => {
      setPlaintext(resp.token);
      setShowCreate(false);
      setName("");
      qc.invalidateQueries({ queryKey: ["api-tokens"] });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const revokeMutation = useMutation({
    mutationFn: (id: string) => apiDelete(`/api-tokens/${id}`),
    onSuccess: () => {
      setConfirmRevoke(null);
      qc.invalidateQueries({ queryKey: ["api-tokens"] });
      toast.success("Token revoked");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const copyPlaintext = async () => {
    if (!plaintext) return;
    try {
      await navigator.clipboard.writeText(plaintext);
      toast.success("Token copied to clipboard");
    } catch {
      toast.error("Failed to copy — select and copy manually");
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>API Tokens</CardTitle>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Token
        </Button>
      </CardHeader>
      <CardContent>
        <p className="text-muted-foreground mb-4">
          Tokens are admin-level credentials used by the programmatic API
          (provision-client and delete-by-email). Use{" "}
          <code>Authorization: Bearer &lt;token&gt;</code>. The plaintext is
          shown <strong>once</strong> at creation — store it somewhere safe.
        </p>
        {isLoading ? (
          <p className="text-muted-foreground">Loading tokens…</p>
        ) : !tokens || tokens.length === 0 ? (
          <p className="text-muted-foreground">No tokens yet.</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Last used</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tokens.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>{t.name}</TableCell>
                  <TableCell>{fmtTime(t.created_at)}</TableCell>
                  <TableCell>{fmtTime(t.last_used_at)}</TableCell>
                  <TableCell>
                    {t.revoked_at ? (
                      <Badge variant="secondary">Revoked</Badge>
                    ) : (
                      <Badge>Active</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    {!t.revoked_at && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setConfirmRevoke(t)}
                        aria-label={`Revoke ${t.name}`}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>

      {/* Create dialog */}
      <Dialog open={showCreate} onOpenChange={(o) => !createMutation.isPending && setShowCreate(o)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New API Token</DialogTitle>
            <DialogDescription>
              Give the token a short, descriptive name so you can recognize it
              later (e.g. <code>deploy-bot</code>, <code>ci-runner</code>).
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-2 py-2">
            <Label htmlFor="token-name">Name</Label>
            <Input
              id="token-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="deploy-bot"
              disabled={createMutation.isPending}
            />
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setShowCreate(false)} disabled={createMutation.isPending}>
              Cancel
            </Button>
            <Button
              onClick={() => createMutation.mutate(name.trim())}
              disabled={!name.trim() || createMutation.isPending}
            >
              {createMutation.isPending ? "Creating…" : "Create"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Plaintext-once dialog */}
      <Dialog
        open={plaintext !== null}
        onOpenChange={(o) => {
          if (!o) setPlaintext(null);
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Token created</DialogTitle>
            <DialogDescription>
              Copy this token now — it will <strong>not</strong> be shown
              again. If you lose it, revoke the entry and create a new one.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-2 py-2">
            <Label htmlFor="plaintext-token">Token</Label>
            <div className="flex gap-2">
              <Input
                id="plaintext-token"
                readOnly
                value={plaintext ?? ""}
                className="font-mono text-xs"
                onFocus={(e) => e.currentTarget.select()}
              />
              <Button variant="outline" size="icon" onClick={copyPlaintext} aria-label="Copy token">
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <div className="flex justify-end">
            <Button onClick={() => setPlaintext(null)}>I&apos;ve saved it</Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Revoke confirmation */}
      <Dialog
        open={confirmRevoke !== null}
        onOpenChange={(o) => {
          if (!o && !revokeMutation.isPending) setConfirmRevoke(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke token</DialogTitle>
            <DialogDescription>
              Revoke <strong>{confirmRevoke?.name}</strong>? Any client using
              it will start receiving 401 responses immediately.
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setConfirmRevoke(null)} disabled={revokeMutation.isPending}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => confirmRevoke && revokeMutation.mutate(confirmRevoke.id)}
              disabled={revokeMutation.isPending}
            >
              {revokeMutation.isPending ? "Revoking…" : "Revoke"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

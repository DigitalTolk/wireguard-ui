import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut } from "@/lib/api-client";
import { splitList } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "sonner";
import type { Server } from "@/lib/types";

export function ServerPage() {
  const qc = useQueryClient();
  const { data: server, isLoading } = useQuery({
    queryKey: ["server"],
    queryFn: () => apiGet<Server>("/server"),
  });

  const iface = server?.Interface;
  const [addresses, setAddresses] = useState<string | null>(null);
  const [listenPort, setListenPort] = useState<string | null>(null);
  const [postUp, setPostUp] = useState<string | null>(null);
  const [preDown, setPreDown] = useState<string | null>(null);
  const [postDown, setPostDown] = useState<string | null>(null);

  const addrValue = addresses ?? iface?.addresses?.join(", ") ?? "";
  const portValue = listenPort ?? String(iface?.listen_port ?? "");
  const postUpValue = postUp ?? iface?.post_up ?? "";
  const preDownValue = preDown ?? iface?.pre_down ?? "";
  const postDownValue = postDown ?? iface?.post_down ?? "";

  const saveInterface = useMutation({
    mutationFn: () =>
      apiPut("/server/interface", {
        addresses: splitList(addrValue),
        listen_port: Number(portValue) || 0,
        post_up: postUpValue,
        pre_down: preDownValue,
        post_down: postDownValue,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["server"] });
      setAddresses(null);
      setListenPort(null);
      setPostUp(null);
      setPreDown(null);
      setPostDown(null);
      toast.success("Interface settings saved");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const regenerateKeypair = useMutation({
    mutationFn: () => apiPost("/server/keypair"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["server"] });
      toast.success("Keypair regenerated");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const applyConfig = useMutation({
    mutationFn: () => apiPost("/server/apply-config"),
    onSuccess: () => toast.success("Config applied"),
    onError: (err: Error) => toast.error(err.message),
  });

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Server Configuration</h2>
        <Button onClick={() => applyConfig.mutate()} disabled={applyConfig.isPending}>
          {applyConfig.isPending ? "Applying..." : "Apply Config"}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Interface</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-5">
          <div className="grid gap-2">
            <Label htmlFor="srv-addresses">Addresses</Label>
            <Input
              id="srv-addresses"
              value={addrValue}
              onChange={(e) => setAddresses(e.target.value)}
              placeholder="10.252.1.0/24"
              aria-label="Server addresses"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="srv-port">Listen Port</Label>
            <Input
              id="srv-port"
              type="number"
              value={portValue}
              onChange={(e) => setListenPort(e.target.value)}
              placeholder="51820"
              aria-label="Listen port"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="srv-postup">Post-Up Script</Label>
            <Textarea
              id="srv-postup"
              value={postUpValue}
              onChange={(e) => setPostUp(e.target.value)}
              placeholder="iptables -A FORWARD ..."
              rows={3}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="srv-predown">Pre-Down Script</Label>
            <Textarea
              id="srv-predown"
              value={preDownValue}
              onChange={(e) => setPreDown(e.target.value)}
              placeholder="Optional pre-down script"
              rows={3}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="srv-postdown">Post-Down Script</Label>
            <Textarea
              id="srv-postdown"
              value={postDownValue}
              onChange={(e) => setPostDown(e.target.value)}
              placeholder="iptables -D FORWARD ..."
              rows={3}
            />
          </div>
          <div className="flex justify-end">
            <Button
              onClick={() => saveInterface.mutate()}
              disabled={saveInterface.isPending}
            >
              {saveInterface.isPending ? "Saving..." : "Save Interface"}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Keypair</CardTitle>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                if (confirm("Regenerate server keypair? All clients will need to be updated.")) {
                  regenerateKeypair.mutate();
                }
              }}
              disabled={regenerateKeypair.isPending}
            >
              Regenerate
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-2">
            <Label>Public Key</Label>
            <Input value={server?.KeyPair?.public_key || ""} readOnly aria-label="Server public key" />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

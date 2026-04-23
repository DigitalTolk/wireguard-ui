import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut } from "@/lib/api-client";
import { splitList } from "@/lib/utils";
import { isValidCIDRList, isValidPort } from "@/lib/validation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { Save } from "lucide-react";
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

  const serverErrors = useMemo(() => {
    const errors: Record<string, string> = {};
    if (!addrValue.trim()) {
      errors.addresses = "At least one address is required";
    } else if (!isValidCIDRList(addrValue)) {
      errors.addresses = "Each address must be valid CIDR (e.g. 10.252.1.0/24)";
    }
    const portNum = Number(portValue);
    if (!portValue.trim()) {
      errors.port = "Listen port is required";
    } else if (!isValidPort(portNum)) {
      errors.port = "Port must be between 1 and 65535";
    }
    return errors;
  }, [addrValue, portValue]);
  const serverValid = Object.keys(serverErrors).length === 0;

  const saveAndApply = useMutation({
    mutationFn: async () => {
      await apiPut("/server/interface", {
        addresses: splitList(addrValue),
        listen_port: Number(portValue) || 0,
        post_up: postUpValue,
        pre_down: preDownValue,
        post_down: postDownValue,
      });
      await apiPost("/server/apply-config");
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["server"] });
      setAddresses(null);
      setListenPort(null);
      setPostUp(null);
      setPreDown(null);
      setPostDown(null);
      toast.success("Interface saved and config applied");
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

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-2xl font-bold tracking-tight">
          Server Configuration
        </h2>
        <Button
          onClick={() => saveAndApply.mutate()}
          disabled={!serverValid || saveAndApply.isPending}
        >
          <Save className="mr-2 h-4 w-4" />
          {saveAndApply.isPending ? "Applying..." : "Apply Config"}
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
            {serverErrors.addresses && (
              <p className="text-destructive">{serverErrors.addresses}</p>
            )}
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
            {serverErrors.port && (
              <p className="text-destructive">{serverErrors.port}</p>
            )}
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
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Keypair</CardTitle>
            <Button
              variant="outline"
              onClick={() => {
                if (
                  confirm(
                    "Regenerate server keypair? All clients will need to be updated."
                  )
                ) {
                  regenerateKeypair.mutate();
                }
              }}
              disabled={regenerateKeypair.isPending}
            >
              {regenerateKeypair.isPending ? "Regenerating..." : "Regenerate"}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-2">
            <Label>Public Key</Label>
            <Input
              value={server?.KeyPair?.public_key || ""}
              readOnly
              aria-label="Server public key"
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

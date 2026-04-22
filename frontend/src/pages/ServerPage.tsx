import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import type { Server } from "@/lib/types";

export function ServerPage() {
  const qc = useQueryClient();
  const { data: server, isLoading } = useQuery({
    queryKey: ["server"],
    queryFn: () => apiGet<Server>("/server"),
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
        <CardContent className="space-y-4">
          <div>
            <Label>Addresses</Label>
            <Input
              defaultValue={server?.Interface?.addresses?.join(", ")}
              disabled
              aria-label="Server addresses"
            />
          </div>
          <div>
            <Label>Listen Port</Label>
            <Input
              defaultValue={server?.Interface?.listen_port}
              disabled
              aria-label="Listen port"
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
        <CardContent className="space-y-4">
          <div>
            <Label>Public Key</Label>
            <Input value={server?.KeyPair?.public_key || ""} readOnly aria-label="Server public key" />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

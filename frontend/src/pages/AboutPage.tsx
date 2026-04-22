import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function AboutPage() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold tracking-tight">About</h2>
      <Card>
        <CardHeader><CardTitle>WireGuard UI</CardTitle></CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <p>A web interface to manage your WireGuard setup.</p>
          <p>
            Based on{" "}
            <a
              href="https://github.com/ngoduykhanh/wireguard-ui"
              className="text-primary underline"
              target="_blank"
              rel="noopener noreferrer"
            >
              wireguard-ui
            </a>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function AuditPage() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold tracking-tight">Audit Logs</h2>
      <Card>
        <CardHeader><CardTitle>Activity Log</CardTitle></CardHeader>
        <CardContent className="py-8 text-center text-muted-foreground">
          Audit logs will be available once the audit system is enabled.
        </CardContent>
      </Card>
    </div>
  );
}

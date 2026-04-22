import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet, API_BASE } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ChevronLeft, ChevronRight, Download } from "lucide-react";
import type { AuditLog } from "@/lib/types";

interface AuditLogResponse {
  data: AuditLog[];
  total: number;
  page: number;
  per_page: number;
}

export function AuditPage() {
  const [page, setPage] = useState(1);
  const perPage = 50;

  const [filterFrom, setFilterFrom] = useState("");
  const [filterTo, setFilterTo] = useState("");
  const [filterActor, setFilterActor] = useState("");
  const [filterAction, setFilterAction] = useState("");

  const buildFilterParams = () => {
    const params = new URLSearchParams();
    if (filterFrom) params.set("from", filterFrom);
    if (filterTo) params.set("to", filterTo);
    if (filterActor) params.set("actor", filterActor);
    if (filterAction) params.set("action", filterAction);
    return params;
  };

  const buildQueryString = (p: number) => {
    const params = buildFilterParams();
    params.set("page", String(p));
    params.set("per_page", String(perPage));
    return params.toString();
  };

  const { data, isLoading } = useQuery({
    queryKey: [
      "audit-logs",
      page,
      filterFrom,
      filterTo,
      filterActor,
      filterAction,
    ],
    queryFn: () =>
      apiGet<AuditLogResponse>(`/audit-logs?${buildQueryString(page)}`),
  });

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

  const handleExport = () => {
    const qs = buildFilterParams().toString();
    window.open(`${API_BASE}/audit-logs/export${qs ? `?${qs}` : ""}`, "_blank");
  };

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Audit Logs</h2>
        <Button variant="outline" onClick={handleExport}>
          <Download className="mr-2 h-4 w-4" />
          Export to Excel
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Filters</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          <div className="grid gap-2">
            <Label htmlFor="filter-from">Date From</Label>
            <Input
              id="filter-from"
              type="date"
              value={filterFrom}
              onChange={(e) => {
                setFilterFrom(e.target.value);
                setPage(1);
              }}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="filter-to">Date To</Label>
            <Input
              id="filter-to"
              type="date"
              value={filterTo}
              onChange={(e) => {
                setFilterTo(e.target.value);
                setPage(1);
              }}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="filter-actor">Actor</Label>
            <Input
              id="filter-actor"
              placeholder="e.g. admin"
              value={filterActor}
              onChange={(e) => {
                setFilterActor(e.target.value);
                setPage(1);
              }}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="filter-action">Action</Label>
            <Input
              id="filter-action"
              placeholder="e.g. create"
              value={filterAction}
              onChange={(e) => {
                setFilterAction(e.target.value);
                setPage(1);
              }}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Activity Log</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Timestamp</TableHead>
                <TableHead>Actor</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Resource Type</TableHead>
                <TableHead>Resource ID</TableHead>
                <TableHead>IP Address</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data?.data?.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="text-sm whitespace-nowrap">
                    {new Date(log.timestamp).toLocaleString()}
                  </TableCell>
                  <TableCell>{log.actor}</TableCell>
                  <TableCell>{log.action}</TableCell>
                  <TableCell>{log.resource_type}</TableCell>
                  <TableCell className="font-mono text-sm">
                    {log.resource_id}
                  </TableCell>
                  <TableCell className="font-mono text-sm">
                    {log.ip_address}
                  </TableCell>
                </TableRow>
              ))}
              {(!data?.data || data.data.length === 0) && (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="text-center text-muted-foreground"
                  >
                    No audit logs found
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>

          <div className="flex items-center justify-between pt-4">
            <div className="text-sm text-muted-foreground">
              Page {data?.page ?? 1} of {totalPages} ({data?.total ?? 0} total)
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                <ChevronLeft className="mr-1 h-4 w-4" />
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
                <ChevronRight className="ml-1 h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

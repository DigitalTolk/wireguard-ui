import { useCallback, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { apiGet, API_BASE } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ChevronLeft, ChevronRight, Download, Search } from "lucide-react";
import type { AuditLog } from "@/lib/types";

interface AuditLogResponse {
  data: AuditLog[];
  total: number;
  page: number;
  per_page: number;
}

interface AuditFiltersResponse {
  actors: string[];
  actions: string[];
}

function parseDetails(details: string): Record<string, string> {
  try {
    return JSON.parse(details);
  } catch {
    return {};
  }
}

function formatResource(log: AuditLog): string {
  const d = parseDetails(log.details);
  const name = d.name || d.email || "";
  const email = d.email || "";
  if (name && email && name !== email) {
    return `${name} <${email}> (${log.resource_id})`;
  }
  if (name) {
    return `${name} (${log.resource_id})`;
  }
  return log.resource_id;
}

const PER_PAGE = 50;

export function AuditPage() {
  const [searchParams, setSearchParams] = useSearchParams();

  const page = parseInt(searchParams.get("page") || "1", 10) || 1;
  const filterFrom = searchParams.get("from") || "";
  const filterTo = searchParams.get("to") || "";
  const filterActor = searchParams.get("actor") || "";
  const filterAction = searchParams.get("action") || "";
  const filterSearch = searchParams.get("search") || "";
  const [searchInput, setSearchInput] = useState(filterSearch);

  const setFilter = useCallback(
    (key: string, value: string) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (value) {
          next.set(key, value);
        } else {
          next.delete(key);
        }
        // reset to page 1 when filters change (unless changing page itself)
        if (key !== "page") {
          next.delete("page");
        }
        return next;
      });
    },
    [setSearchParams]
  );

  const setPage = useCallback(
    (p: number) => setFilter("page", p > 1 ? String(p) : ""),
    [setFilter]
  );

  const { data: filters } = useQuery({
    queryKey: ["audit-filters"],
    queryFn: () => apiGet<AuditFiltersResponse>("/audit-logs/filters"),
    staleTime: 0,
    refetchOnWindowFocus: true,
  });

  const buildApiParams = useCallback(() => {
    const params = new URLSearchParams();
    if (filterFrom) params.set("from", filterFrom);
    if (filterTo) params.set("to", filterTo);
    if (filterActor) params.set("actor", filterActor);
    if (filterAction) params.set("action", filterAction);
    if (filterSearch) params.set("search", filterSearch);
    params.set("page", String(page));
    params.set("per_page", String(PER_PAGE));
    return params.toString();
  }, [filterFrom, filterTo, filterActor, filterAction, filterSearch, page]);

  const { data, isLoading } = useQuery({
    queryKey: [
      "audit-logs",
      page,
      filterFrom,
      filterTo,
      filterActor,
      filterAction,
      filterSearch,
    ],
    queryFn: () =>
      apiGet<AuditLogResponse>(`/audit-logs?${buildApiParams()}`),
  });

  const totalPages = data
    ? Math.max(1, Math.ceil(data.total / PER_PAGE))
    : 1;

  const handleExport = () => {
    const params = new URLSearchParams();
    if (filterFrom) params.set("from", filterFrom);
    if (filterTo) params.set("to", filterTo);
    if (filterActor) params.set("actor", filterActor);
    if (filterAction) params.set("action", filterAction);
    if (filterSearch) params.set("search", filterSearch);
    const qs = params.toString();
    window.open(
      `${API_BASE}/audit-logs/export${qs ? `?${qs}` : ""}`,
      "_blank"
    );
  };

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Audit Logs</h2>
        <Button onClick={handleExport}>
          <Download className="mr-2 h-4 w-4" />
          Export to Excel
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Filters</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-5 sm:grid-cols-2 lg:grid-cols-5">
          <div className="grid gap-2">
            <Label htmlFor="filter-from">Date From</Label>
            <Input
              id="filter-from"
              type="date"
              className="dark:color-scheme-dark"
              value={filterFrom}
              onChange={(e) => setFilter("from", e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="filter-to">Date To</Label>
            <Input
              id="filter-to"
              type="date"
              className="dark:color-scheme-dark"
              value={filterTo}
              onChange={(e) => setFilter("to", e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label>Actor</Label>
            <Select
              value={filterActor || undefined}
              onValueChange={(v: string | null) => setFilter("actor", !v || v === "_all" ? "" : v)}
            >
              <SelectTrigger>
                <SelectValue placeholder="All actors" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="_all">All actors</SelectItem>
                {filters?.actors?.map((a) => (
                  <SelectItem key={a} value={a}>
                    {a}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label>Action</Label>
            <Select
              value={filterAction || undefined}
              onValueChange={(v: string | null) => setFilter("action", !v || v === "_all" ? "" : v)}
            >
              <SelectTrigger>
                <SelectValue placeholder="All actions" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="_all">All actions</SelectItem>
                {filters?.actions?.map((a) => (
                  <SelectItem key={a} value={a}>
                    {a}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="filter-search">Search</Label>
            <div className="flex gap-2">
              <Input
                id="filter-search"
                className="min-w-0"
                placeholder="Name, email, or ID..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") setFilter("search", searchInput);
                }}
              />
              <Button
                variant="outline"
                size="icon"
                onClick={() => setFilter("search", searchInput)}
                aria-label="Search"
              >
                <Search className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Activity Log</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Timestamp</TableHead>
                <TableHead>Actor</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Resource Type</TableHead>
                <TableHead>Resource</TableHead>
                <TableHead>IP Address</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data?.data?.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="whitespace-nowrap">
                    {new Date(log.timestamp).toLocaleString()}
                  </TableCell>
                  <TableCell>{log.actor}</TableCell>
                  <TableCell>{log.action}</TableCell>
                  <TableCell>{log.resource_type}</TableCell>
                  <TableCell>{formatResource(log)}</TableCell>
                  <TableCell className="font-mono">
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
          </div>

          <div className="flex flex-col items-start gap-4 pt-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="text-muted-foreground">
              Page {page} of {totalPages} ({data?.total ?? 0} total)
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
              >
                <ChevronLeft className="mr-1 h-4 w-4" />
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage(page + 1)}
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

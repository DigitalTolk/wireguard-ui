import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPatch } from "@/lib/api-client";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { toast } from "sonner";
import type { User } from "@/lib/types";

export function UsersPage() {
  const qc = useQueryClient();
  const { data: users, isLoading } = useQuery({
    queryKey: ["users"],
    queryFn: () => apiGet<User[]>("/users"),
  });

  const toggleAdmin = useMutation({
    mutationFn: ({ username, admin }: { username: string; admin: boolean }) =>
      apiPatch(`/users/${username}/admin`, { admin }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["users"] }),
    onError: (err: Error) => toast.error(err.message),
  });

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold tracking-tight">Users</h2>
      <Card>
        <CardHeader>
          <CardTitle>SSO Users</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Display Name</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Last Login</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users?.map((user) => (
                <TableRow key={user.username}>
                  <TableCell className="font-medium">
                    {user.username}
                  </TableCell>
                  <TableCell>{user.display_name || "-"}</TableCell>
                  <TableCell>{user.email || "-"}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Switch
                        checked={user.admin}
                        onCheckedChange={(checked) =>
                          toggleAdmin.mutate({ username: user.username, admin: checked })
                        }
                        aria-label={`Toggle admin for ${user.username}`}
                      />
                      <Badge variant={user.admin ? "default" : "secondary"}>
                        {user.admin ? "Admin" : "User"}
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell>
                    {user.updated_at
                      ? new Date(user.updated_at).toLocaleString()
                      : "-"}
                  </TableCell>
                </TableRow>
              ))}
              {(!users || users.length === 0) && (
                <TableRow>
                  <TableCell
                    colSpan={5}
                    className="text-center text-muted-foreground"
                  >
                    No users have logged in yet
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          </div>
          <p className="pt-4 text-muted-foreground">
            Users are managed through your SSO provider. This list shows
            everyone who has logged in via OIDC.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

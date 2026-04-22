import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut, apiDelete } from "@/lib/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { User } from "@/lib/types";

export function UsersPage() {
  const qc = useQueryClient();
  const { data: users, isLoading } = useQuery({
    queryKey: ["users"],
    queryFn: () => apiGet<User[]>("/users"),
  });

  const deleteUser = useMutation({
    mutationFn: (username: string) => apiDelete(`/users/${username}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      toast.success("User deleted");
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const [showCreate, setShowCreate] = useState(false);
  const [newUser, setNewUser] = useState({
    username: "",
    email: "",
    admin: false,
  });

  const createUser = useMutation({
    mutationFn: (data: { username: string; email: string; admin: boolean }) =>
      apiPost<User>("/users", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      toast.success("User created");
      setShowCreate(false);
      setNewUser({ username: "", email: "", admin: false });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const [editUser, setEditUser] = useState<User | null>(null);
  const [editForm, setEditForm] = useState({
    email: "",
    display_name: "",
    admin: false,
  });

  const updateUser = useMutation({
    mutationFn: ({
      username,
      ...data
    }: {
      username: string;
      email: string;
      display_name: string;
      admin: boolean;
    }) => apiPut<User>(`/users/${username}`, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      toast.success("User updated");
      setEditUser(null);
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const handleOpenEdit = (user: User) => {
    setEditForm({
      email: user.email || "",
      display_name: user.display_name || "",
      admin: user.admin,
    });
    setEditUser(user);
  };

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Users</h2>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New User
        </Button>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Registered Users</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Role</TableHead>
                <TableHead className="w-24"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users?.map((user) => (
                <TableRow key={user.username}>
                  <TableCell className="font-medium">{user.username}</TableCell>
                  <TableCell>{user.email || "-"}</TableCell>
                  <TableCell>
                    <Badge variant={user.admin ? "default" : "secondary"}>
                      {user.admin ? "Admin" : "User"}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleOpenEdit(user)}
                        aria-label={`Edit user ${user.username}`}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => {
                          if (confirm(`Delete user "${user.username}"?`)) {
                            deleteUser.mutate(user.username);
                          }
                        }}
                        aria-label={`Delete user ${user.username}`}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Create User Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>New User</DialogTitle>
          </DialogHeader>
          <div className="grid gap-5 py-4">
            <div className="grid gap-2">
              <Label htmlFor="create-username">Username</Label>
              <Input
                id="create-username"
                placeholder="johndoe"
                value={newUser.username}
                onChange={(e) =>
                  setNewUser((p) => ({ ...p, username: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="create-email">Email</Label>
              <Input
                id="create-email"
                type="email"
                placeholder="john@example.com"
                value={newUser.email}
                onChange={(e) =>
                  setNewUser((p) => ({ ...p, email: e.target.value }))
                }
              />
            </div>
            <div className="flex items-center gap-3">
              <Switch
                id="create-admin"
                checked={newUser.admin}
                onCheckedChange={(v) =>
                  setNewUser((p) => ({ ...p, admin: v }))
                }
              />
              <Label htmlFor="create-admin">Admin</Label>
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createUser.mutate(newUser)}
              disabled={!newUser.username || createUser.isPending}
            >
              {createUser.isPending ? "Creating..." : "Create"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Edit User Dialog */}
      <Dialog open={!!editUser} onOpenChange={() => setEditUser(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Edit User</DialogTitle>
          </DialogHeader>
          <div className="grid gap-5 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-email">Email</Label>
              <Input
                id="edit-email"
                type="email"
                value={editForm.email}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, email: e.target.value }))
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-displayname">Display Name</Label>
              <Input
                id="edit-displayname"
                value={editForm.display_name}
                onChange={(e) =>
                  setEditForm((p) => ({ ...p, display_name: e.target.value }))
                }
              />
            </div>
            <div className="flex items-center gap-3">
              <Switch
                id="edit-admin"
                checked={editForm.admin}
                onCheckedChange={(v) =>
                  setEditForm((p) => ({ ...p, admin: v }))
                }
              />
              <Label htmlFor="edit-admin">Admin</Label>
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setEditUser(null)}>
              Cancel
            </Button>
            <Button
              onClick={() =>
                editUser &&
                updateUser.mutate({
                  username: editUser.username,
                  ...editForm,
                })
              }
              disabled={updateUser.isPending}
            >
              {updateUser.isPending ? "Saving..." : "Save"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

import { useState } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import {
  Users,
  Monitor,
  Server,
  Settings,
  Shield,
  Wifi,
  ClipboardList,
  Info,
  LogOut,
  Menu,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { apiPost } from "@/lib/api-client";

const navItems = [
  { to: "/", icon: Shield, label: "Clients", end: true },
  { to: "/status", icon: Monitor, label: "Status" },
  { to: "/server", icon: Server, label: "Server", admin: true },
  { to: "/settings", icon: Settings, label: "Settings", admin: true },
  { to: "/users", icon: Users, label: "Users", admin: true },
  { to: "/wol", icon: Wifi, label: "Wake-on-LAN" },
  { to: "/audit", icon: ClipboardList, label: "Audit Logs", admin: true },
  { to: "/about", icon: Info, label: "About" },
];

export function AppShell() {
  const { data: me, isLoading } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const handleLogout = async () => {
    await apiPost("/auth/logout");
    window.location.href = "./api/v1/auth/oidc/login";
  };

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Skeleton className="h-8 w-48" />
      </div>
    );
  }

  const sidebarContent = (
    <>
      <div className="p-4">
        <h1 className="text-lg font-semibold text-sidebar-foreground">
          WireGuard UI
        </h1>
      </div>
      <Separator />
      <nav className="flex-1 space-y-1 p-2">
        {navItems
          .filter((item) => !item.admin || me?.admin)
          .map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              onClick={() => setSidebarOpen(false)}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
                    : "text-sidebar-foreground hover:bg-sidebar-accent/50"
                }`
              }
            >
              <item.icon className="h-4 w-4" aria-hidden="true" />
              {item.label}
            </NavLink>
          ))}
      </nav>
      <Separator />
      <div className="p-4">
        <div className="flex items-center justify-between">
          <div className="text-sm">
            <div className="font-medium text-sidebar-foreground">
              {me?.display_name || me?.username}
            </div>
            {me?.admin && (
              <span className="text-xs text-muted-foreground">Admin</span>
            )}
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={handleLogout}
            aria-label="Log out"
          >
            <LogOut className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </>
  );

  return (
    <div className="flex h-screen flex-col md:flex-row">
      {/* Mobile header */}
      <header className="flex items-center justify-between border-b border-sidebar-border bg-sidebar-background p-3 md:hidden">
        <h1 className="text-lg font-semibold text-sidebar-foreground">
          WireGuard UI
        </h1>
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setSidebarOpen(!sidebarOpen)}
          aria-label={sidebarOpen ? "Close menu" : "Open menu"}
        >
          {sidebarOpen ? (
            <X className="h-5 w-5" />
          ) : (
            <Menu className="h-5 w-5" />
          )}
        </Button>
      </header>

      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={() => setSidebarOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-sidebar-border bg-sidebar-background transition-transform md:static md:translate-x-0 ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full"
        }`}
        role="navigation"
        aria-label="Main navigation"
      >
        {sidebarContent}
      </aside>

      {/* Main content */}
      <main className="min-h-0 flex-1 overflow-auto" role="main">
        <div className="container mx-auto max-w-6xl p-4 sm:p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}

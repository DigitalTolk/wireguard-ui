import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ThemeProvider } from "@/components/layout/ThemeProvider";
import { AppShell } from "@/components/layout/AppShell";
import { ClientsPage } from "@/pages/ClientsPage";
import { StatusPage } from "@/pages/StatusPage";
import { ServerPage } from "@/pages/ServerPage";
import { SettingsPage } from "@/pages/SettingsPage";
import { UsersPage } from "@/pages/UsersPage";
import { WolPage } from "@/pages/WolPage";
import { AuditPage } from "@/pages/AuditPage";
import { AboutPage } from "@/pages/AboutPage";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30 * 1000,
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <TooltipProvider>
          <BrowserRouter>
            <Routes>
              <Route element={<AppShell />}>
                <Route path="/" element={<ClientsPage />} />
                <Route path="/status" element={<StatusPage />} />
                <Route path="/server" element={<ServerPage />} />
                <Route path="/settings" element={<SettingsPage />} />
                <Route path="/users" element={<UsersPage />} />
                <Route path="/wol" element={<WolPage />} />
                <Route path="/audit" element={<AuditPage />} />
                <Route path="/about" element={<AboutPage />} />
              </Route>
            </Routes>
          </BrowserRouter>
          <Toaster />
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;

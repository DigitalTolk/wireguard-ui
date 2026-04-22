import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut, apiPatch, apiDelete } from "@/lib/api-client";
import type { Client, ClientData } from "@/lib/types";

export function useClients() {
  return useQuery({
    queryKey: ["clients"],
    queryFn: () => apiGet<ClientData[]>("/clients"),
  });
}

export function useClient(id: string) {
  return useQuery({
    queryKey: ["clients", id],
    queryFn: () => apiGet<ClientData>(`/clients/${id}`),
    enabled: !!id,
  });
}

export function useCreateClient() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Client>) =>
      apiPost<Client>("/clients", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clients"] }),
  });
}

export function useUpdateClient() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Client> & { id: string }) =>
      apiPut<Client>(`/clients/${id}`, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clients"] }),
  });
}

export function useSetClientStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      apiPatch<Client>(`/clients/${id}/status`, { enabled }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clients"] }),
  });
}

export function useDeleteClient() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/clients/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clients"] }),
  });
}

export function useSuggestClientIPs(subnetRange?: string) {
  return useQuery({
    queryKey: ["suggest-ips", subnetRange],
    queryFn: () =>
      apiGet<string[]>(`/suggest-client-ips${subnetRange ? `?sr=${subnetRange}` : ""}`),
    enabled: false,
  });
}

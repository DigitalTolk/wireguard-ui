import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api-client";
import type { MeResponse } from "@/lib/types";

export function useAuth() {
  return useQuery({
    queryKey: ["auth", "me"],
    queryFn: () => apiGet<MeResponse>("/auth/me"),
    retry: false,
    staleTime: 5 * 60 * 1000,
  });
}

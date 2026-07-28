import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { get, type ApiOptions } from "../lib/api";
import type { PlanView } from "../types/plan";

export function usePlan(
  id: string | undefined,
  options?: ApiOptions,
): UseQueryResult<PlanView> {
  return useQuery<PlanView>({
    queryKey: ["plan", id],
    queryFn: ({ signal }) => {
      if (id === undefined) throw new Error("Plan ID is required");
      return get<PlanView>(`/plans/${encodeURIComponent(id)}`, {
        ...options,
        signal,
      });
    },
    enabled: id !== undefined && id !== "",
  });
}

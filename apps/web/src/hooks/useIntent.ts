import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { get, type ApiOptions } from "../lib/api";
import type { IntentView } from "../types/intent";

export function useIntent(
  id: string | undefined,
  options?: ApiOptions,
): UseQueryResult<IntentView> {
  return useQuery<IntentView>({
    queryKey: ["intent", id],
    queryFn: ({ signal }) => {
      if (id === undefined) throw new Error("Intent ID is required");
      return get<IntentView>(`/intents/${encodeURIComponent(id)}`, {
        ...options,
        signal,
      });
    },
    enabled: id !== undefined && id !== "",
  });
}

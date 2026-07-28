import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { get, type ApiOptions } from "../lib/api";
import type { IntentView } from "../types/intent";

const INTENTS_KEY = "intents";

export interface UseIntentsOptions {
  readonly orgId?: string;
  readonly limit?: number;
  readonly offset?: number;
}

export function useIntents(
  options?: UseIntentsOptions & ApiOptions,
): UseQueryResult<IntentView[]> {
  const { orgId = "default", limit = 20, offset = 0, ...apiOpts } =
    options ?? {};

  return useQuery<IntentView[]>({
    queryKey: [INTENTS_KEY, orgId, limit, offset],
    queryFn: ({ signal }) => {
      const params = new URLSearchParams({
        org_id: orgId,
        limit: String(limit),
        offset: String(offset),
      });
      return get<IntentView[]>(`/intents?${params.toString()}`, {
        ...apiOpts,
        signal,
      });
    },
  });
}

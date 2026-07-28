import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { get, type ApiOptions } from "../lib/api";
import type { TaskView } from "../types/task";

export function useTasks(
  intentId: string | undefined,
  options?: ApiOptions,
): UseQueryResult<TaskView[]> {
  return useQuery<TaskView[]>({
    queryKey: ["tasks", intentId],
    queryFn: ({ signal }) => {
      const params = new URLSearchParams();
      if (intentId !== undefined) {
        params.set("intentId", intentId);
      }
      return get<TaskView[]>(`/tasks?${params.toString()}`, {
        ...options,
        signal,
      });
    },
    enabled: intentId !== undefined && intentId !== "",
  });
}

import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { useSSE } from "./useSSE";
import type { ConnectionState, StreamEvent } from "../types/stream";

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

interface UseIntentStreamOptions {
  readonly intentId: string | undefined;
}

interface UseIntentStreamResult {
  readonly events: StreamEvent[];
  readonly connectionState: ConnectionState;
  readonly clearEvents: () => void;
}

export function useIntentStream(options: UseIntentStreamOptions): UseIntentStreamResult {
  const { intentId } = options;
  const [events, setEvents] = useState<StreamEvent[]>([]);
  const queryClient = useQueryClient();

  const url = intentId !== undefined
    ? `${BASE_URL}/intents/${encodeURIComponent(intentId)}/stream`
    : "";

  const onEvent = useCallback(
    (event: StreamEvent) => {
      setEvents((prev) => [event, ...prev].slice(0, 200));

      // Invalidate affected queries so React Query refetches.
      void queryClient.invalidateQueries({ queryKey: ["intent", intentId] });
      void queryClient.invalidateQueries({ queryKey: ["tasks", intentId] });
      void queryClient.invalidateQueries({ queryKey: ["intents"] });

      // Also invalidate plan if available.
      void queryClient.invalidateQueries({ queryKey: ["plan", intentId] });
    },
    [queryClient, intentId],
  );

  const { connectionState } = useSSE({
    url,
    onEvent,
    enabled: intentId !== undefined,
  });

  const clearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  return { events, connectionState, clearEvents };
}

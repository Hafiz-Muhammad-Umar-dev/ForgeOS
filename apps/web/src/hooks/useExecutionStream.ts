import { useCallback, useEffect, useRef, useState } from "react";
import type { ExecutionEvent } from "../types/execution";

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

interface UseExecutionStreamResult {
  readonly liveEvents: ExecutionEvent[];
  readonly isConnected: boolean;
}

export function useExecutionStream(intentId: string | undefined): UseExecutionStreamResult {
  const [liveEvents, setLiveEvents] = useState<ExecutionEvent[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  const connect = useCallback(() => {
    if (intentId === undefined) return;

    const url = `${BASE_URL}/executions/${encodeURIComponent(intentId)}/stream`;
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => {
      setIsConnected(true);
    };

    es.addEventListener("execution_event", (event: MessageEvent) => {
      try {
        const evt = JSON.parse(String(event.data)) as ExecutionEvent;
        setLiveEvents((prev) => [evt, ...prev].slice(0, 200));
      } catch {
        // Ignore parse errors.
      }
    });

    es.onerror = () => {
      setIsConnected(false);
    };
  }, [intentId]);

  useEffect(() => {
    connect();
    return () => {
      if (eventSourceRef.current !== null) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
        setIsConnected(false);
      }
    };
  }, [connect]);

  return { liveEvents, isConnected };
}

import { useCallback, useEffect, useRef, useState } from "react";
import { SSEClient } from "../lib/sse";
import type { ConnectionState, StreamEvent } from "../types/stream";

interface UseSSEOptions {
  readonly url: string;
  readonly onEvent?: (event: StreamEvent) => void;
  readonly enabled?: boolean;
}

interface UseSSEResult {
  readonly connectionState: ConnectionState;
  readonly connect: () => void;
  readonly disconnect: () => void;
}

export function useSSE(options: UseSSEOptions): UseSSEResult {
  const { url, onEvent, enabled = true } = options;
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected");
  const clientRef = useRef<SSEClient | null>(null);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const connect = useCallback(() => {
    if (clientRef.current !== null) {
      clientRef.current.disconnect();
    }

    const client = new SSEClient({
      url,
      onEvent: (event) => {
        onEventRef.current?.(event);
      },
      onStateChange: (state) => {
        setConnectionState(state);
      },
    });

    clientRef.current = client;
    client.connect();
  }, [url]);

  const disconnect = useCallback(() => {
    if (clientRef.current !== null) {
      clientRef.current.disconnect();
      clientRef.current = null;
    }
    setConnectionState("disconnected");
  }, []);

  // Connect on mount if enabled, disconnect on unmount.
  useEffect(() => {
    if (enabled) {
      connect();
    }
    return () => {
      disconnect();
    };
  }, [enabled, connect, disconnect]);

  return { connectionState, connect, disconnect };
}

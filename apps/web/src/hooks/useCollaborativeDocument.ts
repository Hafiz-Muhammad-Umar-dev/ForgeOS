import { useEffect, useRef, useState } from "react";
import * as Y from "yjs";
import { YjsWebSocketProvider } from "../lib/collaboration/YjsWebSocketProvider";
import type { ConnectionState } from "../lib/collaboration/wsProvider";

interface UseCollaborativeDocumentOptions {
  readonly intentId: string | undefined;
  readonly fieldName?: string;
}

interface UseCollaborativeDocumentResult {
  readonly provider: YjsWebSocketProvider | null;
  readonly ytext: Y.Text | null;
  readonly connectionState: ConnectionState;
  readonly connectedUsers: number;
}

export function useCollaborativeDocument(
  options: UseCollaborativeDocumentOptions,
): UseCollaborativeDocumentResult {
  const { intentId, fieldName = "content" } = options;
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected");
  const providerRef = useRef<YjsWebSocketProvider | null>(null);
  const ytextRef = useRef<Y.Text | null>(null);
  const [connectedUsers, setConnectedUsers] = useState(0);

  useEffect(() => {
    if (intentId === undefined) return;

    const provider = new YjsWebSocketProvider(intentId);
    const ytext = provider.doc.getText(fieldName);
    providerRef.current = provider;
    ytextRef.current = ytext;

    // Track connection state.
    const unsubscribeSync = provider.onSync(() => {
      setConnectionState("connected");
    });

    // Poll connection state periodically.
    const intervalId = setInterval(() => {
      setConnectionState(provider.wsProvider.getState());
      // Count connected users (including self).
      const states = provider.awareness.getStates();
      setConnectedUsers(states.size);
    }, 1000);

    return () => {
      clearInterval(intervalId);
      unsubscribeSync();
      provider.destroy();
      providerRef.current = null;
      ytextRef.current = null;
      setConnectionState("disconnected");
      setConnectedUsers(0);
    };
  }, [intentId, fieldName]);

  return {
    provider: providerRef.current,
    ytext: ytextRef.current,
    connectionState,
    connectedUsers,
  };
}

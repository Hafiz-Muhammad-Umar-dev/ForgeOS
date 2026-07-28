import { useEffect, useState } from "react";
import type { YjsWebSocketProvider } from "../lib/collaboration/YjsWebSocketProvider";

interface AwarenessUser {
  readonly clientId: number;
  readonly username?: string;
  readonly color?: string;
}

export function useAwareness(
  provider: YjsWebSocketProvider | null,
  username?: string,
  color?: string,
): AwarenessUser[] {
  const [users, setUsers] = useState<AwarenessUser[]>([]);

  useEffect(() => {
    if (provider === null) return;

    // Set local awareness state.
    if (username !== undefined || color !== undefined) {
      const localState: Record<string, string> = {};
      if (username !== undefined) localState.username = username;
      if (color !== undefined) localState.color = color;
      provider.awareness.setLocalStateField("user", localState);
    }

    const handleChange = () => {
      const states = provider.awareness.getStates();
      const userList: AwarenessUser[] = [];
      states.forEach((state: unknown, clientId: number) => {
        const s = state as { user?: { username?: string; color?: string } };
        const user = s.user;
        userList.push({
          clientId,
          username: user?.username,
          color: user?.color,
        });
      });
      setUsers(userList);
    };

    provider.awareness.on("change", handleChange);
    handleChange();

    return () => {
      provider.awareness.off("change", handleChange);
    };
  }, [provider, username, color]);

  return users;
}

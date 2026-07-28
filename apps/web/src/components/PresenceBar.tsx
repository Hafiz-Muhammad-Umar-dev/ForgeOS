interface PresenceUser {
  readonly clientId: number;
  readonly username?: string;
  readonly color?: string;
}

interface PresenceBarProps {
  readonly users: PresenceUser[];
}

const DEFAULT_COLORS = [
  "#6366f1", // indigo
  "#22c55e", // green
  "#f59e0b", // amber
  "#ef4444", // red
  "#ec4899", // pink
  "#06b6d4", // cyan
  "#a855f7", // purple
  "#f97316", // orange
];

export function PresenceBar({ users }: PresenceBarProps): React.ReactNode {
  if (users.length === 0) {
    return (
      <div className="flex items-center gap-2 px-4 py-2">
        <span className="text-xs text-slate-500">No other users connected</span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 px-4 py-2">
      <span className="text-xs text-slate-400">
        {String(users.length)} connected
      </span>
      <div className="flex -space-x-1.5">
        {users.map((user, i) => {
          const color = user.color ?? DEFAULT_COLORS[i % DEFAULT_COLORS.length];
          const initial = (user.username ?? "?").charAt(0).toUpperCase();
          return (
            <div
              key={user.clientId}
              className="flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-medium text-white ring-2 ring-slate-900"
              style={{ backgroundColor: color }}
              title={user.username ?? `User ${String(user.clientId)}`}
            >
              {initial}
            </div>
          );
        })}
      </div>
    </div>
  );
}

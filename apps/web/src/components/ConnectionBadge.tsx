import type { ConnectionState } from "../lib/collaboration/wsProvider";

interface ConnectionBadgeProps {
  readonly state: ConnectionState;
}

const config: Record<ConnectionState, { readonly label: string; readonly className: string }> = {
  connected: { label: "Connected", className: "bg-green-500" },
  connecting: { label: "Connecting", className: "bg-yellow-500" },
  reconnecting: { label: "Reconnecting", className: "bg-yellow-500" },
  disconnected: { label: "Disconnected", className: "bg-slate-600" },
  error: { label: "Error", className: "bg-red-500" },
};

export function ConnectionBadge({ state }: ConnectionBadgeProps): React.ReactNode {
  const c = config[state];
  return (
    <div className="flex items-center gap-2">
      <span className={`h-2 w-2 rounded-full ${c.className}`} />
      <span className="text-xs text-slate-400">{c.label}</span>
    </div>
  );
}

import type { ConnectionState } from "../types/stream";

interface ConnectionStatusProps {
  readonly state: ConnectionState;
}

const statusConfig: Record<ConnectionState, { readonly label: string; readonly className: string }> = {
  connected: { label: "Connected", className: "bg-green-500" },
  connecting: { label: "Connecting", className: "bg-yellow-500" },
  reconnecting: { label: "Reconnecting", className: "bg-yellow-500" },
  disconnected: { label: "Disconnected", className: "bg-slate-600" },
  error: { label: "Error", className: "bg-red-500" },
};

export function ConnectionStatus({ state }: ConnectionStatusProps): React.ReactNode {
  const config = statusConfig[state];

  return (
    <div className="flex items-center gap-2">
      <span className={`h-2 w-2 rounded-full ${config.className}`} />
      <span className="text-xs text-slate-400">{config.label}</span>
    </div>
  );
}

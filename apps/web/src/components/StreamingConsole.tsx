import type { StreamEvent } from "../types/stream";
import { ConnectionStatus } from "./ConnectionStatus";
import type { ConnectionState } from "../types/stream";

interface StreamingConsoleProps {
  readonly events: StreamEvent[];
  readonly connectionState: ConnectionState;
}

export function StreamingConsole({ events, connectionState }: StreamingConsoleProps): React.ReactNode {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/50">
      <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
        <h3 className="text-sm font-semibold text-white">Live Stream</h3>
        <ConnectionStatus state={connectionState} />
      </div>

      {events.length === 0 && (
        <div className="p-6 text-center">
          <p className="text-sm text-slate-500">Waiting for events...</p>
        </div>
      )}

      {events.length > 0 && (
        <div className="max-h-80 overflow-y-auto">
          {events.map((event) => (
            <div
              key={`${event.id}-${String(event.timestamp)}`}
              className="border-b border-slate-800/50 px-4 py-2.5 last:border-0"
            >
              <div className="mb-1 flex items-center gap-2">
                <span className="rounded bg-slate-800 px-1.5 py-0.5 font-mono text-[10px] font-medium uppercase tracking-wider text-slate-300">
                  {event.type}
                </span>
                <span className="text-[10px] text-slate-500">
                  {new Date(event.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <pre className="overflow-x-auto font-mono text-[11px] text-slate-400">
                {truncateJson(event.data, 120)}
              </pre>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function truncateJson(data: string, maxChars: number): string {
  if (data.length <= maxChars) return data;
  return `${data.slice(0, maxChars)}...`;
}

import type { AgentEvent } from "../../types/workspace";

interface AgentTimelineProps {
  readonly events?: AgentEvent[];
}

const eventColors: Record<string, string> = {
  planning: "text-violet-400",
  searching: "text-blue-400",
  executing: "text-amber-400",
  reading: "text-cyan-400",
  writing: "text-green-400",
  running: "text-indigo-400",
  completed: "text-emerald-400",
  failed: "text-red-400",
};

const eventDots: Record<string, string> = {
  planning: "bg-violet-500",
  searching: "bg-blue-500",
  executing: "bg-amber-500",
  reading: "bg-cyan-500",
  writing: "bg-green-500",
  running: "bg-indigo-500",
  completed: "bg-emerald-500",
  failed: "bg-red-500",
};

function formatDuration(ms?: number): string {
  if (ms === undefined) return "";
  if (ms < 1000) return `${String(ms)}ms`;
  if (ms < 60000) return `${String(Math.round(ms / 1000))}s`;
  return `${String(Math.floor(ms / 60000))}m ${String(Math.round((ms % 60000) / 1000))}s`;
}

export function AgentTimeline({ events }: AgentTimelineProps): React.ReactNode {
  const items = events ?? [];

  return (
    <div>
      <h3 className="mb-3 px-1 text-[10px] font-semibold uppercase tracking-widest text-slate-500">
        Agent Timeline
      </h3>
      {items.length === 0 && (
        <div className="px-1">
          <p className="text-xs text-slate-500">No events yet.</p>
        </div>
      )}
      <div className="space-y-2">
        {items.map((event) => (
          <div
            key={event.id}
            className="flex items-start gap-3 rounded-lg border border-slate-800 bg-slate-900/30 px-3 py-2"
          >
            <span
              className={`mt-1 h-2 w-2 shrink-0 rounded-full ${
                eventDots[event.type] ?? "bg-slate-600"
              }`}
            />
            <div className="min-w-0 flex-1">
              <p
                className={`text-xs font-medium ${
                  eventColors[event.type] ?? "text-slate-400"
                }`}
              >
                {event.description}
              </p>
              <div className="mt-0.5 flex items-center gap-2 text-[10px] text-slate-500">
                <span>
                  {new Date(event.timestamp).toLocaleTimeString()}
                </span>
                {event.tool !== undefined && (
                  <>
                    <span>·</span>
                    <span>{event.tool}</span>
                  </>
                )}
                {event.duration !== undefined && (
                  <>
                    <span>·</span>
                    <span>{formatDuration(event.duration)}</span>
                  </>
                )}
              </div>
            </div>
            <span
              className={`shrink-0 text-[10px] font-medium ${
                event.status === "running"
                  ? "text-blue-400"
                  : event.status === "failed"
                    ? "text-red-400"
                    : "text-emerald-400"
              }`}
            >
              {event.status}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

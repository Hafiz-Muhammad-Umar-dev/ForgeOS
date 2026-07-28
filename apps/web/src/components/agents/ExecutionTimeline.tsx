import type { ExecutionEvent } from "../../types/execution";

interface ExecutionTimelineProps {
  readonly events: ExecutionEvent[];
}

const typeDots: Record<string, string> = {
  tool_started: "bg-blue-500",
  tool_completed: "bg-green-500",
  tool_failed: "bg-red-500",
  reasoning: "bg-purple-500",
  memory: "bg-cyan-500",
  cost_update: "bg-yellow-500",
};

export function ExecutionTimeline({ events }: ExecutionTimelineProps): React.ReactNode {
  if (events.length === 0) {
    return (
      <div className="px-1">
        <p className="text-xs text-slate-500">No timeline events.</p>
      </div>
    );
  }

  return (
    <div className="relative space-y-0">
      {events.slice(0, 50).map((evt, i) => (
        <div key={evt.id} className="relative flex gap-3 pb-3 pl-5">
          {/* Vertical line */}
          {i < events.length - 1 && (
            <div className="absolute bottom-0 left-[7px] top-3 w-px bg-slate-800" />
          )}
          {/* Dot */}
          <span
            className={`mt-1 h-2 w-2 shrink-0 rounded-full ${typeDots[evt.type] ?? "bg-slate-600"}`}
          />
          {/* Content */}
          <div className="min-w-0 flex-1">
            <p className="text-xs text-slate-300">{evt.content}</p>
            <p className="text-[10px] text-slate-500">
              {new Date(evt.timestamp).toLocaleTimeString()}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}

import { useEffect, useRef } from "react";
import type { ExecutionEvent } from "../../types/execution";

interface ExecutionConsoleProps {
  readonly events: ExecutionEvent[];
}

const eventStyles: Record<string, string> = {
  tool_started: "text-blue-400",
  tool_completed: "text-green-400",
  tool_failed: "text-red-400",
  reasoning: "text-purple-400",
  memory: "text-cyan-400",
  cost_update: "text-yellow-400",
};

const eventLabels: Record<string, string> = {
  tool_started: "TOOL STARTED",
  tool_completed: "TOOL COMPLETED",
  tool_failed: "TOOL FAILED",
  reasoning: "REASONING",
  memory: "MEMORY",
  cost_update: "COST UPDATE",
};

export function ExecutionConsole({ events }: ExecutionConsoleProps): React.ReactNode {
  const scrollRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (scrollRef.current !== null) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [events]);

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-slate-800 px-3 py-1.5">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">
          Execution Console
        </span>
      </div>
      <div ref={scrollRef} className="flex-1 overflow-y-auto p-2 font-mono text-xs">
        {events.length === 0 && (
          <p className="text-slate-500">No events yet.</p>
        )}
        {events.map((evt) => (
          <div key={evt.id} className="mb-1">
            <span className="text-slate-600">{new Date(evt.timestamp).toLocaleTimeString()}</span>{" "}
            <span className={eventStyles[evt.type] ?? "text-slate-400"}>
              [{eventLabels[evt.type] ?? evt.type}]
            </span>{" "}
            <span className="text-slate-400">{evt.content}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

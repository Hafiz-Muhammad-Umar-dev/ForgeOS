import type { ExecutionEvent } from "../../types/execution";

interface LiveLogViewerProps {
  readonly events: ExecutionEvent[];
}

export function LiveLogViewer({ events }: LiveLogViewerProps): React.ReactNode {
  const displayEvents = events.slice(0, 100);

  return (
    <div className="h-full overflow-y-auto font-mono text-xs">
      {displayEvents.length === 0 && (
        <p className="p-3 text-slate-500">No log entries yet.</p>
      )}
      {displayEvents.map((evt) => (
        <div key={evt.id} className="flex border-b border-slate-800/30 px-3 py-1.5">
          <span className="shrink-0 text-slate-600">
            {new Date(evt.timestamp).toLocaleTimeString()}
          </span>
          <span className="mx-2 text-slate-700">|</span>
          <span className="text-slate-400">{evt.content}</span>
        </div>
      ))}
    </div>
  );
}

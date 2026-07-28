import { useEffect, useRef, useState } from "react";
import type { DeploymentLog } from "../../types/deployment";

interface DeploymentLogsProps {
  readonly logs: DeploymentLog[];
  readonly isLoading?: boolean;
}

export function DeploymentLogs({ logs, isLoading = false }: DeploymentLogsProps): React.ReactNode {
  const [autoScroll, setAutoScroll] = useState(true);
  const scrollRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (autoScroll && scrollRef.current !== null) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  const levelColors: Record<string, string> = {
    info: "text-slate-400",
    warn: "text-yellow-400",
    error: "text-red-400",
    debug: "text-slate-600",
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-slate-800 px-3 py-1.5">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">Build Logs</span>
        <label className="flex items-center gap-1.5 text-[10px] text-slate-500">
          <input type="checkbox" checked={autoScroll} onChange={(e) => { setAutoScroll(e.target.checked); }} />
          Auto-scroll
        </label>
      </div>
      <div ref={scrollRef} className="flex-1 overflow-y-auto p-2 font-mono text-xs">
        {isLoading && <p className="text-slate-500">Loading logs...</p>}
        {!isLoading && logs.length === 0 && <p className="text-slate-500">No logs yet.</p>}
        {logs.map((log, i) => (
          <div key={`${log.timestamp}-${String(i)}`} className="mb-0.5">
            <span className="text-slate-600">{log.timestamp}</span>{" "}
            <span className={levelColors[log.level] ?? "text-slate-400"}>
              [{log.level.toUpperCase()}]
            </span>{" "}
            <span className="text-slate-300">{log.message}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

import type { AgentStatus } from "../../types/agent";
import type { TaskStatus } from "../../types/execution";

type StatusLike = AgentStatus | TaskStatus;

const config: Record<string, { readonly label: string; readonly className: string }> = {
  idle: { label: "Idle", className: "bg-slate-500" },
  pending: { label: "Pending", className: "bg-slate-500" },
  running: { label: "Running", className: "bg-blue-500" },
  thinking: { label: "Thinking", className: "bg-purple-500" },
  waiting: { label: "Waiting", className: "bg-yellow-500" },
  completed: { label: "Completed", className: "bg-green-500" },
  failed: { label: "Failed", className: "bg-red-500" },
  skipped: { label: "Skipped", className: "bg-slate-600" },
};

interface ExecutionStatusBadgeProps {
  readonly status: StatusLike;
}

export function ExecutionStatusBadge({ status }: ExecutionStatusBadgeProps): React.ReactNode {
  const c = config[status] ?? { label: status, className: "bg-slate-500" };
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-slate-800 px-2.5 py-0.5 text-[10px] font-medium text-white">
      <span className={`h-1.5 w-1.5 rounded-full ${c.className}`} />
      {c.label}
    </span>
  );
}

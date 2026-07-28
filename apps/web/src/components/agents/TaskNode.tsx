import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";

const statusColors: Record<string, string> = {
  pending: "border-slate-600 bg-slate-900",
  running: "border-blue-500 bg-blue-950/20",
  completed: "border-green-500 bg-green-950/20",
  failed: "border-red-500 bg-red-950/20",
  skipped: "border-slate-700 bg-slate-900/50",
};

const statusDots: Record<string, string> = {
  pending: "bg-slate-500",
  running: "bg-blue-500",
  completed: "bg-green-500",
  failed: "bg-red-500",
  skipped: "bg-slate-600",
};

export const TaskNode = memo(function TaskNode({ data }: NodeProps) {
  const d: Record<string, unknown> = data;
  const statusVal = d["status"];
  const status = typeof statusVal === "string" ? statusVal : "pending";
  const border = statusColors[status] ?? statusColors.pending;
  const dot = statusDots[status] ?? statusDots.pending;
  const labelVal = d["label"];
  const label = typeof labelVal === "string" ? labelVal : "";
  const progress = typeof d["progress"] === "number" ? d["progress"] : 0;
  const runtime = typeof d["runtime"] === "number" ? d["runtime"] : 0;
  const tokens = typeof d["tokens"] === "number" ? d["tokens"] : 0;
  const cost = typeof d["cost"] === "number" ? d["cost"] : 0;

  return (
    <div
      className={`rounded-xl border-2 px-4 py-3 shadow-lg ${border}`}
      style={{ minWidth: 180 }}
    >
      <Handle type="target" position={Position.Top} className="!bg-slate-600" />
      <div className="mb-2 flex items-center gap-2">
        <span className={`h-2 w-2 rounded-full ${dot}`} />
        <span className="text-sm font-medium text-white">{label}</span>
      </div>
      <div className="space-y-1 text-[10px] text-slate-400">
        <div className="flex justify-between">
          <span>Progress</span>
          <span>{String(progress)}%</span>
        </div>
        <div className="flex justify-between">
          <span>Runtime</span>
          <span>{runtime > 0 ? `${String(Math.round(runtime / 1000))}s` : "—"}</span>
        </div>
        <div className="flex justify-between">
          <span>Tokens</span>
          <span>{String(tokens)}</span>
        </div>
        <div className="flex justify-between">
          <span>Cost</span>
          <span>{`$${cost.toFixed(4)}`}</span>
        </div>
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-slate-600" />
    </div>
  );
});

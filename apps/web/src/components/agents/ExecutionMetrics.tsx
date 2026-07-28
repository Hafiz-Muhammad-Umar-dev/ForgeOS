import type { ExecutionMetrics as Metrics } from "../../types/execution";

interface ExecutionMetricsProps {
  readonly metrics: Metrics | null;
}

export function ExecutionMetrics({ metrics }: ExecutionMetricsProps): React.ReactNode {
  const items = metrics === null
    ? [
        { label: "Total Tokens", value: "—" },
        { label: "Prompt Tokens", value: "—" },
        { label: "Completion Tokens", value: "—" },
        { label: "Est. Cost", value: "—" },
        { label: "Duration", value: "—" },
        { label: "Avg Latency", value: "—" },
        { label: "Tools Executed", value: "—" },
      ]
    : [
        { label: "Total Tokens", value: String(metrics.totalTokens) },
        { label: "Prompt Tokens", value: String(metrics.promptTokens) },
        { label: "Completion Tokens", value: String(metrics.completionTokens) },
        { label: "Est. Cost", value: `$${metrics.estimatedCost.toFixed(4)}` },
        { label: "Duration", value: `${String(Math.round(metrics.executionDuration / 1000))}s` },
        { label: "Avg Latency", value: `${String(Math.round(metrics.averageLatency))}ms` },
        { label: "Tools Executed", value: String(metrics.toolsExecuted) },
      ];

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
      {items.map((item) => (
        <div
          key={item.label}
          className="rounded-lg border border-slate-800 bg-slate-900/30 px-3 py-2"
        >
          <p className="text-[10px] font-medium uppercase tracking-wider text-slate-500">
            {item.label}
          </p>
          <p className="mt-0.5 text-sm font-semibold text-white">{item.value}</p>
        </div>
      ))}
    </div>
  );
}

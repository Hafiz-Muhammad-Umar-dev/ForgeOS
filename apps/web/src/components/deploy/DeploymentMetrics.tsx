import type { DeploymentMetric } from "../../types/deployment";

interface DeploymentMetricsProps {
  readonly metrics: DeploymentMetric[];
}

export function DeploymentMetrics({ metrics }: DeploymentMetricsProps): React.ReactNode {
  if (metrics.length === 0) {
    return (
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {["CPU", "Memory", "Network", "Requests/s"].map((label) => (
          <div key={label} className="rounded-lg border border-slate-800 bg-slate-900/30 px-3 py-2">
            <p className="text-[10px] font-medium uppercase tracking-wider text-slate-500">{label}</p>
            <p className="mt-0.5 text-sm font-semibold text-white">—</p>
          </div>
        ))}
      </div>
    );
  }

  const latest = metrics[metrics.length - 1];
  const items = [
    { label: "CPU", value: `${(latest.cpu * 100).toFixed(1)}%` },
    { label: "Memory", value: `${(latest.memory * 100).toFixed(1)}%` },
    { label: "Disk", value: `${(latest.disk * 100).toFixed(1)}%` },
    { label: "Network", value: `${(latest.network / 1024 / 1024).toFixed(1)} MB/s` },
    { label: "Requests/s", value: String(Math.round(latest.requestsPerSec)) },
    { label: "Latency", value: `${String(Math.round(latest.latency))}ms` },
    { label: "Errors", value: String(latest.errors) },
    { label: "Region", value: "—" },
  ];

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      {items.map((item) => (
        <div key={item.label} className="rounded-lg border border-slate-800 bg-slate-900/30 px-3 py-2">
          <p className="text-[10px] font-medium uppercase tracking-wider text-slate-500">{item.label}</p>
          <p className="mt-0.5 text-sm font-semibold text-white">{item.value}</p>
        </div>
      ))}
    </div>
  );
}

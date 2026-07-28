import type { GitMetrics as GitMetricsType } from "../../types/git";

interface GitMetricsProps {
  readonly metrics: GitMetricsType | null;
}

export function GitMetrics({ metrics }: GitMetricsProps): React.ReactNode {
  const items = metrics === null
    ? [
        { label: "Repository Size", value: "—" },
        { label: "Commits", value: "—" },
        { label: "Branches", value: "—" },
        { label: "Changed Files", value: "—" },
        { label: "Staged Files", value: "—" },
      ]
    : [
        { label: "Repository Size", value: metrics.repositorySize },
        { label: "Commits", value: String(metrics.commitCount) },
        { label: "Branches", value: String(metrics.branchCount) },
        { label: "Changed Files", value: String(metrics.changedFiles) },
        { label: "Staged Files", value: String(metrics.stagedFiles) },
      ];

  return (
    <div className="grid grid-cols-2 gap-2 px-3 py-2">
      {items.map((item) => (
        <div key={item.label} className="rounded bg-slate-800/30 px-2 py-1">
          <p className="text-[10px] text-slate-500">{item.label}</p>
          <p className="text-xs font-medium text-slate-200">{item.value}</p>
        </div>
      ))}
    </div>
  );
}

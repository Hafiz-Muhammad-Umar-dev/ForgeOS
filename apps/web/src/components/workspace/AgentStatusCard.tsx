interface StatusItem {
  readonly label: string;
  readonly value: string;
}

interface AgentStatusCardProps {
  readonly title?: string;
  readonly items?: StatusItem[];
}

export function AgentStatusCard({
  title = "Agent Status",
  items,
}: AgentStatusCardProps): React.ReactNode {
  const statusItems: StatusItem[] = items ?? [
    { label: "Active Agent", value: "—" },
    { label: "Current Task", value: "—" },
    { label: "Current Tool", value: "—" },
    { label: "Progress", value: "Idle" },
    { label: "Token Usage", value: "0" },
    { label: "Cost", value: "$0.00" },
    { label: "Runtime", value: "—" },
    { label: "Workspace", value: "—" },
    { label: "Intent", value: "—" },
  ];

  return (
    <div>
      <h3 className="mb-3 px-1 text-[10px] font-semibold uppercase tracking-widest text-slate-500">
        {title}
      </h3>
      <div className="space-y-1">
        {statusItems.map((item) => (
          <div
            key={item.label}
            className="rounded-lg border border-slate-800 bg-slate-900/30 px-3 py-2"
          >
            <p className="text-[10px] font-medium uppercase tracking-wider text-slate-500">
              {item.label}
            </p>
            <p className="mt-0.5 text-sm text-slate-200">{item.value}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

interface HealthBadgeProps {
  readonly status: string;
}

const config: Record<string, { readonly label: string; readonly className: string }> = {
  healthy: { label: "Healthy", className: "bg-green-500" },
  degraded: { label: "Degraded", className: "bg-yellow-500" },
  unhealthy: { label: "Unhealthy", className: "bg-red-500" },
  pending: { label: "Pending", className: "bg-slate-500" },
  deploying: { label: "Deploying", className: "bg-blue-500" },
  building: { label: "Building", className: "bg-blue-500" },
  uploading: { label: "Uploading", className: "bg-indigo-500" },
  running: { label: "Running", className: "bg-green-500" },
  stopped: { label: "Stopped", className: "bg-slate-600" },
  failed: { label: "Failed", className: "bg-red-500" },
  cancelled: { label: "Cancelled", className: "bg-slate-500" },
};

export function HealthBadge({ status }: HealthBadgeProps): React.ReactNode {
  const c = config[status] ?? { label: status, className: "bg-slate-500" };
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-slate-800 px-2.5 py-0.5 text-[10px] font-medium text-white">
      <span className={`h-1.5 w-1.5 rounded-full ${c.className}`} />
      {c.label}
    </span>
  );
}

import type { TimelineStage, DeployStatus } from "../../types/deployment";

interface DeploymentTimelineProps {
  readonly stages: TimelineStage[];
}

const stageDots: Record<DeployStatus, string> = {
  pending: "bg-slate-500",
  deploying: "bg-blue-500",
  building: "bg-blue-500",
  uploading: "bg-indigo-500",
  running: "bg-green-500",
  healthy: "bg-green-500",
  stopped: "bg-slate-600",
  failed: "bg-red-500",
  cancelled: "bg-slate-500",
};

export function DeploymentTimeline({ stages }: DeploymentTimelineProps): React.ReactNode {
  if (stages.length === 0) {
    return (
      <div className="px-1">
        <p className="text-xs text-slate-500">No timeline stages.</p>
      </div>
    );
  }

  return (
    <div className="relative space-y-0">
      {stages.map((stage, i) => (
        <div key={stage.name} className="relative flex gap-3 pb-4 pl-5">
          {i < stages.length - 1 && <div className="absolute bottom-0 left-[7px] top-3 w-px bg-slate-800" />}
          <span className={`mt-1 h-2 w-2 shrink-0 rounded-full ${stageDots[stage.status]}`} />
          <div className="min-w-0 flex-1">
            <p className="text-xs font-medium text-slate-200">{stage.name}</p>
            <p className="text-[10px] text-slate-500">
              {stage.status} {stage.duration !== undefined ? `· ${String(Math.round(stage.duration / 1000))}s` : ""}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}

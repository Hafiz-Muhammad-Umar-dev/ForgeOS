import { HealthBadge } from "./HealthBadge";
import { DeploymentActions } from "./DeploymentActions";
import { DeploymentLogs } from "./DeploymentLogs";
import { DeploymentMetrics } from "./DeploymentMetrics";
import { DeploymentTimeline } from "./DeploymentTimeline";
import { EnvironmentEditor } from "./EnvironmentEditor";
import { RollbackPanel } from "./RollbackPanel";
import { useDeployment } from "../../hooks/useDeployment";

interface DeploymentInspectorProps {
  readonly id: string;
}

export function DeploymentInspector({ id }: DeploymentInspectorProps): React.ReactNode {
  const { deployment, logs, metrics, timeline, healthChecks, envVars, isLoading, start, stop, restart, pause, resume, del } = useDeployment(id);

  if (isLoading) {
    return <div className="p-4 text-center"><p className="text-xs text-slate-500">Loading...</p></div>;
  }
  if (deployment === null) {
    return <div className="p-4 text-center"><p className="text-xs text-slate-500">Deployment not found.</p></div>;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-bold text-white">{deployment.projectName}</h2>
          <p className="text-xs text-slate-400">ID: {deployment.id}</p>
        </div>
        <HealthBadge status={deployment.status} />
      </div>

      {/* Actions */}
      <DeploymentActions
        onStart={start}
        onStop={stop}
        onRestart={restart}
        onPause={pause}
        onResume={resume}
        onDelete={del}
        isRunning={deployment.status === "running" || deployment.status === "healthy"}
      />

      {/* Details */}
      <div className="grid grid-cols-2 gap-3">
        <DetailItem label="Branch" value={deployment.branch} />
        <DetailItem label="Commit" value={deployment.commit?.slice(0, 7)} />
        <DetailItem label="Image" value={deployment.image} />
        <DetailItem label="Region" value={deployment.region} />
        <DetailItem label="Created By" value={deployment.createdBy} />
        <DetailItem label="Started" value={deployment.startedAt ? new Date(deployment.startedAt).toLocaleString() : undefined} />
        <DetailItem label="Finished" value={deployment.finishedAt ? new Date(deployment.finishedAt).toLocaleString() : undefined} />
        <DetailItem label="URL" value={deployment.url} />
      </div>

      {/* Health Checks */}
      {healthChecks.length > 0 && (
        <div>
          <h4 className="mb-2 text-[10px] font-semibold uppercase tracking-widest text-slate-500">Health Checks</h4>
          {healthChecks.map((h) => (
            <div key={h.endpoint} className="flex items-center gap-2 rounded bg-slate-800/30 px-3 py-2 text-xs">
              <HealthBadge status={h.status} />
              <span className="text-slate-300">{h.type.toUpperCase()}</span>
              <span className="text-slate-500">{h.endpoint}</span>
            </div>
          ))}
        </div>
      )}

      {/* Metrics */}
      <DeploymentMetrics metrics={metrics} />

      {/* Timeline */}
      <DeploymentTimeline stages={timeline} />

      {/* Env Vars */}
      <EnvironmentEditor variables={envVars} onAdd={() => undefined} onDelete={() => undefined} />

      {/* Rollback */}
      <RollbackPanel previousDeployments={[]} onRollback={() => undefined} />

      {/* Logs */}
      <div className="h-64">
        <DeploymentLogs logs={logs} />
      </div>
    </div>
  );
}

function DetailItem({ label, value }: { readonly label: string; readonly value?: string }): React.ReactNode {
  return (
    <div className="rounded bg-slate-800/30 px-3 py-2">
      <p className="text-[10px] font-medium uppercase tracking-wider text-slate-500">{label}</p>
      <p className="mt-0.5 text-xs text-slate-200">{value ?? "—"}</p>
    </div>
  );
}

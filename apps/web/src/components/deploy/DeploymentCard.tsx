import { Link } from "react-router-dom";
import type { Deployment } from "../../types/deployment";
import { HealthBadge } from "./HealthBadge";

interface DeploymentCardProps {
  readonly deployment: Deployment;
}

export function DeploymentCard({ deployment }: DeploymentCardProps): React.ReactNode {
  return (
    <Link
      to={`/deployments?id=${deployment.id}`}
      className="block rounded-xl border border-slate-800 bg-slate-900/50 p-4 transition-colors hover:border-slate-700"
    >
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm font-medium text-white">{deployment.projectName}</span>
        <HealthBadge status={deployment.status} />
      </div>
      <div className="space-y-1 text-xs text-slate-400">
        {deployment.branch !== undefined && (
          <div className="flex justify-between">
            <span>Branch</span>
            <span className="text-slate-300">{deployment.branch}</span>
          </div>
        )}
        {deployment.commit !== undefined && (
          <div className="flex justify-between">
            <span>Commit</span>
            <span className="font-mono text-slate-300">{deployment.commit.slice(0, 7)}</span>
          </div>
        )}
        <div className="flex justify-between">
          <span>Created</span>
          <span className="text-slate-300">{new Date(deployment.createdAt).toLocaleDateString()}</span>
        </div>
        {deployment.region !== undefined && (
          <div className="flex justify-between">
            <span>Region</span>
            <span className="text-slate-300">{deployment.region}</span>
          </div>
        )}
      </div>
    </Link>
  );
}

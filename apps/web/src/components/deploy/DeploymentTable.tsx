import { Link } from "react-router-dom";
import type { Deployment } from "../../types/deployment";
import { HealthBadge } from "./HealthBadge";

interface DeploymentTableProps {
  readonly deployments: Deployment[];
}

export function DeploymentTable({ deployments }: DeploymentTableProps): React.ReactNode {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left text-sm">
        <thead className="border-b border-slate-800 bg-slate-900/80">
          <tr>
            <th className="px-4 py-3 font-medium text-slate-400">Project</th>
            <th className="px-4 py-3 font-medium text-slate-400">Status</th>
            <th className="px-4 py-3 font-medium text-slate-400">Branch</th>
            <th className="px-4 py-3 font-medium text-slate-400">Commit</th>
            <th className="px-4 py-3 font-medium text-slate-400">Region</th>
            <th className="px-4 py-3 font-medium text-slate-400">Created</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {deployments.map((d) => (
            <tr key={d.id} className="hover:bg-slate-900/30">
              <td className="px-4 py-3">
                <Link to={`/deployments?id=${d.id}`} className="text-white hover:text-forge-400">
                  {d.projectName}
                </Link>
              </td>
              <td className="px-4 py-3"><HealthBadge status={d.status} /></td>
              <td className="px-4 py-3 text-slate-300">{d.branch ?? "—"}</td>
              <td className="px-4 py-3 font-mono text-xs text-slate-300">{d.commit?.slice(0, 7) ?? "—"}</td>
              <td className="px-4 py-3 text-slate-300">{d.region ?? "—"}</td>
              <td className="px-4 py-3 text-slate-400">{new Date(d.createdAt).toLocaleDateString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

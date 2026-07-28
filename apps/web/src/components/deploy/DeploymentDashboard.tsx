import { useSearchParams } from "react-router-dom";
import { useDeployments } from "../../hooks/useDeployments";
import { DeploymentToolbar } from "./DeploymentToolbar";
import { DeploymentFilters } from "./DeploymentFilters";
import { DeploymentTable } from "./DeploymentTable";
import { DeploymentCard } from "./DeploymentCard";
import { DeploymentInspector } from "./DeploymentInspector";

export function DeploymentDashboard(): React.ReactNode {
  const [params] = useSearchParams();
  const selectedId = params.get("id");
  const { deployments, isLoading, error, refresh } = useDeployments();

  if (selectedId !== null) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-6">
        <DeploymentInspector id={selectedId} />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <DeploymentToolbar onRefresh={refresh} />
      <DeploymentFilters onFilterChange={() => undefined} />

      {isLoading && (
        <div className="flex flex-1 items-center justify-center">
          <p className="text-sm text-slate-500">Loading deployments...</p>
        </div>
      )}

      {error !== null && (
        <div className="flex flex-1 flex-col items-center justify-center gap-3">
          <p className="text-sm text-red-400">{error}</p>
          <button type="button" onClick={refresh} className="rounded bg-slate-800 px-3 py-1.5 text-xs text-slate-400">
            Retry
          </button>
        </div>
      )}

      {!isLoading && error === null && deployments.length === 0 && (
        <div className="flex flex-1 items-center justify-center">
          <p className="text-sm text-slate-500">No deployments found.</p>
        </div>
      )}

      {!isLoading && error === null && deployments.length > 0 && (
        <div className="flex-1 overflow-y-auto p-4">
          <div className="hidden md:block">
            <DeploymentTable deployments={deployments} />
          </div>
          <div className="grid gap-4 md:hidden">
            {deployments.map((d) => (
              <DeploymentCard key={d.id} deployment={d} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

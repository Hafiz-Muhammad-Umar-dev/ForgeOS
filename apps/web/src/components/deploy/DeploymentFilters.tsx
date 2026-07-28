import { useState } from "react";

interface DeploymentFiltersProps {
  readonly onFilterChange: (filters: { project?: string; status?: string; search?: string }) => void;
}

export function DeploymentFilters({ onFilterChange }: DeploymentFiltersProps): React.ReactNode {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");

  function handleChange(): void {
    onFilterChange({
      search: search || undefined,
      status: status || undefined,
    });
  }

  return (
    <div className="flex items-center gap-3 px-4 py-2">
      <input
        value={search}
        onChange={(e) => { setSearch(e.target.value); handleChange(); }}
        placeholder="Search deployments..."
        className="flex-1 rounded-lg border border-slate-700 bg-slate-800 px-3 py-1.5 text-xs text-white placeholder-slate-500"
      />
      <select
        value={status}
        onChange={(e) => { setStatus(e.target.value); handleChange(); }}
        className="rounded-lg border border-slate-700 bg-slate-800 px-2 py-1.5 text-xs text-white"
      >
        <option value="">All</option>
        <option value="running">Running</option>
        <option value="building">Building</option>
        <option value="healthy">Healthy</option>
        <option value="failed">Failed</option>
        <option value="stopped">Stopped</option>
      </select>
    </div>
  );
}

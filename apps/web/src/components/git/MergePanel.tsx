import { useState } from "react";

interface MergePanelProps {
  readonly onMerge: (branch: string) => void;
  readonly branches: string[];
}

export function MergePanel({ onMerge, branches }: MergePanelProps): React.ReactNode {
  const [selectedBranch, setSelectedBranch] = useState("");

  return (
    <div className="space-y-3 px-3 py-2">
      <span className="text-xs font-semibold text-slate-400">Merge</span>
      <select
        value={selectedBranch}
        onChange={(e) => { setSelectedBranch(e.target.value); }}
        className="w-full rounded border border-slate-700 bg-slate-800 px-2 py-1.5 text-xs text-white"
      >
        <option value="">Select branch...</option>
        {branches.map((b) => (
          <option key={b} value={b}>
            {b}
          </option>
        ))}
      </select>
      <button
        type="button"
        disabled={selectedBranch.length === 0}
        onClick={() => { onMerge(selectedBranch); }}
        className="w-full rounded bg-forge-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-forge-500 disabled:opacity-50"
      >
        Merge
      </button>
    </div>
  );
}

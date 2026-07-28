import { useState } from "react";
import type { GitBranch } from "../../types/git";

interface BranchSelectorProps {
  readonly branches: GitBranch[];
  readonly currentBranch: string | null;
  readonly onCheckout: (name: string) => void;
  readonly onCreate: (name: string, base?: string) => void;
  readonly onDelete: (name: string) => void;
}

export function BranchSelector({
  branches,
  onCheckout,
  onCreate,
  onDelete,
}: BranchSelectorProps): React.ReactNode {
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");

  const localBranches = branches.filter((b) => !b.isRemote);
  const remoteBranches = branches.filter((b) => b.isRemote);

  function handleCreate(): void {
    if (newName.trim().length === 0) return;
    onCreate(newName.trim());
    setNewName("");
    setShowCreate(false);
  }

  return (
    <div>
      <div className="mb-2 flex items-center justify-between px-3">
        <span className="text-xs font-semibold text-slate-400">Branches</span>
        <button
          type="button"
          onClick={() => { setShowCreate(true); }}
          className="rounded px-1.5 text-xs text-slate-500 hover:bg-slate-800 hover:text-slate-200"
        >
          + New
        </button>
      </div>

      {showCreate && (
        <div className="mb-2 px-3">
          <input
            value={newName}
            onChange={(e) => { setNewName(e.target.value); }}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleCreate();
              if (e.key === "Escape") setShowCreate(false);
            }}
            placeholder="Branch name"
            className="w-full rounded border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-white placeholder-slate-500"
            autoFocus
          />
        </div>
      )}

      <div className="space-y-0.5">
        {localBranches.map((b) => (
          <div
            key={b.name}
            className={`group flex items-center gap-2 px-3 py-1 text-sm ${
              b.isCurrent ? "bg-forge-800/30 text-white" : "text-slate-400 hover:bg-slate-800/30"
            }`}
          >
            <span className="text-[10px]">{b.isCurrent ? "✓" : "○"}</span>
            <span
              className="flex-1 cursor-pointer truncate"
              onClick={() => { onCheckout(b.name); }}
            >
              {b.name}
            </span>
            {!b.isCurrent && (
              <button
                type="button"
                onClick={() => { onDelete(b.name); }}
                className="hidden px-1 text-[10px] text-red-500 group-hover:inline hover:text-red-400"
              >
                del
              </button>
            )}
            {(b.ahead > 0 || b.behind > 0) && (
              <span className="text-[10px] text-slate-500">
                {b.ahead > 0 && <span className="text-green-400">↑{String(b.ahead)}</span>}
                {b.behind > 0 && <span className="text-red-400">↓{String(b.behind)}</span>}
              </span>
            )}
          </div>
        ))}
      </div>

      {remoteBranches.length > 0 && (
        <div className="mt-3">
          <p className="mb-1 px-3 text-[10px] font-medium uppercase tracking-wider text-slate-500">Remote</p>
          {remoteBranches.map((b) => (
            <div key={b.name} className="flex items-center gap-2 px-3 py-1 text-sm text-slate-500">
              <span className="text-[10px]">↕</span>
              <span className="truncate">{b.name}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

import { useState } from "react";

interface RollbackPanelProps {
  readonly previousDeployments: Array<{ id: string; projectName: string; createdAt: string }>;
  readonly onRollback: (targetId: string) => void;
}

export function RollbackPanel({ previousDeployments, onRollback }: RollbackPanelProps): React.ReactNode {
  const [selectedId, setSelectedId] = useState("");
  const [confirming, setConfirming] = useState(false);

  function handleRollback(): void {
    if (selectedId.length === 0) return;
    if (!confirming) {
      setConfirming(true);
      return;
    }
    onRollback(selectedId);
    setConfirming(false);
    setSelectedId("");
  }

  return (
    <div className="space-y-3">
      <h4 className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">Rollback</h4>
      {previousDeployments.length === 0 && (
        <p className="text-xs text-slate-500">No previous deployments available.</p>
      )}
      {previousDeployments.length > 0 && (
        <>
          <select
            value={selectedId}
            onChange={(e) => { setSelectedId(e.target.value); setConfirming(false); }}
            className="w-full rounded border border-slate-700 bg-slate-800 px-2 py-1.5 text-xs text-white"
          >
            <option value="">Select target...</option>
            {previousDeployments.map((d) => (
              <option key={d.id} value={d.id}>{d.projectName} — {new Date(d.createdAt).toLocaleDateString()}</option>
            ))}
          </select>
          <button
            type="button"
            onClick={handleRollback}
            disabled={selectedId.length === 0}
            className="w-full rounded bg-amber-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-amber-500 disabled:opacity-50"
          >
            {confirming ? "Confirm Rollback" : "Rollback"}
          </button>
        </>
      )}
    </div>
  );
}

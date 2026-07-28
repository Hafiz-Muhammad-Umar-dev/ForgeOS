import { useState } from "react";
import { commit } from "../../lib/git/gitClient";

interface CommitPanelProps {
  readonly onCommitComplete?: () => void;
}

export function CommitPanel({ onCommitComplete }: CommitPanelProps): React.ReactNode {
  const [message, setMessage] = useState("");
  const [description, setDescription] = useState("");
  const [signOff, setSignOff] = useState(false);
  const [isCommitting, setIsCommitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const isValid = message.trim().length > 0;

  function handleCommit(): void {
    if (!isValid) return;
    setIsCommitting(true);
    setError(null);
    commit(message.trim(), description.trim() || undefined, signOff)
      .then(() => {
        setMessage("");
        setDescription("");
        onCommitComplete?.();
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        setIsCommitting(false);
      });
  }

  return (
    <div className="space-y-3 px-3 py-2">
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold text-slate-400">Commit</span>
      </div>

      <input
        value={message}
        onChange={(e) => { setMessage(e.target.value); }}
        onKeyDown={(e) => {
          if (e.key === "Enter" && e.ctrlKey) handleCommit();
        }}
        placeholder="Commit message"
        className="w-full rounded border border-slate-700 bg-slate-800 px-2 py-1.5 text-xs text-white placeholder-slate-500"
      />

      <textarea
        value={description}
        onChange={(e) => { setDescription(e.target.value); }}
        placeholder="Description (optional)"
        rows={3}
        className="w-full resize-none rounded border border-slate-700 bg-slate-800 px-2 py-1.5 text-xs text-white placeholder-slate-500"
      />

      <label className="flex items-center gap-2 text-xs text-slate-400">
        <input
          type="checkbox"
          checked={signOff}
          onChange={(e) => { setSignOff(e.target.checked); }}
          className="rounded border-slate-700 bg-slate-800 text-forge-500"
        />
        Sign-off commit
      </label>

      {error !== null && (
        <p className="text-xs text-red-400">{error}</p>
      )}

      <button
        type="button"
        disabled={!isValid || isCommitting}
        onClick={handleCommit}
        className="w-full rounded bg-forge-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-forge-500 disabled:opacity-50"
      >
        {isCommitting ? "Committing..." : "Commit"}
      </button>
    </div>
  );
}

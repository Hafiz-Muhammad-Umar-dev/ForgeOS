import { useCallback, useState } from "react";
import { useGit } from "../../hooks/useGit";
import { useGitBranches } from "../../hooks/useGitBranches";
import { push, pull, fetch as gitFetch } from "../../lib/git/gitClient";
import { GitToolbar } from "./GitToolbar";
import { GitStatusList } from "./GitStatusList";
import { GitMetrics } from "./GitMetrics";
import { BranchSelector } from "./BranchSelector";
import { CommitPanel } from "./CommitPanel";
import { GitIcon } from "./GitIcon";

type GitView = "status" | "branches" | "history";

export function GitSidebar(): React.ReactNode {
  const [activeView, setActiveView] = useState<GitView>("status");
  const { files, metrics, refresh, stage, unstage, discard, backendAvailable } = useGit();
  const { branches, currentBranch, create: createBranch, checkout: checkoutBranch, delete: deleteBranch } = useGitBranches();

  const handleCommit = useCallback(() => {
    refresh();
  }, [refresh]);

  const handlePush = useCallback(() => {
    push().catch(() => {/* ignore */});
  }, []);

  const handlePull = useCallback(() => {
    pull().catch(() => {/* ignore */});
  }, []);

  const handleFetch = useCallback(() => {
    gitFetch().catch(() => {/* ignore */});
  }, []);

  if (!backendAvailable) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex items-center gap-2 border-b border-slate-800 px-3 py-2">
          <GitIcon />
          <span className="text-xs font-semibold text-slate-400">Source Control</span>
        </div>
        <div className="flex flex-1 items-center justify-center">
          <p className="px-4 text-center text-xs text-slate-500">
            Git backend unavailable
          </p>
        </div>
      </div>
    );
  }


  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-slate-800 px-3 py-2">
        <GitIcon />
        <span className="text-xs font-semibold text-slate-400">Source Control</span>
        {currentBranch !== null && (
          <span className="ml-auto truncate rounded bg-slate-800 px-1.5 py-0.5 text-[10px] text-slate-400">
            {currentBranch}
          </span>
        )}
      </div>

      {/* Toolbar */}
      <GitToolbar
        onCommit={handleCommit}
        onPush={handlePush}
        onPull={handlePull}
        onFetch={handleFetch}
        onRefresh={refresh}
      />

      {/* View tabs */}
      <div className="flex border-b border-slate-800">
        {(["status", "branches", "history"] as GitView[]).map((view) => (
          <button
            key={view}
            type="button"
            onClick={() => { setActiveView(view); }}
            className={`flex-1 py-1.5 text-[10px] font-medium uppercase tracking-wider transition-colors ${
              activeView === view
                ? "border-b-2 border-forge-500 text-white"
                : "text-slate-500 hover:text-slate-300"
            }`}
          >
            {view === "status" ? "Changes" : view}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {activeView === "status" && (
          <>
            <GitMetrics metrics={metrics} />
            <CommitPanel onCommitComplete={handleCommit} />
            <div className="border-t border-slate-800 pt-2">
              <GitStatusList files={files} onStage={stage} onUnstage={unstage} onDiscard={discard} />
            </div>
          </>
        )}
        {activeView === "branches" && (
          <BranchSelector
            branches={branches}
            currentBranch={currentBranch}
            onCheckout={checkoutBranch}
            onCreate={createBranch}
            onDelete={deleteBranch}
          />
        )}
        {activeView === "history" && (
          <div className="p-3 text-center">
            <p className="text-xs text-slate-500">Commit history will appear here.</p>
          </div>
        )}
      </div>
    </div>
  );
}

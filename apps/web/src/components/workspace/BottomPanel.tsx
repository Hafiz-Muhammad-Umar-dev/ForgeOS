import { useState } from "react";
import { TerminalPanel } from "./TerminalPanel";
import { OutputPanel } from "./OutputPanel";
import { ProblemsPanel } from "./ProblemsPanel";

type BottomTab = "terminal" | "problems" | "output" | "logs";

interface BottomPanelProps {
  readonly aiLogs?: string[];
}

const tabs: Array<{ id: BottomTab; label: string }> = [
  { id: "terminal", label: "Terminal" },
  { id: "problems", label: "Problems" },
  { id: "output", label: "Output" },
  { id: "logs", label: "AI Logs" },
];

export function BottomPanel({ aiLogs }: BottomPanelProps): React.ReactNode {
  const [activeTab, setActiveTab] = useState<BottomTab>("terminal");

  return (
    <div className="flex h-full flex-col border-t border-slate-800 bg-slate-900/80">
      {/* Tabs */}
      <div className="flex shrink-0 border-b border-slate-800">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => { setActiveTab(tab.id); }}
            className={`px-3 py-1.5 text-[11px] font-medium transition-colors ${
              activeTab === tab.id
                ? "border-b-2 border-forge-500 text-white"
                : "text-slate-500 hover:text-slate-300"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === "terminal" && <TerminalPanel />}
        {activeTab === "problems" && <ProblemsPanel />}
        {activeTab === "output" && <OutputPanel />}
        {activeTab === "logs" && (
          <div className="flex h-full flex-col">
            <div className="border-b border-slate-800 px-3 py-1.5">
              <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">
                AI Logs
              </span>
            </div>
            <div className="flex-1 overflow-y-auto p-3">
              {aiLogs !== undefined && aiLogs.length > 0 ? (
                aiLogs.map((log, i) => (
                  <pre key={i} className="font-mono text-xs text-slate-400">
                    {log}
                  </pre>
                ))
              ) : (
                <p className="text-xs text-slate-500">No AI logs yet.</p>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

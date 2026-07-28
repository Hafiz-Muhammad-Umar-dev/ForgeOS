import { useCallback } from "react";
import { useEditor } from "../hooks/useEditor";
import { writeFile } from "../lib/workspace/workspaceClient";
import { WorkspaceExplorer } from "../components/workspace/WorkspaceExplorer";
import { EditorView } from "../components/workspace/EditorView";
import { BottomPanel } from "../components/workspace/BottomPanel";
import { AgentStatusCard } from "../components/workspace/AgentStatusCard";
import { AgentTimeline } from "../components/workspace/AgentTimeline";
import type { AgentEvent } from "../types/workspace";

const sampleEvents: AgentEvent[] = [
  {
    id: "evt-1",
    type: "planning",
    description: "Planning task structure",
    status: "completed",
    timestamp: Date.now() - 60000,
    duration: 3200,
  },
  {
    id: "evt-2",
    type: "executing",
    description: "Implementing file operations",
    status: "running",
    timestamp: Date.now() - 30000,
    tool: "execute_command",
    duration: 15000,
  },
];

export function WorkspaceView(): React.ReactNode {
  const {
    tabs,
    activeTabId,
    openFile,
    closeTab,
    setActiveTab,
    updateContent,
    markSaved,
  } = useEditor();

  const handleOpenFile = useCallback(
    (path: string, name: string, content: string) => {
      openFile(path, name, content);
    },
    [openFile],
  );

  const handleSave = useCallback(
    (tabId: string) => {
      const tab = tabs.find((t) => t.id === tabId);
      if (tab === undefined) return;
      writeFile(tab.path, tab.content)
        .then(() => {
          markSaved(tabId);
        })
        .catch(() => undefined);
    },
    [tabs, markSaved],
  );

  return (
    <div className="flex h-[calc(100vh-3.5rem)] flex-col">
      {/* Main content area: sidebar + editor + right panel */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left sidebar — 280px workspace explorer */}
        <aside className="hidden w-[280px] shrink-0 border-r border-slate-800 bg-slate-900/50 lg:block">
          <WorkspaceExplorer onOpenFile={handleOpenFile} />
        </aside>

        {/* Center — editor + bottom panel */}
        <div className="flex flex-1 flex-col overflow-hidden">
          {/* Editor */}
          <div className="flex-1 overflow-hidden">
            <EditorView
              tabs={tabs}
              activeTabId={activeTabId}
              onSelectTab={setActiveTab}
              onCloseTab={closeTab}
              onContentChange={updateContent}
              onSave={handleSave}
            />
          </div>

          {/* Bottom panel — 250px default height */}
          <div className="h-[250px] shrink-0">
            <BottomPanel />
          </div>
        </div>

        {/* Right sidebar — 280px agent activity */}
        <aside className="hidden w-[280px] shrink-0 border-l border-slate-800 bg-slate-900/50 overflow-y-auto lg:block">
          <div className="space-y-6 px-4 py-4">
            <AgentStatusCard />
            <AgentTimeline events={sampleEvents} />
          </div>
        </aside>
      </div>
    </div>
  );
}

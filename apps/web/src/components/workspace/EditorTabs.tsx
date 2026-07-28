import { memo } from "react";
import { X } from "lucide-react";
import type { EditorTab } from "../../types/workspace";

interface EditorTabsProps {
  readonly tabs: EditorTab[];
  readonly activeTabId: string | null;
  readonly onSelect: (id: string) => void;
  readonly onClose: (id: string) => void;
}

export const EditorTabs = memo(function EditorTabs({
  tabs,
  activeTabId,
  onSelect,
  onClose,
}: EditorTabsProps): React.ReactNode {
  if (tabs.length === 0) return null;

  return (
    <div className="flex overflow-x-auto border-b border-slate-800 bg-slate-900/80">
      {tabs.map((tab) => (
        <div
          key={tab.id}
          onClick={() => { onSelect(tab.id); }}
          className={`group flex shrink-0 cursor-pointer items-center gap-2 border-r border-slate-800 px-3 py-2 text-sm transition-colors ${
            tab.id === activeTabId
              ? "border-t-2 border-t-forge-500 bg-slate-800/50 text-white"
              : "text-slate-400 hover:bg-slate-800/30 hover:text-slate-200"
          }`}
        >
          <span className="truncate max-w-[120px]">{tab.name}</span>
          {tab.isDirty && (
            <span className="h-2 w-2 rounded-full bg-forge-500" />
          )}
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onClose(tab.id);
            }}
            className="ml-1 rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-100 hover:bg-slate-700"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      ))}
    </div>
  );
});

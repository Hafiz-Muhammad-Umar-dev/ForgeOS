interface SidebarItem {
  readonly label: string;
  readonly count?: number;
  readonly active?: boolean;
  readonly onClick?: () => void;
}

interface ConversationSidebarProps {
  readonly onItemClick?: (label: string) => void;
}

const sections: Array<{ title: string; items: SidebarItem[] }> = [
  {
    title: "Conversations",
    items: [
      { label: "New Chat" },
      { label: "Recent" },
    ],
  },
  {
    title: "Intents",
    items: [
      { label: "Active", count: 0 },
      { label: "Completed", count: 0 },
    ],
  },
  {
    title: "Workspaces",
    items: [
      { label: "Recent Workspaces", count: 0 },
      { label: "Running Tasks", count: 0 },
    ],
  },
  {
    title: "Deployments",
    items: [
      { label: "Deployments", count: 0 },
    ],
  },
];

export function ConversationSidebar({
  onItemClick,
}: ConversationSidebarProps): React.ReactNode {
  return (
    <nav className="flex h-full flex-col overflow-y-auto px-3 py-4">
      {sections.map((section) => (
        <div key={section.title} className="mb-6">
          <h3 className="mb-2 px-2 text-[10px] font-semibold uppercase tracking-widest text-slate-500">
            {section.title}
          </h3>
          <div className="space-y-0.5">
            {section.items.map((item) => (
              <button
                key={item.label}
                type="button"
                onClick={() => {
                  onItemClick?.(item.label);
                }}
                className={`flex w-full items-center justify-between rounded-lg px-2 py-1.5 text-left text-sm transition-colors ${
                  item.active === true
                    ? "bg-forge-800/50 text-white"
                    : "text-slate-400 hover:bg-slate-800 hover:text-slate-200"
                }`}
              >
                <span>{item.label}</span>
                {item.count !== undefined && (
                  <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[10px] text-slate-500">
                    {String(item.count)}
                  </span>
                )}
              </button>
            ))}
          </div>
        </div>
      ))}
    </nav>
  );
}

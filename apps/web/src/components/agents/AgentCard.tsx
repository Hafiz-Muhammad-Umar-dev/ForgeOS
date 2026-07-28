import type { AgentInfo } from "../../types/agent";
import { ExecutionStatusBadge } from "./ExecutionStatusBadge";

interface AgentCardProps {
  readonly agent: AgentInfo;
  readonly onSelect?: (id: string) => void;
}

export function AgentCard({ agent, onSelect }: AgentCardProps): React.ReactNode {
  return (
    <button
      type="button"
      onClick={() => onSelect?.(agent.id)}
      className="w-full rounded-lg border border-slate-800 bg-slate-900/30 p-3 text-left transition-colors hover:border-slate-700"
    >
      <div className="mb-2 flex items-center justify-between">
        <span className="text-sm font-medium text-white">{agent.name}</span>
        <ExecutionStatusBadge status={agent.status} />
      </div>
      <div className="space-y-1 text-xs text-slate-400">
        <div className="flex justify-between">
          <span>Role</span>
          <span className="text-slate-300">{agent.role}</span>
        </div>
        <div className="flex justify-between">
          <span>Queue</span>
          <span className="text-slate-300">{String(agent.queueLength)}</span>
        </div>
        {agent.currentTool !== undefined && (
          <div className="flex justify-between">
            <span>Tool</span>
            <span className="text-slate-300">{agent.currentTool}</span>
          </div>
        )}
        <div className="flex justify-between">
          <span>Tokens</span>
          <span className="text-slate-300">{String(agent.tokenUsage.total)}</span>
        </div>
      </div>
    </button>
  );
}

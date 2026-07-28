import type { AgentInfo } from "../../types/agent";

interface AgentInspectorProps {
  readonly agent: AgentInfo | null;
}

export function AgentInspector({ agent }: AgentInspectorProps): React.ReactNode {
  if (agent === null) {
    return (
      <div className="p-4 text-center">
        <p className="text-sm text-slate-500">Select an agent to inspect.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-2 text-[10px] font-semibold uppercase tracking-widest text-slate-500">
          {agent.name}
        </h3>
        <div className="space-y-2">
          <DetailRow label="Role" value={agent.role} />
          <DetailRow label="Model" value={agent.model} />
          <DetailRow label="Temperature" value={String(agent.temperature)} />
          <DetailRow label="Status" value={agent.status} />
          {agent.currentTool !== undefined && (
            <DetailRow label="Current Tool" value={agent.currentTool} />
          )}
          <DetailRow label="Tokens" value={String(agent.tokenUsage.total)} />
          <DetailRow label="Cost" value={`$${agent.cost.toFixed(4)}`} />
          <DetailRow label="Execution Time" value={`${String(Math.round(agent.executionTime / 1000))}s`} />
        </div>
      </div>

      {agent.reasoning !== undefined && (
        <div>
          <h4 className="mb-1 text-[10px] font-semibold uppercase tracking-widest text-slate-500">
            Reasoning
          </h4>
          <p className="text-xs text-slate-300">{agent.reasoning}</p>
        </div>
      )}

      {agent.output !== undefined && (
        <div>
          <h4 className="mb-1 text-[10px] font-semibold uppercase tracking-widest text-slate-500">
            Output
          </h4>
          <pre className="overflow-x-auto rounded bg-slate-800 p-2 font-mono text-xs text-slate-300">
            {agent.output}
          </pre>
        </div>
      )}
    </div>
  );
}

function DetailRow({ label, value }: { readonly label: string; readonly value: string }): React.ReactNode {
  return (
    <div className="flex justify-between rounded bg-slate-800/30 px-2 py-1">
      <span className="text-xs text-slate-400">{label}</span>
      <span className="text-xs font-medium text-slate-200">{value}</span>
    </div>
  );
}

import { useAgents } from "../../hooks/useAgents";
import { useExecution } from "../../hooks/useExecution";
import { useExecutionStream } from "../../hooks/useExecutionStream";
import { AgentCard } from "./AgentCard";
import { ExecutionGraph } from "./ExecutionGraph";
import { ExecutionMetrics } from "./ExecutionMetrics";
import { ExecutionConsole } from "./ExecutionConsole";
import { ExecutionToolbar } from "./ExecutionToolbar";
import { AgentInspector } from "./AgentInspector";
import { useState } from "react";

interface AgentDashboardProps {
  readonly intentId: string | undefined;
}

export function AgentDashboard({ intentId }: AgentDashboardProps): React.ReactNode {
  const { agents } = useAgents();
  const { plan, metrics, events, isRunning, run, pause, resume, stop } = useExecution(intentId);
  const { liveEvents } = useExecutionStream(intentId);
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null);
  const selectedAgent = agents.find((a) => a.id === selectedAgentId) ?? null;

  const allEvents = [...liveEvents, ...events].sort((a, b) => b.timestamp - a.timestamp).slice(0, 200);

  return (
    <div className="flex h-[calc(100vh-3.5rem)]">
      {/* Left — Agent list */}
      <aside className="hidden w-[260px] shrink-0 overflow-y-auto border-r border-slate-800 bg-slate-900/50 p-3 lg:block">
        <h3 className="mb-3 text-[10px] font-semibold uppercase tracking-widest text-slate-500">
          Agents
        </h3>
        <div className="space-y-2">
          {agents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              onSelect={setSelectedAgentId}
            />
          ))}
        </div>
      </aside>

      {/* Center — Graph + Console */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Toolbar */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-2">
          <ExecutionToolbar
            isRunning={isRunning}
            onRun={run}
            onPause={pause}
            onResume={resume}
            onStop={stop}
          />
          <div className="flex items-center gap-4 text-xs text-slate-500">
            {intentId !== undefined && (
              <>
                <span>Intent: {intentId.slice(0, 8)}</span>
                {plan !== null && <span>Status: {plan.status}</span>}
              </>
            )}
          </div>
        </div>

        {/* Metrics */}
        <div className="border-b border-slate-800 px-4 py-3">
          <ExecutionMetrics metrics={metrics} />
        </div>

        {/* Graph */}
        <div className="flex-1 overflow-hidden">
          <ExecutionGraph plan={plan} />
        </div>

        {/* Console */}
        <div className="h-[180px] shrink-0 border-t border-slate-800">
          <ExecutionConsole events={allEvents} />
        </div>
      </div>

      {/* Right — Inspector */}
      <aside className="hidden w-[280px] shrink-0 overflow-y-auto border-l border-slate-800 bg-slate-900/50 p-4 lg:block">
        <AgentInspector agent={selectedAgent} />
      </aside>
    </div>
  );
}

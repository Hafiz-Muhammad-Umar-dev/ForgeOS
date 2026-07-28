import { useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { PageContainer } from "../layout/PageContainer";
import { useTasks } from "../hooks/useTasks";
import { LoadingState } from "../components/LoadingState";
import { ErrorState } from "../components/ErrorState";
import { EmptyState } from "../components/EmptyState";
import type { TaskView } from "../types/task";

const statusColors: Record<string, string> = {
  completed: "text-green-400",
  failed: "text-red-400",
  running: "text-blue-400",
  pending: "text-yellow-400",
};

function statusColor(status: string): string {
  return statusColors[status] ?? "text-slate-400";
}

export function TaskBoard(): React.ReactNode {
  const [params] = useSearchParams();
  const intentId = params.get("intentId") ?? undefined;
  const { data: tasks, isLoading, isError, error, refetch } = useTasks(intentId);
  const handleRetry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return (
    <PageContainer
      title="Tasks"
      description={
        intentId !== undefined
          ? `Tasks for intent ${intentId}`
          : "Track agent task execution across all projects."
      }
    >
      {isLoading && <LoadingState count={4} />}
      {isError && (
        <ErrorState
          message={error.message}
          onRetry={handleRetry}
        />
      )}
      {!isLoading && !isError && tasks !== undefined && tasks.length === 0 && (
        <EmptyState message="No tasks found." />
      )}
      {!isLoading && !isError && tasks !== undefined && tasks.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-slate-800">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-800 bg-slate-900/80">
              <tr>
                <th className="px-4 py-3 font-medium text-slate-400">ID</th>
                <th className="px-4 py-3 font-medium text-slate-400">Agent</th>
                <th className="px-4 py-3 font-medium text-slate-400">Status</th>
                <th className="px-4 py-3 font-medium text-slate-400">Summary</th>
                <th className="px-4 py-3 font-medium text-slate-400">Tokens</th>
                <th className="px-4 py-3 font-medium text-slate-400">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {tasks.map((task: TaskView) => (
                <tr key={task.id} className="hover:bg-slate-900/30">
                  <td className="px-4 py-3 font-mono text-xs text-white">
                    {task.id.slice(0, 8)}
                  </td>
                  <td className="px-4 py-3 text-white">{task.agent_name ?? "—"}</td>
                  <td className={`px-4 py-3 font-medium ${statusColor(task.status)}`}>
                    {task.status}
                  </td>
                  <td className="max-w-xs truncate px-4 py-3 text-slate-300">
                    {task.summary ?? "—"}
                  </td>
                  <td className="px-4 py-3 text-slate-300">
                    {String(task.input_tokens + task.output_tokens)}
                  </td>
                  <td className="px-4 py-3 text-slate-400">
                    {new Date(task.created_at).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </PageContainer>
  );
}

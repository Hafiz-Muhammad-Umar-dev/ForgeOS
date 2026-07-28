import { useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { PageContainer } from "../layout/PageContainer";
import { useIntent } from "../hooks/useIntent";
import { useIntentStream } from "../hooks/useIntentStream";
import { LoadingState } from "../components/LoadingState";
import { ErrorState } from "../components/ErrorState";
import { EmptyState } from "../components/EmptyState";
import { StreamingConsole } from "../components/StreamingConsole";

export function IntentView(): React.ReactNode {
  const [params] = useSearchParams();
  const intentId = params.get("id") ?? undefined;
  const { data: intent, isLoading, isError, error, refetch } = useIntent(intentId);
  const { events, connectionState, clearEvents } = useIntentStream({ intentId });

  const handleRetry = useCallback(() => {
    clearEvents();
    void refetch();
  }, [refetch, clearEvents]);

  return (
    <PageContainer
      title={intentId !== undefined ? `Intent ${intentId}` : "Intents"}
      description="View and manage user intents."
    >
      {intentId === undefined && (
        <EmptyState message="Select an intent from the dashboard to view details." />
      )}
      {isLoading && <LoadingState count={1} />}
      {isError && (
        <ErrorState
          message={error.message}
          onRetry={handleRetry}
        />
      )}
      {!isLoading && !isError && intent !== undefined && (
        <div className="grid gap-6 lg:grid-cols-3">
          {/* Intent detail panel */}
          <div className="lg:col-span-2">
            <div className="rounded-xl border border-slate-800 bg-slate-900/50 p-6">
              <dl className="grid gap-4 sm:grid-cols-2">
                <div>
                  <dt className="text-xs font-medium uppercase tracking-wider text-slate-500">ID</dt>
                  <dd className="mt-1 text-sm text-white">{intent.id}</dd>
                </div>
                <div>
                  <dt className="text-xs font-medium uppercase tracking-wider text-slate-500">Status</dt>
                  <dd className="mt-1 text-sm text-white">{intent.status}</dd>
                </div>
                <div>
                  <dt className="text-xs font-medium uppercase tracking-wider text-slate-500">Text</dt>
                  <dd className="mt-1 text-sm text-white">{intent.text ?? "—"}</dd>
                </div>
                <div>
                  <dt className="text-xs font-medium uppercase tracking-wider text-slate-500">Summary</dt>
                  <dd className="mt-1 text-sm text-white">{intent.summary ?? "—"}</dd>
                </div>
                <div>
                  <dt className="text-xs font-medium uppercase tracking-wider text-slate-500">Created</dt>
                  <dd className="mt-1 text-sm text-white">
                    {new Date(intent.created_at).toLocaleString()}
                  </dd>
                </div>
                <div>
                  <dt className="text-xs font-medium uppercase tracking-wider text-slate-500">Updated</dt>
                  <dd className="mt-1 text-sm text-white">
                    {new Date(intent.updated_at).toLocaleString()}
                  </dd>
                </div>
              </dl>
              {intent.error !== undefined && intent.error !== "" && (
                <div className="mt-4 rounded-lg bg-red-950/20 p-4">
                  <p className="text-sm font-medium text-red-400">Error</p>
                  <p className="mt-1 text-sm text-red-300">{intent.error}</p>
                </div>
              )}
            </div>
          </div>

          {/* Live stream console */}
          <div className="lg:col-span-1">
            <StreamingConsole events={events} connectionState={connectionState} />
          </div>
        </div>
      )}
    </PageContainer>
  );
}

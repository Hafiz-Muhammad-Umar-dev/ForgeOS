import { useCallback } from "react";
import { PageContainer } from "../layout/PageContainer";
import { useIntents } from "../hooks/useIntents";
import { IntentCard } from "../components/IntentCard";
import { LoadingState } from "../components/LoadingState";
import { ErrorState } from "../components/ErrorState";
import { EmptyState } from "../components/EmptyState";

export function Dashboard(): React.ReactNode {
  const { data: intents, isLoading, isError, error, refetch } = useIntents();
  const handleRetry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return (
    <PageContainer
      title="Dashboard"
      description="Overview of active intents, tasks, and system status."
    >
      {isLoading && <LoadingState count={3} />}
      {isError && (
        <ErrorState
          message={error.message}
          onRetry={handleRetry}
        />
      )}
      {!isLoading && !isError && intents !== undefined && intents.length === 0 && (
        <EmptyState message="No intents yet. Create your first intent to get started." />
      )}
      {!isLoading && !isError && intents !== undefined && intents.length > 0 && (
        <section>
          <h2 className="mb-4 text-lg font-semibold text-white">Recent Intents</h2>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {intents.map((intent) => (
              <IntentCard key={intent.id} intent={intent} />
            ))}
          </div>
        </section>
      )}
    </PageContainer>
  );
}

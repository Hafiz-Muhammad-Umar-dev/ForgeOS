import { PageContainer } from "../layout/PageContainer";

export function DeploymentView(): React.ReactNode {
  return (
    <PageContainer
      title="Deployments"
      description="Review deployment history and current status."
    >
      <div className="rounded-xl border border-slate-800 bg-slate-900/50 p-8 text-center">
        <p className="text-sm text-slate-400">
          Deployment list will be displayed here.
        </p>
      </div>
    </PageContainer>
  );
}

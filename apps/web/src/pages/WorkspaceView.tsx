import { PageContainer } from "../layout/PageContainer";

export function WorkspaceView(): React.ReactNode {
  return (
    <PageContainer
      title="Workspaces"
      description="Monitor active workspaces and their resource usage."
    >
      <div className="rounded-xl border border-slate-800 bg-slate-900/50 p-8 text-center">
        <p className="text-sm text-slate-400">
          Workspace list will be displayed here.
        </p>
      </div>
    </PageContainer>
  );
}

import { useSearchParams } from "react-router-dom";
import { AgentDashboard } from "../components/agents/AgentDashboard";

export function AgentDashboardPage(): React.ReactNode {
  const [params] = useSearchParams();
  const intentId = params.get("intentId") ?? undefined;

  return <AgentDashboard intentId={intentId} />;
}

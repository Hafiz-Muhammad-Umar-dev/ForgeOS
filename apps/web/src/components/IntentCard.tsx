import { Link } from "react-router-dom";
import type { IntentView } from "../types/intent";

interface IntentCardProps {
  readonly intent: IntentView;
}

const statusColors: Record<string, string> = {
  completed: "text-green-400",
  failed: "text-red-400",
  running: "text-blue-400",
  pending: "text-yellow-400",
};

function statusColor(status: string): string {
  return statusColors[status] ?? "text-slate-400";
}

export function IntentCard({ intent }: IntentCardProps): React.ReactNode {
  return (
    <Link
      to={`/intents?id=${intent.id}`}
      className="block rounded-xl border border-slate-800 bg-slate-900/50 p-6 transition-colors hover:border-slate-700"
    >
      <div className="mb-2 flex items-center justify-between">
        <span className="text-xs font-medium uppercase tracking-wider text-slate-500">
          Intent
        </span>
        <span className={`text-xs font-semibold ${statusColor(intent.status)}`}>
          {intent.status}
        </span>
      </div>
      <p className="mb-1 truncate text-sm font-medium text-white">
        {intent.text ?? intent.summary ?? intent.id}
      </p>
      <p className="text-xs text-slate-500">
        {new Date(intent.created_at).toLocaleString()}
      </p>
    </Link>
  );
}

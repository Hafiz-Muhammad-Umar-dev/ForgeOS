import { Play, Pause, Square, RotateCcw } from "lucide-react";

interface ExecutionToolbarProps {
  readonly isRunning: boolean;
  readonly onRun: () => void;
  readonly onPause: () => void;
  readonly onResume: () => void;
  readonly onStop: () => void;
}

export function ExecutionToolbar({
  isRunning,
  onRun,
  onPause,
  onResume,
  onStop,
}: ExecutionToolbarProps): React.ReactNode {
  return (
    <div className="flex items-center gap-2">
      {!isRunning ? (
        <ToolbarButton icon={<Play className="h-3.5 w-3.5" />} label="Run" onClick={onRun} />
      ) : (
        <>
          <ToolbarButton icon={<Pause className="h-3.5 w-3.5" />} label="Pause" onClick={onPause} />
          <ToolbarButton icon={<RotateCcw className="h-3.5 w-3.5" />} label="Resume" onClick={onResume} />
          <ToolbarButton icon={<Square className="h-3.5 w-3.5" />} label="Stop" onClick={onStop} variant="danger" />
        </>
      )}
    </div>
  );
}

interface ToolbarButtonProps {
  readonly icon: React.ReactNode;
  readonly label: string;
  readonly onClick: () => void;
  readonly variant?: "default" | "danger";
}

function ToolbarButton({
  icon,
  label,
  onClick,
  variant = "default",
}: ToolbarButtonProps): React.ReactNode {
  const hoverClass = variant === "danger"
    ? "hover:bg-red-950/30 hover:text-red-400"
    : "hover:bg-slate-800 hover:text-slate-200";

  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium text-slate-400 transition-colors ${hoverClass}`}
    >
      {icon}
      {label}
    </button>
  );
}

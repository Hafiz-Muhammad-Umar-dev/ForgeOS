import { useTerminal } from "../../hooks/useTerminal";

export function TerminalPanel(): React.ReactNode {
  const { terminalRef, isConnected, clear, backendAvailable } = useTerminal();

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-slate-800 px-3 py-1.5">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">
          Terminal
        </span>
        <div className="flex items-center gap-2">
          <span
            className={`h-1.5 w-1.5 rounded-full ${
              isConnected ? "bg-green-500" : "bg-red-500"
            }`}
          />
          <button
            type="button"
            onClick={clear}
            className="rounded px-1.5 py-0.5 text-xs text-slate-500 hover:bg-slate-800 hover:text-slate-300"
          >
            Clear
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-hidden">
        {!backendAvailable && (
          <div className="flex h-full items-center justify-center">
            <p className="text-sm text-slate-500">Terminal backend unavailable</p>
          </div>
        )}
        <div
          ref={terminalRef}
          className="h-full w-full"
          style={{ visibility: backendAvailable ? "visible" : "hidden" }}
        />
      </div>
    </div>
  );
}

import { useState } from "react";
import type { EnvVariable } from "../../types/deployment";

interface EnvironmentEditorProps {
  readonly variables: EnvVariable[];
  readonly onAdd: (key: string, value: string, isSecret: boolean) => void;
  readonly onDelete: (key: string) => void;
}

export function EnvironmentEditor({ variables, onAdd, onDelete }: EnvironmentEditorProps): React.ReactNode {
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const [isSecret, setIsSecret] = useState(false);

  function handleAdd(): void {
    if (key.trim().length === 0) return;
    onAdd(key.trim(), value, isSecret);
    setKey("");
    setValue("");
    setIsSecret(false);
  }

  return (
    <div className="space-y-3">
      <h4 className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">Environment</h4>
      <div className="space-y-1">
        {variables.map((v) => (
          <div key={v.key} className="flex items-center gap-2 rounded bg-slate-800/30 px-2 py-1">
            <span className="flex-1 text-xs font-medium text-slate-200">{v.key}</span>
            <span className="flex-1 text-xs text-slate-400">
              {v.isSecret ? "••••••••" : v.value}
            </span>
            <button
              type="button"
              onClick={() => { onDelete(v.key); }}
              className="text-[10px] text-red-500 hover:text-red-400"
            >
              del
            </button>
          </div>
        ))}
      </div>
      <div className="flex gap-2">
        <input value={key} onChange={(e) => { setKey(e.target.value); }} placeholder="KEY" className="w-1/3 rounded border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-white" />
        <input value={value} onChange={(e) => { setValue(e.target.value); }} placeholder="value" className="flex-1 rounded border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-white" />
      </div>
      <label className="flex items-center gap-2 text-xs text-slate-400">
        <input type="checkbox" checked={isSecret} onChange={(e) => { setIsSecret(e.target.checked); }} />
        Secret (masked in UI)
      </label>
      <button type="button" onClick={handleAdd} disabled={key.trim().length === 0}
        className="w-full rounded bg-forge-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
      >
        Add Variable
      </button>
    </div>
  );
}

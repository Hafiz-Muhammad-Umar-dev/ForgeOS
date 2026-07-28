import { useState, type FormEvent } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

interface LocationState {
  readonly from?: { readonly pathname: string };
}

export function Login(): React.ReactNode {
  const { login, isAuthenticated, isLoading, error } = useAuth();
  const location = useLocation();
  const state = location.state as LocationState | null;
  const from = state?.from?.pathname ?? "/";

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [remember, setRemember] = useState(false);

  const isValid = username.trim().length > 0 && password.length > 0;

  if (isAuthenticated && !isLoading) {
    return <Navigate to={from} replace />;
  }

  async function handleSubmit(e: FormEvent): Promise<void> {
    e.preventDefault();
    if (!isValid || isSubmitting) return;

    setIsSubmitting(true);
    setSubmitError(null);

    try {
      await login(username.trim(), password);
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Login failed. Please try again.";
      setSubmitError(message);
    } finally {
      setIsSubmitting(false);
    }
  }

  const displayError = submitError ?? error;

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-950 px-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-bold text-white">ForgeOS</h1>
          <p className="mt-1 text-sm text-slate-400">
            Sign in to your development workspace
          </p>
        </div>

        <form
          onSubmit={(e) => {
            handleSubmit(e).catch(() => undefined);
          }}
          className="rounded-xl border border-slate-800 bg-slate-900/50 p-6"
        >
          <div className="mb-4">
            <label
              htmlFor="username"
              className="mb-1.5 block text-sm font-medium text-slate-300"
            >
              Username
            </label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => { setUsername(e.target.value); }}
              placeholder="Enter your username"
              autoComplete="username"
              className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-white placeholder-slate-500 transition-colors focus:border-forge-500 focus:outline-none focus:ring-1 focus:ring-forge-500"
            />
          </div>

          <div className="mb-4">
            <label
              htmlFor="password"
              className="mb-1.5 block text-sm font-medium text-slate-300"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => { setPassword(e.target.value); }}
              placeholder="Enter your password"
              autoComplete="current-password"
              className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-white placeholder-slate-500 transition-colors focus:border-forge-500 focus:outline-none focus:ring-1 focus:ring-forge-500"
            />
          </div>

          <div className="mb-4 flex items-center gap-2">
            <input
              id="remember"
              type="checkbox"
              checked={remember}
              onChange={(e) => { setRemember(e.target.checked); }}
              className="rounded border-slate-700 bg-slate-800 text-forge-500 focus:ring-forge-500"
            />
            <label htmlFor="remember" className="text-xs text-slate-400">
              Remember me
            </label>
          </div>

          {displayError !== null && (
            <div className="mb-4 rounded-lg bg-red-950/20 p-3">
              <p className="text-xs text-red-400">{displayError}</p>
            </div>
          )}

          <button
            type="submit"
            disabled={!isValid || isSubmitting}
            className="w-full rounded-lg bg-forge-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-forge-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSubmitting ? "Signing in..." : "Sign in"}
          </button>

          <p className="mt-4 text-center text-[10px] text-slate-600">
            Development credentials: admin / admin123
          </p>
        </form>
      </div>
    </div>
  );
}

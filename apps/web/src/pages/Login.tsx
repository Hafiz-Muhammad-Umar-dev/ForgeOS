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

  const [token, setToken] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [touched, setTouched] = useState(false);

  const isValid = token.trim().length > 0;

  // Already authenticated — redirect.
  if (isAuthenticated && !isLoading) {
    return <Navigate to={from} replace />;
  }

  async function handleSubmit(e: FormEvent): Promise<void> {
    e.preventDefault();
    if (!isValid || isSubmitting) return;

    setIsSubmitting(true);
    setSubmitError(null);

    try {
      await login(token.trim());
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
            Sign in with your API token
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
              htmlFor="token"
              className="mb-1.5 block text-sm font-medium text-slate-300"
            >
              API Token
            </label>
            <input
              id="token"
              type="password"
              value={token}
              onChange={(e) => {
                setToken(e.target.value);
                setTouched(true);
              }}
              placeholder="Enter your API token"
              autoComplete="off"
              className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-white placeholder-slate-500 transition-colors focus:border-forge-500 focus:outline-none focus:ring-1 focus:ring-forge-500"
            />
            {touched && !isValid && (
              <p className="mt-1 text-xs text-red-400">
                Token is required.
              </p>
            )}
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
        </form>
      </div>
    </div>
  );
}

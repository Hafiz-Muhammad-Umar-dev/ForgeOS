import { useEffect } from "react";
import { useAuth } from "../hooks/useAuth";

interface LoginGateProps {
  readonly children: React.ReactNode;
}

export function LoginGate({ children }: LoginGateProps): React.ReactNode {
  const { refreshUser, logout, isLoading } = useAuth();

  useEffect(() => {
    const storedToken = sessionStorage.getItem("auth_token");
    if (storedToken !== null) {
      // A token was persisted — validate it.
      refreshUser().catch(() => {
        logout().catch(() => undefined);
      });
    } else {
      // No token — mark as not loading so the UI can show the login page.
      logout().catch(() => undefined);
    }
    // Run once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-950">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-forge-500" />
      </div>
    );
  }

  return <>{children}</>;
}

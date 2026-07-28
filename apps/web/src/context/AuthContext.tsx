import {
  createContext,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  fetchMe,
  login as apiLogin,
  logout as apiLogout,
  type AuthUser,
} from "../lib/auth";
import { setAuthToken, onUnauthorized } from "../lib/api";

export interface AuthState {
  readonly user: AuthUser | null;
  readonly token: string | null;
  readonly isLoading: boolean;
  readonly isAuthenticated: boolean;
  readonly error: string | null;
}

export interface AuthContextValue extends AuthState {
  readonly login: (token: string) => Promise<void>;
  readonly logout: () => Promise<void>;
  readonly refreshUser: () => Promise<void>;
}

const initialState: AuthState = {
  user: null,
  token: null,
  isLoading: true,
  isAuthenticated: false,
  error: null,
};

export const AuthContext = createContext<AuthContextValue>({
  ...initialState,
  login: () => Promise.resolve(),
  logout: () => Promise.resolve(),
  refreshUser: () => Promise.resolve(),
});

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

interface AuthProviderProps {
  readonly children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps): React.ReactNode {
  const [state, setState] = useState<AuthState>(initialState);
  const tokenRef = useRef<string | null>(null);

  // -----------------------------------------------------------------------
  // Bootstrap: try restoring a saved session.
  // -----------------------------------------------------------------------
  useEffect(() => {
    const savedToken = sessionStorage.getItem("auth_token");
    if (savedToken !== null) {
      tokenRef.current = savedToken;
      setAuthToken(savedToken);

      fetchMe(savedToken)
        .then((user) => {
          setState({
            user,
            token: savedToken,
            isLoading: false,
            isAuthenticated: true,
            error: null,
          });
        })
        .catch(() => {
          sessionStorage.removeItem("auth_token");
          tokenRef.current = null;
          setAuthToken(null);
          setState({
            user: null,
            token: null,
            isLoading: false,
            isAuthenticated: false,
            error: null,
          });
        });
    } else {
      setState((prev) => ({ ...prev, isLoading: false }));
    }
  }, []);

  // -----------------------------------------------------------------------
  // Register the 401/403 handler once.
  // -----------------------------------------------------------------------
  useEffect(() => {
    onUnauthorized(() => {
      if (tokenRef.current !== null) {
        sessionStorage.removeItem("auth_token");
        tokenRef.current = null;
        setAuthToken(null);
        setState({
          user: null,
          token: null,
          isLoading: false,
          isAuthenticated: false,
          error: "Session expired.",
        });
      }
    });
  }, []);

  // -----------------------------------------------------------------------
  // Login
  // -----------------------------------------------------------------------
  const login = useCallback(async (rawToken: string) => {
    setState((prev) => ({ ...prev, isLoading: true, error: null }));

    try {
      let token = rawToken;
      let user: AuthUser;

      try {
        const loginResp = await apiLogin({ token: rawToken });
        token = loginResp.token;
        user = loginResp.user;
      } catch {
        // Login endpoint may not exist (404). Fall back to treating the
        // raw token as a Bearer token and verifying it.
        user = await fetchMe(rawToken);
        token = rawToken;
      }

      tokenRef.current = token;
      setAuthToken(token);
      sessionStorage.setItem("auth_token", token);
      setState({
        user,
        token,
        isLoading: false,
        isAuthenticated: true,
        error: null,
      });
    } catch (err: unknown) {
      tokenRef.current = null;
      setAuthToken(null);
      const message =
        err instanceof Error ? err.message : "Authentication failed.";
      setState({
        user: null,
        token: null,
        isLoading: false,
        isAuthenticated: false,
        error: message,
      });
    }
  }, []);

  // -----------------------------------------------------------------------
  // Logout
  // -----------------------------------------------------------------------
  const logout = useCallback(async () => {
    const currentToken = tokenRef.current;
    tokenRef.current = null;
    setAuthToken(null);
    sessionStorage.removeItem("auth_token");
    setState({
      user: null,
      token: null,
      isLoading: false,
      isAuthenticated: false,
      error: null,
    });

    if (currentToken !== null) {
      try {
        await apiLogout(currentToken);
      } catch {
        // Best-effort.
      }
    }
  }, []);

  // -----------------------------------------------------------------------
  // Refresh user
  // -----------------------------------------------------------------------
  const refreshUser = useCallback(async () => {
    const currentToken = tokenRef.current;
    if (currentToken === null) return;

    try {
      const user = await fetchMe(currentToken);
      setState((prev) => ({ ...prev, user }));
    } catch {
      tokenRef.current = null;
      setAuthToken(null);
      sessionStorage.removeItem("auth_token");
      setState({
        user: null,
        token: null,
        isLoading: false,
        isAuthenticated: false,
        error: "Session expired.",
      });
    }
  }, []);

  // -----------------------------------------------------------------------
  // Value
  // -----------------------------------------------------------------------
  const value = useMemo<AuthContextValue>(
    () => ({
      user: state.user,
      token: state.token,
      isLoading: state.isLoading,
      isAuthenticated: state.isAuthenticated,
      error: state.error,
      login,
      logout,
      refreshUser,
    }),
    [state, login, logout, refreshUser],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

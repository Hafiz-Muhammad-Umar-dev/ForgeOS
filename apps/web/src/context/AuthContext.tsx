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
  getStoredToken,
  storeToken,
  clearToken,
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
  readonly login: (username: string, password: string) => Promise<void>;
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

  // Bootstrap: restore saved token.
  useEffect(() => {
    const savedToken = getStoredToken();
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
          clearToken();
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

  // Register 401/403 handler.
  useEffect(() => {
    onUnauthorized(() => {
      if (tokenRef.current !== null) {
        clearToken();
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

  // Login with username/password.
  const login = useCallback(async (username: string, password: string) => {
    setState((prev) => ({ ...prev, isLoading: true, error: null }));

    try {
      const resp = await apiLogin({ username, password });

      tokenRef.current = resp.access_token;
      setAuthToken(resp.access_token);
      storeToken(resp.access_token);
      setState({
        user: resp.user,
        token: resp.access_token,
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

  // Logout.
  const logout = useCallback(async () => {
    const currentToken = tokenRef.current;
    tokenRef.current = null;
    setAuthToken(null);
    clearToken();
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

  // Refresh user.
  const refreshUser = useCallback(async () => {
    const currentToken = tokenRef.current;
    if (currentToken === null) return;

    try {
      const user = await fetchMe(currentToken);
      setState((prev) => ({ ...prev, user }));
    } catch {
      clearToken();
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
  }, []);

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

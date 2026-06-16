import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { logoutUser, getProfile } from "@/lib/api/auth";
import {
  clearStoredSession,
  getStoredSession,
  setStoredSession,
  SESSION_STORAGE_KEY,
  type AuthUser,
  type Session,
} from "@/lib/session";

type AuthContextValue = {
  session: Session | null;
  user: AuthUser | null;
  isAuthenticated: boolean;
  isHydrated: boolean;
  setSession: (s: Session | null) => void;
  refreshProfile: () => Promise<AuthUser | null>;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSessionState] = useState<Session | null>(null);
  const [isHydrated, setIsHydrated] = useState(false);

  useEffect(() => {
    setSessionState(getStoredSession());
    setIsHydrated(true);

    const onStorage = (e: StorageEvent) => {
      if (e.key && e.key !== SESSION_STORAGE_KEY) return;
      setSessionState(getStoredSession());
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  const setSession = useCallback((next: Session | null) => {
    if (next) setStoredSession(next);
    else clearStoredSession();
    setSessionState(next);
  }, []);

  const refreshProfile = useCallback(async () => {
    const current = getStoredSession();
    if (!current) return null;
    try {
      const user = await getProfile();
      const updated = { ...current, user };
      setStoredSession(updated);
      setSessionState(updated);
      return user;
    } catch {
      return null;
    }
  }, []);

  const signOut = useCallback(async () => {
    const current = getStoredSession();
    if (current?.refreshToken) {
      await logoutUser(current.refreshToken);
    }
    clearStoredSession();
    setSessionState(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      session,
      user: session?.user ?? null,
      isAuthenticated: Boolean(session?.accessToken),
      isHydrated,
      setSession,
      refreshProfile,
      signOut,
    }),
    [session, isHydrated, setSession, refreshProfile, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}

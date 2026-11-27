import { useQueryClient } from "@tanstack/react-query";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  useRef,
  type ReactNode,
} from "react";

import { logoutUser, getProfile } from "@/lib/api/auth";
import { refreshSession } from "@/lib/api/client";
import {
  clearStoredSession,
  getStoredSession,
  setStoredSession,
  SESSION_STORAGE_KEY,
  type AuthUser,
  type Session,
} from "@/lib/session";

function portalHomeForRole(role: AuthUser["role"]): string {
  if (role === "admin") return "/admin";
  if (role === "teacher") return "/teacher";
  return "/student";
}

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
  const queryClient = useQueryClient();

  const setSession = useCallback(
    (next: Session | null) => {
      if (next) setStoredSession(next);
      else clearStoredSession();
      setSessionState(next);
      queryClient.clear();
    },
    [queryClient],
  );

  const refreshProfile = useCallback(async () => {
    const current = getStoredSession();
    if (!current) return null;
    try {
      const user = await getProfile();
      const roleChanged = user.role !== current.user.role;
      let baseSession = current;
      if (roleChanged) {
        // Re-issue the JWT so backend role claims immediately match the DB role.
        const refreshed = await refreshSession();
        if (refreshed) baseSession = refreshed;
      }
      const updated = { ...baseSession, user };
      if (roleChanged && typeof window !== "undefined") {
        setStoredSession(updated, { silent: true });
        const path = window.location.pathname;
        const onRolePortal =
          path.startsWith("/admin") || path.startsWith("/teacher") || path.startsWith("/student");
        if (onRolePortal) {
          window.location.assign(portalHomeForRole(user.role));
          return user;
        }
      }
      setStoredSession(updated);
      setSessionState(updated);
      return user;
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    const stored = getStoredSession();
    setSessionState(stored);
    setIsHydrated(true);
    if (stored?.accessToken) {
      void refreshProfile();
    }

    const onStorage = (e: StorageEvent) => {
      if (e.key && e.key !== SESSION_STORAGE_KEY) return;
      setSessionState(getStoredSession());
      queryClient.clear();
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, [queryClient, refreshProfile]);

  const signOut = useCallback(async () => {
    const current = getStoredSession();
    if (current?.refreshToken) {
      await logoutUser(current.refreshToken);
    }
    clearStoredSession();
    setSessionState(null);
    queryClient.clear();
  }, [queryClient]);

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

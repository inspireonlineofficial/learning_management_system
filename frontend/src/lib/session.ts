export type AuthUser = {
  id: string;
  email: string;
  full_name: string;
  role: "student" | "teacher" | "admin";
  onboarded?: boolean;
  profile_complete?: boolean;
  avatar_url?: string | null;
};

export type Session = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

const STORAGE_KEY = "inspire.session";

export function getStoredSession(): Session | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as Session;
  } catch {
    return null;
  }
}

export function setStoredSession(session: Session) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
  window.dispatchEvent(new StorageEvent("storage", { key: STORAGE_KEY }));
}

export function clearStoredSession() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(STORAGE_KEY);
  window.dispatchEvent(new StorageEvent("storage", { key: STORAGE_KEY }));
}

export const SESSION_STORAGE_KEY = STORAGE_KEY;

// ---------- Impersonation ----------

const IMPERSONATION_KEY = "inspire.impersonation.origin";

export function startImpersonation(impersonated: Session) {
  if (typeof window === "undefined") return;
  const current = getStoredSession();
  if (current && !isImpersonating()) {
    window.localStorage.setItem(IMPERSONATION_KEY, JSON.stringify(current));
  }
  setStoredSession(impersonated);
}

export function stopImpersonation(): Session | null {
  if (typeof window === "undefined") return null;
  const raw = window.localStorage.getItem(IMPERSONATION_KEY);
  if (!raw) return null;
  try {
    const origin = JSON.parse(raw) as Session;
    window.localStorage.removeItem(IMPERSONATION_KEY);
    setStoredSession(origin);
    return origin;
  } catch {
    window.localStorage.removeItem(IMPERSONATION_KEY);
    return null;
  }
}

export function isImpersonating(): boolean {
  if (typeof window === "undefined") return false;
  return Boolean(window.localStorage.getItem(IMPERSONATION_KEY));
}

export function getImpersonationOrigin(): Session | null {
  if (typeof window === "undefined") return null;
  const raw = window.localStorage.getItem(IMPERSONATION_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as Session;
  } catch {
    return null;
  }
}

import { redirect } from "@tanstack/react-router";

import { getStoredSession } from "@/lib/session";

export type Role = "student" | "teacher" | "admin";

/**
 * Throw inside a `beforeLoad` to enforce role-based access.
 * Unauthenticated users are sent to /login. Wrong-role users are sent to /403.
 */
export function requireRole(allowed: Role | Role[], returnHref?: string): void {
  const session = getStoredSession();
  if (!session) {
    throw redirect({ to: "/login", search: returnHref ? { return: returnHref } : undefined });
  }
  const roles = Array.isArray(allowed) ? allowed : [allowed];
  if (!roles.includes(session.user.role)) {
    throw redirect({ to: "/403" });
  }
}

export function hasRole(role: Role | Role[]): boolean {
  const session = getStoredSession();
  if (!session) return false;
  const roles = Array.isArray(role) ? role : [role];
  return roles.includes(session.user.role);
}

import { Link } from "@tanstack/react-router";

import { useAuth } from "@/context/auth-context";

/**
 * PublicHeaderActions renders the right-hand side of a public-page header.
 *
 * When the visitor is signed in, it shows a compact user pill with their
 * initials and a "Dashboard" link that points to the role-appropriate home
 * (admin → /admin, teacher → /teacher, student → /student). When signed out,
 * it shows a "Sign in" text link and a "Register" CTA button.
 *
 * Use this anywhere we previously had an unconditional "Sign in" link so the
 * header stays consistent across public pages.
 */
export function PublicHeaderActions({ compact = false }: { compact?: boolean }) {
  const { isAuthenticated, isHydrated, user } = useAuth();
  const dashboardTo =
    user?.role === "admin" ? "/admin" : user?.role === "teacher" ? "/teacher" : "/student";

  // Before hydration we don't know who the visitor is. Render the signed-out
  // shape to avoid showing a flash of "Dashboard" to a logged-out user.
  if (!isHydrated || !isAuthenticated) {
    return (
      <div
        className={
          compact
            ? "flex flex-nowrap items-center gap-2 text-xs"
            : "flex flex-wrap items-center gap-x-4 gap-y-2"
        }
      >
        <Link
          to="/login"
          className="whitespace-nowrap text-brand/60 transition-colors hover:text-brand"
        >
          Sign in
        </Link>
        <Link
          to="/register"
          className={
            compact
              ? "whitespace-nowrap bg-brand px-3 py-2 text-white transition-colors hover:bg-brand/90"
              : "whitespace-nowrap bg-brand px-4 py-2.5 text-white transition-colors hover:bg-brand/90"
          }
        >
          Register
        </Link>
      </div>
    );
  }

  const initials = (user?.full_name ?? user?.email ?? "?")
    .split(/\s+/)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("")
    .slice(0, 2);
  const displayName = user?.full_name?.trim() || user?.email || "Account";

  return (
    <Link
      to={dashboardTo}
      className={`inline-flex items-center gap-2 py-1 transition-colors hover:bg-brand/[0.03] ${
        compact ? "px-1" : "px-2"
      }`}
      aria-label={`Open ${displayName}'s dashboard`}
    >
      <span className="h-9 w-9 grid place-items-center rounded-full bg-brand text-white text-xs font-semibold">
        {initials || "U"}
      </span>
      <span className="hidden sm:inline text-sm font-medium text-brand max-w-[160px] truncate">
        {displayName}
      </span>
    </Link>
  );
}

import { Link } from "@tanstack/react-router";

import { useAuth } from "@/context/auth-context";

/**
 * PublicHeaderActions renders the right-hand side of a public-page header.
 *
 * When the visitor is signed in, it shows a compact user pill with their
 * initials and a "Dashboard" link that points to the role-appropriate home
 * (admin → /admin, teacher → /teacher, student → /student). When signed out,
 * it shows a "Sign in" button and a "Register" CTA button.
 */
export function PublicHeaderActions({
  compact = false,
  mobile = false,
  onNavigate,
}: {
  compact?: boolean;
  mobile?: boolean;
  onNavigate?: () => void;
}) {
  const { isAuthenticated, isHydrated, user } = useAuth();
  const dashboardTo =
    user?.role === "admin" ? "/admin" : user?.role === "teacher" ? "/teacher" : "/student";

  // Before hydration we don't know who the visitor is. Render the signed-out
  // shape to avoid showing a flash of "Dashboard" to a logged-out user.
  if (!isHydrated || !isAuthenticated) {
    if (mobile) {
      return (
        <div className="flex flex-col sm:flex-row gap-2.5">
          <Link
            to="/login"
            onClick={onNavigate}
            className="flex-1 inline-flex items-center justify-center min-h-[44px] px-4 rounded-lg border border-borderui text-sm font-semibold text-navy hover:bg-stoneband transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
          >
            Sign in
          </Link>
          <Link
            to="/register"
            onClick={onNavigate}
            className="flex-1 inline-flex items-center justify-center min-h-[44px] px-4 rounded-lg bg-navy text-sm font-semibold text-white hover:bg-navy-light shadow-sm transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
          >
            Register
          </Link>
        </div>
      );
    }

    return (
      <div
        className={
          compact
            ? "flex flex-nowrap items-center gap-2 text-xs"
            : "flex flex-wrap items-center gap-3"
        }
      >
        <Link
          to="/login"
          onClick={onNavigate}
          className={
            compact
              ? "inline-flex items-center justify-center min-h-[40px] px-3 whitespace-nowrap rounded-lg font-semibold text-navy hover:bg-stoneband transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
              : "inline-flex items-center justify-center min-h-[44px] px-4 whitespace-nowrap rounded-lg text-sm font-semibold text-navy hover:bg-stoneband transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
          }
        >
          Sign in
        </Link>
        <Link
          to="/register"
          onClick={onNavigate}
          className={
            compact
              ? "inline-flex items-center justify-center min-h-[40px] px-3.5 whitespace-nowrap rounded-lg bg-navy font-semibold text-white hover:bg-navy-light shadow-sm transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
              : "inline-flex items-center justify-center min-h-[44px] px-5 whitespace-nowrap rounded-lg bg-navy text-sm font-semibold text-white hover:bg-navy-light shadow-sm transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
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
      onClick={onNavigate}
      className={`inline-flex items-center gap-2.5 min-h-[44px] rounded-lg transition-colors hover:bg-stoneband focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy ${
        mobile ? "w-full px-3 border border-borderui" : compact ? "px-1.5" : "px-2.5"
      }`}
      aria-label={`Open ${displayName}'s dashboard`}
    >
      <span className="h-9 w-9 grid place-items-center rounded-full bg-navy text-white text-xs font-semibold shrink-0">
        {initials || "U"}
      </span>
      <span
        className={`${mobile ? "inline" : "hidden sm:inline"} text-sm font-semibold text-navy max-w-[180px] truncate`}
      >
        {displayName}
      </span>
    </Link>
  );
}

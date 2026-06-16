import { useEffect, useRef } from "react";
import { createFileRoute, Outlet, useNavigate, useRouterState } from "@tanstack/react-router";

import { useAuth } from "@/context/auth-context";

export const Route = createFileRoute("/_authenticated")({
  component: AuthenticatedLayout,
});

function AuthenticatedLayout() {
  const { isAuthenticated, isHydrated } = useAuth();
  const navigate = useNavigate();
  // Capture the original href ONCE when the layout mounts so subsequent
  // route transitions don't re-trigger redirects with a wrapped `return` param.
  const initialHref = useRouterState({ select: (s) => s.location.href });
  const initialHrefRef = useRef(initialHref);
  const redirected = useRef(false);

  useEffect(() => {
    if (!isHydrated || isAuthenticated || redirected.current) return;
    redirected.current = true;
    navigate({
      to: "/login",
      search: { return: initialHrefRef.current },
      replace: true,
    });
  }, [isAuthenticated, isHydrated, navigate]);

  if (!isHydrated) {
    return (
      <div className="min-h-screen grid place-items-center bg-surface text-brand font-sans">
        <div className="text-center">
          <p className="eyebrow text-accent mb-3">Loading</p>
          <p className="font-serif text-2xl">Opening your learning dashboard...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) return null;

  return <Outlet />;
}

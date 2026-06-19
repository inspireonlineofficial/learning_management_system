import { useEffect } from "react";
import { createFileRoute, Outlet, useNavigate } from "@tanstack/react-router";

import { useAuth } from "@/context/auth-context";

export const Route = createFileRoute("/_authenticated/admin")({
  component: AdminLayout,
});

function AdminLayout() {
  const { user, isHydrated } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isHydrated || !user) return;
    if (user.role !== "admin") navigate({ to: "/403", replace: true });
  }, [user, isHydrated, navigate]);

  if (!isHydrated || !user || user.role !== "admin") return null;

  return <Outlet />;
}

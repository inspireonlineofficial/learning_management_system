import { useEffect } from "react";
import { createFileRoute, Outlet, useNavigate } from "@tanstack/react-router";

import { useAuth } from "@/context/auth-context";

export const Route = createFileRoute("/_authenticated/teacher")({
  component: TeacherLayout,
});

function TeacherLayout() {
  const { user, isHydrated } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isHydrated || !user) return;
    if (user.role !== "teacher") {
      navigate({ to: "/403", replace: true });
    }
  }, [user, isHydrated, navigate]);

  if (!isHydrated || !user) return null;
  if (user.role !== "teacher") return null;
  return <Outlet />;
}

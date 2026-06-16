import { useEffect } from "react";
import { createFileRoute, Outlet, useNavigate } from "@tanstack/react-router";

import { useAuth } from "@/context/auth-context";

export const Route = createFileRoute("/_authenticated/student")({
  component: StudentLayout,
});

function StudentLayout() {
  const { user, isHydrated } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isHydrated || !user) return;
    if (user.role !== "student") {
      navigate({ to: "/403", replace: true });
      return;
    }
    if (user.onboarded === false || user.profile_complete === false) {
      navigate({ to: "/onboarding/student-profile", replace: true });
    }
  }, [user, isHydrated, navigate]);

  if (
    !isHydrated ||
    !user ||
    user.role !== "student" ||
    user.onboarded === false ||
    user.profile_complete === false
  )
    return null;
  return <Outlet />;
}

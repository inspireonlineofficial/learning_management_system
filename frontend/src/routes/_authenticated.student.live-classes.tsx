import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/student/live-classes")({
  component: () => <Outlet />,
});

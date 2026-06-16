import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/student/bookshop/library")({
  component: () => <Outlet />,
});

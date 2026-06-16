import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/student/assessments/$quizId")({
  component: () => <Outlet />,
});

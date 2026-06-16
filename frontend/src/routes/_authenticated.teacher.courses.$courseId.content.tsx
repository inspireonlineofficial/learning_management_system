import { createFileRoute, Navigate } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/content")({
  component: Redirect,
});

function Redirect() {
  const { courseId } = Route.useParams();
  return <Navigate to="/teacher/courses/$courseId/edit" params={{ courseId }} replace />;
}

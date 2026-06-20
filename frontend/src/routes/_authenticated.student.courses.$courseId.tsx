import { Navigate, createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/student/courses/$courseId")({
  component: Page,
});

function Page() {
  const { courseId } = Route.useParams();
  return <Navigate to="/courses/$courseId" params={{ courseId }} replace />;
}

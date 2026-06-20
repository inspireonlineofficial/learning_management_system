import { createFileRoute } from "@tanstack/react-router";

import { CourseEditor } from "./_authenticated.teacher.courses.$courseId.edit";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/builder")({
  component: BuilderPage,
});

function BuilderPage() {
  const { courseId } = Route.useParams();
  return <CourseEditor courseId={courseId} />;
}

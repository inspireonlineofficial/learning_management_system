import { createFileRoute } from "@tanstack/react-router";

import { CreateCoursePage } from "./_authenticated.teacher.courses.new";

export const Route = createFileRoute("/_authenticated/teacher/courses/create")({
  component: CreateCoursePage,
});

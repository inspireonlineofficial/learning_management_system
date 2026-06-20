import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/student/checkout/$courseId")({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: "/student/courses/$courseId/request-access",
      params: { courseId: params.courseId },
      replace: true,
    });
  },
});

import { createFileRoute, Navigate } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/student/checkout/$itemType/$itemId")({
  component: Page,
});

function Page() {
  const { itemType, itemId } = Route.useParams();

  if (itemType === "course") {
    return <Navigate to="/student/checkout/$courseId" params={{ courseId: itemId }} replace />;
  }
  if (itemType === "book") {
    return <Navigate to="/student/bookshop/checkout/$itemId" params={{ itemId }} replace />;
  }
  return <Navigate to="/student" replace />;
}

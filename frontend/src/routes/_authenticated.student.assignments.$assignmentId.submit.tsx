import { createFileRoute, Navigate } from "@tanstack/react-router";

// The full submission form lives at /student/assignments/$assignmentId.
// This dedicated /submit URL keeps share/deeplink behaviour for the back-end and
// redirects to the detail page (which already renders the submit form inline).
export const Route = createFileRoute("/_authenticated/student/assignments/$assignmentId/submit")({
  component: RedirectToDetail,
});

function RedirectToDetail() {
  const { assignmentId } = Route.useParams();
  return (
    <Navigate
      to="/student/assignments/$assignmentId"
      params={{ assignmentId }}
      hash="submit"
      replace
    />
  );
}

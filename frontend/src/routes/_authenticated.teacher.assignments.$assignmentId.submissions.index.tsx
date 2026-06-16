import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listSubmissions, type Submission } from "@/lib/api/submissions";

export const Route = createFileRoute(
  "/_authenticated/teacher/assignments/$assignmentId/submissions/",
)({
  component: Page,
});

function Page() {
  const { assignmentId } = Route.useParams();
  return (
    <DataPage
      eyebrow="Submissions"
      title="Submissions"
      queryKey={["submissions", assignmentId]}
      queryFn={() => listSubmissions(assignmentId)}
      empty={{ title: "No submissions yet" }}
    >
      {(data) => (
        <ListTable<Submission>
          rows={data.items}
          columns={[
            {
              key: "student",
              label: "Student",
              render: (r) => (
                <Link
                  to="/teacher/assignments/$assignmentId/submissions/$submissionId"
                  params={{ assignmentId, submissionId: r.id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {r.student.full_name}
                </Link>
              ),
            },
            {
              key: "submitted",
              label: "Submitted",
              render: (r) => new Date(r.submitted_at).toLocaleString(),
            },
            {
              key: "status",
              label: "Status",
              render: (r) => <span className="eyebrow text-brand/55">{r.status}</span>,
            },
            { key: "score", label: "Score", render: (r) => r.score ?? "—" },
          ]}
        />
      )}
    </DataPage>
  );
}

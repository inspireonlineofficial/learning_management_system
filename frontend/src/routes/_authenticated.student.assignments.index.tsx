import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listMyAssignments, type AssignmentSummary } from "@/lib/api/assignments";

export const Route = createFileRoute("/_authenticated/student/assignments/")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Assignments"
      title="Your assignments"
      queryKey={["my-assignments-table"]}
      queryFn={async () => {
        const res = await listMyAssignments({ limit: 50 });
        return { items: res.data };
      }}
      empty={{ title: "No assignments yet" }}
    >
      {(data: { items: AssignmentSummary[] }) => (
        <ListTable
          rows={data.items}
          columns={[
            {
              key: "title",
              label: "Title",
              render: (r) => (
                <Link
                  to="/student/assignments/$assignmentId"
                  params={{ assignmentId: r.id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {r.title}
                </Link>
              ),
            },
            { key: "course", label: "Course", render: (r) => r.course_title ?? "—" },
            {
              key: "due",
              label: "Due",
              render: (r) => (r.due_at ? new Date(r.due_at).toLocaleDateString() : "—"),
            },
            {
              key: "status",
              label: "Status",
              render: (r) => (
                <span className="eyebrow text-brand/55">{r.status.replaceAll("_", " ")}</span>
              ),
            },
          ]}
        />
      )}
    </DataPage>
  );
}

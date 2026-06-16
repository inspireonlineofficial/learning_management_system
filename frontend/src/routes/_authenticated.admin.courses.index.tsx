import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { apiRequest } from "@/lib/api/client";

type ReviewItem = {
  id: string;
  title: string;
  teacher: { full_name: string };
  submitted_at: string;
  status: string;
};

export const Route = createFileRoute("/_authenticated/admin/courses/")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Courses"
      title="Course review queue"
      queryKey={["admin-course-queue"]}
      queryFn={() =>
        apiRequest<{ data?: ReviewItem[]; items?: ReviewItem[]; courses?: ReviewItem[] }>(
          "/v1/admin/courses",
          {
            auth: true,
          },
        ).then((result) => ({ items: result.items ?? result.data ?? result.courses ?? [] }))
      }
      empty={{ title: "Nothing to review" }}
    >
      {(data) => (
        <ListTable<ReviewItem>
          rows={data.items}
          columns={[
            {
              key: "title",
              label: "Course",
              render: (r) => (
                <Link
                  to="/admin/courses/$courseId/review"
                  params={{ courseId: r.id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {r.title}
                </Link>
              ),
            },
            { key: "teacher", label: "Teacher", render: (r) => r.teacher.full_name },
            {
              key: "status",
              label: "Status",
              render: (r) => <span className="eyebrow text-brand/55">{r.status}</span>,
            },
            {
              key: "when",
              label: "Submitted",
              render: (r) => new Date(r.submitted_at).toLocaleDateString(),
            },
          ]}
        />
      )}
    </DataPage>
  );
}

import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listMyTaughtCourses } from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/courses/")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Courses"
      title="Your courses"
      queryKey={["teacher-courses"]}
      queryFn={() => listMyTaughtCourses()}
      toolbar={
        <Link to="/teacher/courses/new" className="bg-brand text-white px-4 py-2 text-xs">
          New course
        </Link>
      }
      empty={{
        title: "No courses yet",
        action: (
          <Link to="/teacher/courses/new" className="bg-brand text-white px-4 py-2 text-xs">
            Create one
          </Link>
        ),
      }}
    >
      {(data: any) => (
        <ListTable
          rows={data.data ?? []}
          columns={[
            {
              key: "title",
              label: "Title",
              render: (r: any) => (
                <Link
                  to="/teacher/courses/$courseId/edit"
                  params={{ courseId: r.id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {r.title}
                </Link>
              ),
            },
            {
              key: "status",
              label: "Status",
              render: (r: any) => (
                <span className="eyebrow text-brand/55">{r.status ?? "draft"}</span>
              ),
            },
            { key: "students", label: "Students", render: (r: any) => r.enrollment_count ?? 0 },
            {
              key: "updated",
              label: "Updated",
              render: (r: any) =>
                r.updated_at ? new Date(r.updated_at).toLocaleDateString() : "—",
            },
          ]}
        />
      )}
    </DataPage>
  );
}

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
        <Link to="/teacher/courses/create" className="bg-brand text-white px-4 py-2 text-xs">
          Create Course
        </Link>
      }
      empty={{
        title: "No courses yet",
        action: (
          <Link to="/teacher/courses/create" className="bg-brand text-white px-4 py-2 text-xs">
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
                  to="/teacher/courses/$courseId/builder"
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
            {
              key: "access",
              label: "Access",
              render: (r: any) => (
                <span className="eyebrow text-brand/55">
                  {(r.price_type ?? (r.price && r.price > 0 ? "paid" : "free")).toString()}
                </span>
              ),
            },
            {
              key: "students",
              label: "Students",
              render: (r: any) => r.enrollment_count ?? r.total_enrolled ?? 0,
            },
            {
              key: "rating",
              label: "Rating",
              render: (r: any) => Number(r.rating_average ?? r.rating ?? 0).toFixed(1),
            },
            {
              key: "updated",
              label: "Updated",
              render: (r: any) =>
                r.updated_at ? new Date(r.updated_at).toLocaleDateString() : "—",
            },
            {
              key: "actions",
              label: "Actions",
              render: (r: any) => (
                <div className="flex flex-wrap gap-2 text-xs">
                  <Link
                    to="/teacher/courses/$courseId/builder"
                    params={{ courseId: r.id }}
                    className="border border-brand/15 px-2 py-1 hover:bg-brand/[0.03]"
                  >
                    Edit
                  </Link>
                  <Link
                    to="/teacher/courses/$courseId/preview"
                    params={{ courseId: r.id }}
                    className="border border-brand/15 px-2 py-1 hover:bg-brand/[0.03]"
                  >
                    Preview
                  </Link>
                  <Link
                    to="/teacher/courses/$courseId/students"
                    params={{ courseId: r.id }}
                    className="border border-brand/15 px-2 py-1 hover:bg-brand/[0.03]"
                  >
                    Students
                  </Link>
                  <Link
                    to="/teacher/courses/$courseId/reviews"
                    params={{ courseId: r.id }}
                    className="border border-brand/15 px-2 py-1 hover:bg-brand/[0.03]"
                  >
                    Reviews
                  </Link>
                </div>
              ),
            },
          ]}
        />
      )}
    </DataPage>
  );
}

import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listCourseAnalytics, type CourseAnalytics } from "@/lib/api/analytics";

export const Route = createFileRoute("/_authenticated/admin/analytics/courses/")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Analytics"
      title="Course analytics"
      queryKey={["course-analytics"]}
      queryFn={listCourseAnalytics}
      empty={{ title: "No course data" }}
    >
      {(data) => (
        <ListTable<CourseAnalytics & { id: string }>
          rows={data.items.map((c) => ({ ...c, id: c.course_id }))}
          columns={[
            {
              key: "course",
              label: "Course",
              render: (r) => (
                <Link
                  to="/admin/analytics/courses/$courseId"
                  params={{ courseId: r.course_id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {r.course_id.slice(0, 8)}
                </Link>
              ),
            },
            { key: "enrolled", label: "Enrolled", render: (r) => r.enrolled },
            { key: "completed", label: "Completed", render: (r) => r.completed },
            {
              key: "progress",
              label: "Avg progress",
              render: (r) => `${Math.round(r.avg_progress * 100)}%`,
            },
            { key: "revenue", label: "Revenue", render: (r) => `$${r.revenue}` },
            { key: "rating", label: "Rating", render: (r) => r.rating.toFixed(1) },
          ]}
        />
      )}
    </DataPage>
  );
}

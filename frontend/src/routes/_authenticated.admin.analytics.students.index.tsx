import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listStudentAnalytics, type StudentAnalytics } from "@/lib/api/analytics";

export const Route = createFileRoute("/_authenticated/admin/analytics/students/")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Analytics"
      title="Student analytics"
      queryKey={["student-analytics"]}
      queryFn={listStudentAnalytics}
      empty={{ title: "No student data" }}
    >
      {(data) => (
        <ListTable<StudentAnalytics & { id: string }>
          rows={data.items.map((s) => ({ ...s, id: s.student_id }))}
          columns={[
            {
              key: "student",
              label: "Student",
              render: (r) => (
                <Link
                  to="/admin/analytics/students/$studentId"
                  params={{ studentId: r.student_id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {r.student_id.slice(0, 8)}
                </Link>
              ),
            },
            { key: "courses", label: "Courses", render: (r) => r.enrolled_courses },
            { key: "hours", label: "Hours", render: (r) => r.hours_learned },
            { key: "avg", label: "Avg score", render: (r) => r.avg_score },
            { key: "streak", label: "Streak", render: (r) => `${r.streak}d` },
            { key: "certs", label: "Certs", render: (r) => r.certificates },
          ]}
        />
      )}
    </DataPage>
  );
}

import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";

import { AppShell, SectionHeading, StatCard } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { getStudentAnalytics } from "@/lib/api/analytics";

export const Route = createFileRoute("/_authenticated/admin/analytics/students/$studentId")({
  component: Page,
});

function Page() {
  const { studentId } = Route.useParams();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["student-analytics", studentId],
    queryFn: () => getStudentAnalytics(studentId),
  });

  const courses = data?.course_progress ?? [];
  const points = data?.points_history_30d ?? [];
  const averageProgress =
    courses.length > 0
      ? Math.round(
          courses.reduce((sum, course) => sum + course.progress_percent, 0) / courses.length,
        )
      : 0;
  const totalPoints = points.reduce((sum, entry) => sum + entry.points, 0);

  return (
    <AppShell eyebrow="Student" title="Student deep-dive">
      <Link
        to="/admin/analytics/students"
        className="mb-5 inline-flex border border-brand/15 px-4 py-2 text-xs text-brand/65 hover:bg-brand/[0.04]"
      >
        Back to students
      </Link>

      {isLoading && <div className="h-44 border border-brand/10 bg-white/40 animate-pulse" />}
      {isError && (
        <QueryErrorPanel
          error={error}
          variant="compact"
          message={(error as Error)?.message ?? "Failed to load student analytics."}
        />
      )}

      {data && (
        <>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard label="Courses" value={courses.length} />
            <StatCard label="Average progress" value={`${averageProgress}%`} />
            <StatCard label="30d points" value={totalPoints} />
            <StatCard label="Global rank" value={`#${data.global_rank}`} />
          </div>

          <SectionHeading title="Course progress" />
          {courses.length === 0 ? (
            <p className="text-sm text-brand/55 border border-dashed border-brand/15 p-8 text-center">
              No course progress is available for this student.
            </p>
          ) : (
            <ul className="divide-y divide-brand/10 border border-brand/10 bg-white/40">
              {courses.map((course) => (
                <li
                  key={course.course_id}
                  className="px-4 py-3 flex items-center justify-between gap-4"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-brand truncate">{course.course_title}</p>
                    <p className="text-xs text-brand/55">
                      Enrolled {new Date(course.enrolled_at).toLocaleDateString()}
                    </p>
                  </div>
                  <div className="flex items-center gap-3 flex-shrink-0">
                    <div className="w-28 h-1.5 bg-brand/10 overflow-hidden">
                      <div
                        className="h-full bg-accent"
                        style={{ width: `${Math.min(100, course.progress_percent)}%` }}
                      />
                    </div>
                    <span className="w-10 text-right text-xs text-brand/60">
                      {Math.round(course.progress_percent)}%
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </AppShell>
  );
}

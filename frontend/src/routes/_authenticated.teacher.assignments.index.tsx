import { createFileRoute, Link } from "@tanstack/react-router";
import { useQueries, useQuery } from "@tanstack/react-query";

import { AppShell, EmptyState, SectionHeading } from "@/components/layout/app-shell";
import { listMyTaughtCourses } from "@/lib/api/teacher";
import { listCourseAssignments, type AssignmentSummary } from "@/lib/api/assignments";

export const Route = createFileRoute("/_authenticated/teacher/assignments/")({
  component: Page,
});

function Page() {
  const { data: courses } = useQuery({
    queryKey: ["teacher-courses"],
    queryFn: () => listMyTaughtCourses(),
  });
  const courseList = courses?.data ?? [];

  const results = useQueries({
    queries: courseList.map((c) => ({
      queryKey: ["course-assignments", c.id],
      queryFn: () => listCourseAssignments(c.id),
      enabled: !!c.id,
    })),
  });

  const grouped = courseList
    .map((c, i) => ({
      course: c,
      items: ((results[i]?.data as { data?: AssignmentSummary[] } | undefined)?.data ??
        []) as AssignmentSummary[],
    }))
    .filter((g) => g.items.length > 0);

  const total = grouped.reduce((n, g) => n + g.items.length, 0);
  const loading = results.some((r) => r.isLoading);
  const firstCourse = courseList[0];

  return (
    <AppShell eyebrow="Assignments" title="Assignments">
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <p className="max-w-2xl text-brand/65 text-sm">
          All assignments across your courses. Open one to view and grade submissions.
        </p>
        {firstCourse && (
          <Link
            to="/teacher/courses/$courseId/assignments/new"
            params={{ courseId: firstCourse.id }}
            className="inline-flex w-fit bg-brand px-4 py-2 text-xs text-white"
          >
            New assignment
          </Link>
        )}
      </div>

      {!loading && courseList.length > 1 && (
        <div className="mb-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {courseList.map((course) => (
            <Link
              key={course.id}
              to="/teacher/courses/$courseId/assignments/new"
              params={{ courseId: course.id }}
              className="border border-brand/10 bg-white/50 p-4 hover:bg-brand/[0.03]"
            >
              <span className="eyebrow text-brand/45">Create for</span>
              <span className="mt-2 block truncate font-medium text-brand">{course.title}</span>
            </Link>
          ))}
        </div>
      )}

      {loading && (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-16 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      )}

      {!loading && total === 0 && (
        <EmptyState
          title="No assignments yet"
          description={
            courseList.length > 0
              ? "Create the first assignment from one of your courses."
              : "Create a course before adding assignments."
          }
          action={
            firstCourse ? (
              <Link
                to="/teacher/courses/$courseId/assignments/new"
                params={{ courseId: firstCourse.id }}
                className="bg-brand px-4 py-2 text-xs text-white"
              >
                Create assignment
              </Link>
            ) : (
              <Link to="/teacher/courses/new" className="bg-brand px-4 py-2 text-xs text-white">
                Create course
              </Link>
            )
          }
        />
      )}

      {!loading &&
        grouped.map((g) => (
          <div key={g.course.id} className="mb-10">
            <SectionHeading title={g.course.title} />
            <ul className="space-y-2 max-w-3xl">
              {g.items.map((a) => (
                <li
                  key={a.id}
                  className="border border-brand/10 bg-white/50 p-4 flex justify-between items-center gap-3"
                >
                  <div className="min-w-0">
                    <p className="font-serif text-base truncate">{a.title}</p>
                    <p className="text-xs text-brand/55 mt-1">
                      Due {a.due_at ? new Date(a.due_at).toLocaleDateString() : "—"} ·{" "}
                      {a.total_points} pts
                    </p>
                  </div>
                  <Link
                    to="/teacher/assignments/$assignmentId/submissions"
                    params={{ assignmentId: a.id }}
                    className="text-xs border border-brand/15 px-3 py-2 hover:bg-brand/[0.03] whitespace-nowrap"
                  >
                    View submissions
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        ))}
    </AppShell>
  );
}

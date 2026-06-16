import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { BookOpen, CheckCircle2 } from "lucide-react";

import { AppShell, EmptyState, StatCard } from "@/components/layout/app-shell";
import { listMyEnrollments } from "@/lib/api/student";

export const Route = createFileRoute("/_authenticated/student/progress")({
  component: StudentProgressPage,
});

function StudentProgressPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["student-progress-overview"],
    queryFn: () => listMyEnrollments({ limit: 100 }),
  });

  const courses = data?.data ?? [];
  const average =
    courses.length > 0
      ? Math.round(courses.reduce((sum, item) => sum + item.progress_percent, 0) / courses.length)
      : 0;
  const completed = courses.filter((item) => item.progress_percent >= 100).length;

  return (
    <AppShell eyebrow="Progress" title="Track every course requirement.">
      <div className="grid sm:grid-cols-3 gap-4">
        <StatCard label="Courses" value={courses.length || "—"} />
        <StatCard label="Average progress" value={courses.length ? `${average}%` : "—"} />
        <StatCard label="Completed" value={courses.length ? completed : "—"} />
      </div>

      {isError && (
        <p className="mt-8 border border-destructive/20 bg-destructive/5 p-6 text-sm text-destructive">
          {(error as Error)?.message ?? "Could not load progress."}
        </p>
      )}

      {isLoading ? (
        <div className="mt-10 grid sm:grid-cols-2 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-36 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : courses.length === 0 ? (
        <div className="mt-10">
          <EmptyState
            icon={BookOpen}
            title="No course progress yet"
            description="Enroll in a course and your lesson, quiz, assignment, and certificate progress will appear here."
            action={
              <Link to="/courses" className="bg-brand text-white px-6 py-3 text-sm">
                Browse courses
              </Link>
            }
          />
        </div>
      ) : (
        <div className="mt-10 grid lg:grid-cols-2 gap-5">
          {courses.map((item) => (
            <Link
              key={item.id}
              to="/student/progress/$courseId"
              params={{ courseId: item.course.id }}
              className="border border-brand/10 bg-white/50 p-5 hover:bg-white transition-colors"
            >
              <div className="flex items-start justify-between gap-4">
                <div>
                  {item.course.category?.name && (
                    <p className="eyebrow text-accent mb-2">{item.course.category.name}</p>
                  )}
                  <h2 className="font-serif text-xl">{item.course.title}</h2>
                  {item.next_lesson?.title && (
                    <p className="mt-2 text-xs text-brand/55">
                      Next: <span className="text-brand/75">{item.next_lesson.title}</span>
                    </p>
                  )}
                </div>
                {item.progress_percent >= 100 && (
                  <CheckCircle2 className="h-5 w-5 text-emerald-600 shrink-0" />
                )}
              </div>
              <div className="mt-6 h-2 bg-brand/10">
                <div
                  className="h-full bg-accent"
                  style={{ width: `${Math.min(100, item.progress_percent)}%` }}
                />
              </div>
              <p className="mt-2 text-xs text-brand/50">
                {Math.round(item.progress_percent)}% complete
              </p>
            </Link>
          ))}
        </div>
      )}
    </AppShell>
  );
}

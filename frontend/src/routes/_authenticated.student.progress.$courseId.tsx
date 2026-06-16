import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Award, CheckCircle2, ClipboardList, PlayCircle } from "lucide-react";

import { AppShell, StatCard } from "@/components/layout/app-shell";
import { getCourse } from "@/lib/api/courses";
import { getCourseProgress } from "@/lib/api/student";

export const Route = createFileRoute("/_authenticated/student/progress/$courseId")({
  component: StudentCourseProgressPage,
});

function StudentCourseProgressPage() {
  const { courseId } = Route.useParams();
  const course = useQuery({
    queryKey: ["course", courseId],
    queryFn: () => getCourse(courseId),
  });
  const progress = useQuery({
    queryKey: ["course-progress", courseId],
    queryFn: () => getCourseProgress(courseId),
  });

  const modules = course.data?.modules ?? progress.data?.modules ?? [];
  const totalLessons = modules.reduce((sum, module) => sum + module.lessons.length, 0);
  const completedLessons = progress.data?.completed_lessons.length ?? 0;
  const percent = Math.round(progress.data?.progress_percent ?? 0);

  return (
    <AppShell eyebrow="Course progress" title={course.data?.title ?? "Course progress"}>
      <div className="grid sm:grid-cols-4 gap-4">
        <StatCard label="Progress" value={`${percent}%`} />
        <StatCard label="Lessons" value={`${completedLessons}/${totalLessons || "—"}`} />
        <StatCard label="Quizzes" value="Tracked" hint="Shown as backend data becomes available." />
        <StatCard label="Assignments" value="Tracked" hint="Grades and revisions appear here." />
      </div>

      <div className="mt-10 flex flex-wrap gap-3">
        <Link
          to="/student/player/$courseId"
          params={{ courseId }}
          className="inline-flex items-center gap-2 bg-brand text-white px-5 py-3 text-sm"
        >
          <PlayCircle className="h-4 w-4" />
          Continue learning
        </Link>
        <Link
          to="/student/certificates/$courseId"
          params={{ courseId }}
          className="inline-flex items-center gap-2 border border-brand/15 px-5 py-3 text-sm"
        >
          <Award className="h-4 w-4" />
          Certificate
        </Link>
      </div>

      {course.isLoading || progress.isLoading ? (
        <div className="mt-10 h-64 border border-brand/10 bg-white/30 animate-pulse" />
      ) : (
        <section className="mt-12">
          <h2 className="font-serif text-2xl mb-5">Requirements</h2>
          {modules.length === 0 ? (
            <p className="border border-dashed border-brand/15 p-8 text-sm text-brand/55 text-center">
              Detailed module progress is not available yet.
            </p>
          ) : (
            <div className="space-y-5">
              {modules.map((module, index) => (
                <div key={module.id} className="border border-brand/10 bg-white/40">
                  <div className="p-5 border-b border-brand/10">
                    <p className="eyebrow text-brand/45">Module {index + 1}</p>
                    <h3 className="mt-1 font-serif text-xl">{module.title}</h3>
                  </div>
                  <ul className="divide-y divide-brand/5">
                    {module.lessons.map((lesson) => {
                      const done = progress.data?.completed_lessons.includes(lesson.id) ?? false;
                      const current = progress.data?.current_lesson_id === lesson.id;
                      return (
                        <li
                          key={lesson.id}
                          className="px-5 py-3 flex items-center justify-between gap-4 text-sm"
                        >
                          <span className="flex items-center gap-3 min-w-0">
                            {done ? (
                              <CheckCircle2 className="h-4 w-4 text-emerald-600 shrink-0" />
                            ) : (
                              <ClipboardList className="h-4 w-4 text-brand/30 shrink-0" />
                            )}
                            <span className="truncate">{lesson.title}</span>
                          </span>
                          <span className="text-xs text-brand/45">
                            {done ? "Complete" : current ? "Current" : "Not started"}
                          </span>
                        </li>
                      );
                    })}
                  </ul>
                </div>
              ))}
            </div>
          )}
        </section>
      )}
    </AppShell>
  );
}

import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Users } from "lucide-react";

import { AppShell } from "@/components/layout/app-shell";
import { listCourseStudents } from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/students")({
  component: Page,
});

function Page() {
  const { courseId } = Route.useParams();
  const students = useQuery({
    queryKey: ["course-students", courseId],
    queryFn: () => listCourseStudents(courseId, { limit: 100 }),
  });

  return (
    <AppShell>
      <Link
        to="/teacher/courses/$courseId/builder"
        params={{ courseId }}
        className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-6"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Back to builder
      </Link>
      <p className="eyebrow text-accent">Students</p>
      <h1 className="mt-3 font-serif text-4xl lg:text-5xl">Course students</h1>

      {students.isLoading ? (
        <div className="mt-8 space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-14 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : !students.data || students.data.data.length === 0 ? (
        <p className="mt-8 text-sm text-brand/55 border border-dashed border-brand/15 p-8 text-center">
          <Users className="h-6 w-6 mx-auto mb-3 text-brand/30" />
          No approved students yet.
        </p>
      ) : (
        <ul className="mt-8 divide-y divide-brand/10 border-y border-brand/10">
          {students.data.data.map((student) => (
            <li key={student.id} className="flex items-center gap-4 py-4 px-2">
              <div className="h-9 w-9 grid place-items-center bg-brand/10 text-brand text-xs font-medium">
                {student.full_name.charAt(0)}
              </div>
              <div className="flex-1 min-w-0">
                <p className="font-medium text-brand truncate">{student.full_name}</p>
                <p className="text-xs text-brand/50 truncate">{student.email}</p>
              </div>
              <div className="text-right shrink-0">
                <p className="text-sm font-medium">{student.progress_percent}%</p>
                <p className="text-[11px] text-brand/45">Progress</p>
              </div>
            </li>
          ))}
        </ul>
      )}
    </AppShell>
  );
}

import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { AppShell, SectionHeading, StatCard } from "@/components/layout/app-shell";
import { listMyTaughtCourses } from "@/lib/api/teacher";
import { getTeacherAnalytics } from "@/lib/api/analytics";

export const Route = createFileRoute("/_authenticated/teacher/")({
  component: Page,
});

function Page() {
  const { data: courses } = useQuery({
    queryKey: ["teacher-courses"],
    queryFn: () => listMyTaughtCourses(),
  });
  const { data: stats } = useQuery({
    queryKey: ["teacher-analytics"],
    queryFn: getTeacherAnalytics,
  });

  const published = courses?.data.filter((c: any) => c.status === "published").length ?? 0;
  const drafts = courses?.data.filter((c: any) => c.status === "draft").length ?? 0;

  return (
    <AppShell eyebrow="Teacher portal" title="Your studio">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Courses"
          value={courses?.data.length ?? "—"}
          hint={`${published} published, ${drafts} drafts`}
        />
        <StatCard label="Students" value={stats?.total_users ?? "—"} hint="across all courses" />
        <StatCard label="Active 30d" value={stats?.active_users_30d ?? "—"} />
        <StatCard label="Revenue" value={stats?.revenue != null ? `$${stats.revenue}` : "—"} />
      </div>
      <SectionHeading
        title="Quick actions"
        action={
          <Link to="/teacher/courses/new" className="bg-brand text-white px-4 py-2 text-xs">
            New course
          </Link>
        }
      />
      <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[
          { to: "/teacher/courses", label: "Manage courses" },
          { to: "/teacher/quiz-builder", label: "Build quizzes" },
          { to: "/teacher/assignments", label: "Review submissions" },
          { to: "/teacher/live", label: "Schedule live class" },
        ].map((a) => (
          <Link
            key={a.to}
            to={a.to as any}
            className="border border-brand/10 bg-white/50 p-5 hover:bg-white"
          >
            <p className="font-serif text-base">{a.label}</p>
          </Link>
        ))}
      </div>
    </AppShell>
  );
}

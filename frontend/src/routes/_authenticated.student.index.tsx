import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Award, Calendar, Flame } from "lucide-react";

import { AppShell, EmptyState, SectionHeading, StatCard } from "@/components/layout/app-shell";
import { useAuth } from "@/context/auth-context";
import { getStudentDashboard } from "@/lib/api/student";

export const Route = createFileRoute("/_authenticated/student/")({
  component: DashboardPage,
});

function DashboardPage() {
  const { user } = useAuth();
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["dashboard"],
    queryFn: getStudentDashboard,
  });

  const firstName = user?.full_name?.split(" ")[0] ?? "Scholar";
  const continueLearning = data?.continue_learning ?? [];
  const upcomingLive = data?.upcoming_live ?? [];
  const recentAchievements = data?.recent_achievements ?? [];

  return (
    <AppShell eyebrow={`Welcome back, ${firstName}`} title="Your study hall.">
      {isError && (
        <div className="border border-destructive/20 bg-destructive/5 p-6 text-sm">
          <p className="font-medium text-destructive">Couldn't load your dashboard</p>
          <p className="mt-1 text-brand/60">{(error as Error)?.message}</p>
          <button onClick={() => refetch()} className="mt-3 px-4 py-2 bg-brand text-white text-xs">
            Try again
          </button>
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Enrolled"
          value={isLoading ? "—" : (data?.stats.enrolled_courses ?? 0)}
          hint="active courses"
        />
        <StatCard
          label="Completed"
          value={isLoading ? "—" : (data?.stats.completed_courses ?? 0)}
          hint="finished"
        />
        <StatCard
          label="Hours"
          value={isLoading ? "—" : (data?.stats.hours_learned ?? 0)}
          hint="of study"
        />
        <StatCard
          label="Points"
          value={isLoading ? "—" : (data?.stats.points ?? 0).toLocaleString()}
          hint={data?.stats.streak_days ? `${data.stats.streak_days}-day streak` : "earned"}
        />
      </div>

      {/* Continue learning */}
      <section className="mt-14">
        <SectionHeading
          title="Continue learning"
          action={
            <Link
              to="/student/my-courses"
              className="text-xs text-brand/55 hover:text-brand underline underline-offset-4"
            >
              See all
            </Link>
          }
        />
        {isLoading ? (
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-40 border border-brand/10 bg-white/30 animate-pulse" />
            ))}
          </div>
        ) : continueLearning.length > 0 ? (
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {continueLearning.map((e) => (
              <Link
                key={e.id}
                to="/student/player/$courseId"
                params={{ courseId: e.course.id }}
                className="group block border border-brand/10 bg-white/50 hover:bg-white p-5 transition-colors"
              >
                {e.course.category?.name && (
                  <p className="eyebrow text-accent mb-2">{e.course.category.name}</p>
                )}
                <p className="font-serif text-lg leading-snug">{e.course.title}</p>
                {e.next_lesson?.title && (
                  <p className="mt-3 text-xs text-brand/55">
                    Next · <span className="text-brand/75">{e.next_lesson.title}</span>
                  </p>
                )}
                <div className="mt-5">
                  <div className="h-1 bg-brand/10">
                    <div
                      className="h-full bg-accent transition-all"
                      style={{ width: `${Math.min(100, e.progress_percent)}%` }}
                    />
                  </div>
                  <p className="mt-2 text-[11px] text-brand/45">
                    {Math.round(e.progress_percent)}% complete
                  </p>
                </div>
              </Link>
            ))}
          </div>
        ) : (
          <EmptyState
            title="Nothing in progress yet"
            description="Browse the catalog and enroll in your first course."
            action={
              <Link to="/courses" className="bg-brand text-white px-6 py-3 text-sm">
                Browse catalog
              </Link>
            }
          />
        )}
      </section>

      {/* Two-column extras */}
      <div className="mt-14 grid lg:grid-cols-2 gap-10">
        <section>
          <SectionHeading title="Upcoming live sessions" />
          {!isLoading && upcomingLive.length === 0 ? (
            <div className="border border-dashed border-brand/15 p-8 text-sm text-brand/55">
              <Calendar className="h-5 w-5 text-brand/30 mb-3" />
              No live sessions scheduled.
            </div>
          ) : (
            <ul className="space-y-3">
              {upcomingLive.map((s) => (
                <li key={s.id} className="border border-brand/10 bg-white/50 p-4 flex gap-4">
                  <Calendar className="h-5 w-5 text-accent flex-shrink-0 mt-0.5" />
                  <div className="min-w-0">
                    <p className="font-serif text-base truncate">{s.title}</p>
                    <p className="text-xs text-brand/55 mt-1">{s.course_title}</p>
                    <p className="text-xs text-brand/45 mt-1">
                      {new Date(s.starts_at).toLocaleString()}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section>
          <SectionHeading title="Recent achievements" />
          {!isLoading && recentAchievements.length === 0 ? (
            <div className="border border-dashed border-brand/15 p-8 text-sm text-brand/55">
              <Award className="h-5 w-5 text-brand/30 mb-3" />
              Earn your first badge by completing a lesson.
            </div>
          ) : (
            <ul className="space-y-3">
              {recentAchievements.map((a) => (
                <li key={a.id} className="border border-brand/10 bg-white/50 p-4 flex gap-4">
                  <Flame className="h-5 w-5 text-accent flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="font-serif text-base">{a.title}</p>
                    <p className="text-xs text-brand/45 mt-1">
                      {new Date(a.earned_at).toLocaleDateString()}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </AppShell>
  );
}

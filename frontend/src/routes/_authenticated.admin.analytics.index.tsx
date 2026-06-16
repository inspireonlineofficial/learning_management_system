import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { AppShell, StatCard, SectionHeading } from "@/components/layout/app-shell";
import { getPlatformAnalytics } from "@/lib/api/analytics";

export const Route = createFileRoute("/_authenticated/admin/analytics/")({
  component: Page,
});

function Page() {
  const { data, isLoading } = useQuery({
    queryKey: ["platform-analytics"],
    queryFn: getPlatformAnalytics,
  });

  const trend = data?.trend ?? [];
  const maxUsers = Math.max(1, ...trend.map((t) => t.users));
  const maxRevenue = Math.max(1, ...trend.map((t) => t.revenue));

  return (
    <AppShell eyebrow="Analytics" title="Platform analytics">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard label="Users" value={data?.total_users ?? "—"} />
        <StatCard label="Active 30d" value={data?.active_users_30d ?? "—"} />
        <StatCard label="Enrolments" value={data?.enrollments ?? "—"} />
        <StatCard label="Revenue" value={data?.revenue != null ? `$${data.revenue}` : "—"} />
      </div>

      <SectionHeading title="Trend" />
      {isLoading && <div className="h-48 border border-brand/10 bg-white/30 animate-pulse" />}
      {trend.length > 0 && (
        <div className="border border-brand/10 bg-white/40 p-6">
          <div className="flex items-end gap-1 h-48">
            {trend.map((t) => (
              <div
                key={t.date}
                title={`${t.date} · ${t.users} users · $${t.revenue}`}
                className="flex-1 flex flex-col justify-end gap-0.5"
              >
                <div
                  className="bg-accent/70"
                  style={{ height: `${(t.users / maxUsers) * 100}%` }}
                />
                <div
                  className="bg-brand/60"
                  style={{ height: `${(t.revenue / maxRevenue) * 30}%` }}
                />
              </div>
            ))}
          </div>
          <div className="mt-4 flex gap-6 text-xs text-brand/55">
            <span className="flex items-center gap-2">
              <span className="h-2.5 w-2.5 bg-accent/70" /> Users
            </span>
            <span className="flex items-center gap-2">
              <span className="h-2.5 w-2.5 bg-brand/60" /> Revenue
            </span>
          </div>
        </div>
      )}

      <SectionHeading title="Drill down" />
      <div className="grid sm:grid-cols-2 gap-4 max-w-2xl">
        <Link
          to="/admin/analytics/courses"
          className="border border-brand/10 bg-white/50 p-5 hover:bg-white"
        >
          By course
        </Link>
        <Link
          to="/admin/analytics/students"
          className="border border-brand/10 bg-white/50 p-5 hover:bg-white"
        >
          By student
        </Link>
      </div>
    </AppShell>
  );
}

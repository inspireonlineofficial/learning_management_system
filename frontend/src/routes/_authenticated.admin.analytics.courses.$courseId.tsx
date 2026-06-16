import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { AppShell, StatCard } from "@/components/layout/app-shell";
import { getCourseAnalytics } from "@/lib/api/analytics";

export const Route = createFileRoute("/_authenticated/admin/analytics/courses/$courseId")({
  component: Page,
});

function Page() {
  const { courseId } = Route.useParams();
  const { data } = useQuery({
    queryKey: ["course-analytics", courseId],
    queryFn: () => getCourseAnalytics(courseId),
  });
  return (
    <AppShell eyebrow="Course" title="Course deep-dive">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard label="Enrolled" value={data?.enrolled ?? "—"} />
        <StatCard label="Completed" value={data?.completed ?? "—"} />
        <StatCard
          label="Avg progress"
          value={data ? `${Math.round(data.avg_progress * 100)}%` : "—"}
        />
        <StatCard label="Revenue" value={data?.revenue != null ? `$${data.revenue}` : "—"} />
      </div>
    </AppShell>
  );
}

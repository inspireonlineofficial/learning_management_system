import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { AppShell, SectionHeading, StatCard } from "@/components/layout/app-shell";
import { getAdminStats } from "@/lib/api/admin";

export const Route = createFileRoute("/_authenticated/admin/")({
  component: Page,
});

function Page() {
  const { data } = useQuery({ queryKey: ["admin-stats"], queryFn: getAdminStats });
  return (
    <AppShell eyebrow="Admin" title="Platform overview">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Users"
          value={data?.total_users ?? "—"}
          hint={data ? `${data.active_users_30d ?? 0} active` : undefined}
        />
        <StatCard label="Courses" value={data?.total_courses ?? "—"} />
        <StatCard label="Enrolments" value={data?.total_enrollments ?? "—"} />
        <StatCard
          label="Revenue"
          value={(data as any)?.total_revenue != null ? `$${(data as any).total_revenue}` : "—"}
        />
      </div>
      <SectionHeading title="Quick actions" />
      <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[
          { to: "/admin/approvals", label: "Pending approvals" },
          { to: "/admin/courses", label: "Course review queue" },
          { to: "/admin/moderation", label: "Moderation queue" },
          { to: "/admin/users/new", label: "Create user" },
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

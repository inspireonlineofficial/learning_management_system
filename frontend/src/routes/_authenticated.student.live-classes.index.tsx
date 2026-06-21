import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Radio, Users, Video } from "lucide-react";
import { useState } from "react";

import { AppShell, EmptyState, SectionHeading } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { listLiveSessions, type LiveSessionSummary } from "@/lib/api/live";

type Scope = "upcoming" | "live" | "past";

export const Route = createFileRoute("/_authenticated/student/live-classes/")({
  component: LiveHub,
});

function LiveHub() {
  const [scope, setScope] = useState<Scope>("upcoming");

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["live-sessions", scope],
    queryFn: () => listLiveSessions({ scope, limit: 50 }),
  });

  return (
    <AppShell eyebrow="Live classes" title="Join the lecture hall.">
      <div className="flex flex-wrap gap-2 mb-8">
        {(["upcoming", "live", "past"] as Scope[]).map((s) => (
          <button
            key={s}
            onClick={() => setScope(s)}
            className={`px-5 py-2 text-sm font-medium capitalize transition-colors ${
              scope === s
                ? "bg-brand text-white"
                : "border border-brand/15 text-brand/70 hover:text-brand hover:bg-brand/[0.03]"
            }`}
          >
            {s}
          </button>
        ))}
      </div>

      {isError ? (
        <QueryErrorPanel error={error} title="Couldn't load sessions" onRetry={() => refetch()} />
      ) : isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-28 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : !data || data.data.length === 0 ? (
        <EmptyState
          icon={Video}
          title="Nothing scheduled"
          description={`No ${scope} live sessions to show.`}
        />
      ) : (
        <>
          <SectionHeading title={`${data.meta.total} session${data.meta.total === 1 ? "" : "s"}`} />
          <ul className="grid gap-4 md:grid-cols-2">
            {data.data.map((s) => (
              <SessionCard key={s.id} session={s} />
            ))}
          </ul>
        </>
      )}
    </AppShell>
  );
}

function SessionCard({ session }: { session: LiveSessionSummary }) {
  const starts = new Date(session.starts_at);
  const date = starts.toLocaleDateString(undefined, {
    weekday: "short",
    month: "short",
    day: "numeric",
  });
  const time = starts.toLocaleTimeString(undefined, {
    hour: "numeric",
    minute: "2-digit",
  });

  return (
    <Link
      to="/student/live-classes/$sessionId"
      params={{ sessionId: session.id }}
      className="block border border-brand/10 bg-white/40 p-6 hover:border-brand/30 transition-colors"
    >
      <div className="flex items-start justify-between gap-3 mb-4">
        {session.course_title && (
          <p className="eyebrow text-brand/40 truncate">{session.course_title}</p>
        )}
        <LiveBadge status={session.status} />
      </div>
      <h3 className="font-serif text-xl leading-snug mb-3">{session.title}</h3>
      {session.host_name && (
        <p className="text-xs text-brand/55 mb-3">Hosted by {session.host_name}</p>
      )}
      <div className="flex items-center justify-between text-xs text-brand/60 pt-3 border-t border-brand/10">
        <span>
          {date} · {time}
          {session.duration_minutes ? ` · ${session.duration_minutes}m` : ""}
        </span>
        {typeof session.attendees_count === "number" && (
          <span className="flex items-center gap-1.5">
            <Users className="h-3 w-3" />
            {session.attendees_count}
            {session.capacity ? ` / ${session.capacity}` : ""}
          </span>
        )}
      </div>
    </Link>
  );
}

function LiveBadge({ status }: { status: LiveSessionSummary["status"] }) {
  if (status === "live")
    return (
      <span className="flex items-center gap-1.5 px-2.5 py-1 text-[11px] font-medium text-destructive bg-destructive/10 border border-destructive/30">
        <Radio className="h-3 w-3 animate-pulse" />
        Live now
      </span>
    );
  const map: Record<string, { label: string; cls: string }> = {
    scheduled: { label: "Scheduled", cls: "text-brand/60 border-brand/15" },
    ended: { label: "Ended", cls: "text-brand/50 border-brand/15" },
    cancelled: { label: "Cancelled", cls: "text-brand/40 border-brand/10 line-through" },
  };
  const m = map[status] ?? map.scheduled;
  return <span className={`px-2.5 py-1 text-[11px] font-medium border ${m.cls}`}>{m.label}</span>;
}

import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { CalendarPlus, Radio, Users } from "lucide-react";

import { AppShell, EmptyState, SectionHeading } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { listTeacherLiveSessions, type LiveSessionSummary } from "@/lib/api/live";

type Scope = "upcoming" | "live" | "past";

export const Route = createFileRoute("/_authenticated/teacher/live/")({
  component: Page,
});

function Page() {
  const [scope, setScope] = useState<Scope>("upcoming");
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["teacher-live", scope],
    queryFn: () => listTeacherLiveSessions({ scope, limit: 50 }),
  });

  return (
    <AppShell eyebrow="Live" title="Live classes">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-8">
        <div className="flex flex-wrap gap-2">
          {(["upcoming", "live", "past"] as Scope[]).map((s) => (
            <button
              key={s}
              onClick={() => setScope(s)}
              className={`px-5 py-2 text-sm font-medium capitalize ${
                scope === s
                  ? "bg-brand text-white"
                  : "border border-brand/15 text-brand/70 hover:bg-brand/[0.03]"
              }`}
            >
              {s}
            </button>
          ))}
        </div>
        <Link
          to="/teacher/live/schedule"
          className="inline-flex items-center gap-2 bg-brand text-white px-4 py-2 text-xs"
        >
          <CalendarPlus className="h-3.5 w-3.5" />
          Schedule session
        </Link>
      </div>

      {isError ? (
        <QueryErrorPanel error={error} title="Couldn't load sessions" onRetry={() => refetch()} />
      ) : isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-24 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : !data || data.data.length === 0 ? (
        <EmptyState
          title="No sessions yet"
          description={`Nothing ${scope}.`}
          action={
            <Link to="/teacher/live/schedule" className="bg-brand text-white px-4 py-2 text-xs">
              Schedule one
            </Link>
          }
        />
      ) : (
        <>
          <SectionHeading title={`${data.meta.total} session${data.meta.total === 1 ? "" : "s"}`} />
          <ul className="space-y-3">
            {data.data.map((s) => (
              <Row key={s.id} session={s} />
            ))}
          </ul>
        </>
      )}
    </AppShell>
  );
}

function Row({ session }: { session: LiveSessionSummary }) {
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
  const isLive = session.status === "live";

  return (
    <li className="border border-brand/10 bg-white/40 p-5 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div className="min-w-0">
        {session.course_title && (
          <p className="eyebrow text-brand/40 truncate">{session.course_title}</p>
        )}
        <p className="font-serif text-lg leading-snug mt-1">{session.title}</p>
        <p className="mt-1 text-xs text-brand/55 flex flex-wrap gap-x-3 gap-y-1">
          <span>
            {date} · {time}
            {session.duration_minutes ? ` · ${session.duration_minutes}m` : ""}
          </span>
          {typeof session.attendees_count === "number" && (
            <span className="inline-flex items-center gap-1">
              <Users className="h-3 w-3" />
              {session.attendees_count}
              {session.capacity ? ` / ${session.capacity}` : ""}
            </span>
          )}
        </p>
      </div>
      <div className="flex items-center gap-2 flex-shrink-0">
        {isLive && (
          <Link
            to="/teacher/live/$sessionId/room"
            params={{ sessionId: session.id }}
            className="inline-flex items-center gap-1.5 bg-destructive text-white px-3 py-2 text-xs"
          >
            <Radio className="h-3 w-3 animate-pulse" />
            Open room
          </Link>
        )}
        <Link
          to="/teacher/live/$sessionId"
          params={{ sessionId: session.id }}
          className="text-xs border border-brand/15 px-3 py-2 hover:bg-brand/[0.03]"
        >
          Manage
        </Link>
      </div>
    </li>
  );
}

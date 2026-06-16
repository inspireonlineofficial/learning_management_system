import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Calendar, Clock, Radio, Users } from "lucide-react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { getLiveSession, joinSession } from "@/lib/api/live";

export const Route = createFileRoute("/_authenticated/student/live-classes/$sessionId")({
  component: LiveSessionPage,
});

function LiveSessionPage() {
  const { sessionId } = Route.useParams();

  const {
    data: session,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["live-session", sessionId],
    queryFn: () => getLiveSession(sessionId),
  });

  const join = useMutation({
    mutationFn: () => joinSession(sessionId),
    onSuccess: (data) => {
      if (data.room_token) {
        window.location.assign(`/student/live-classes/${sessionId}/room`);
      } else {
        toast.error("No room token available");
      }
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not join"),
  });

  if (isLoading) {
    return (
      <AppShell>
        <div className="h-12 w-1/3 bg-brand/10 animate-pulse mb-6" />
        <div className="h-64 bg-brand/5 animate-pulse" />
      </AppShell>
    );
  }

  if (isError || !session) {
    return (
      <AppShell title="Session unavailable">
        <p className="text-sm text-brand/60">{(error as Error)?.message ?? "Not found"}</p>
        <button onClick={() => refetch()} className="mt-4 px-4 py-2 bg-brand text-white text-xs">
          Try again
        </button>
      </AppShell>
    );
  }

  const starts = new Date(session.starts_at);
  const ends = new Date(session.ends_at);
  const isLive = session.status === "live";
  const isEnded = session.status === "ended";

  return (
    <AppShell>
      <Link
        to="/student/live-classes"
        className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-6"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Back to live
      </Link>

      <div className="grid lg:grid-cols-[1fr_320px] gap-10">
        <div>
          <p className="eyebrow text-accent mb-3">Live session</p>
          <h1 className="font-serif text-4xl lg:text-5xl text-balance mb-4">{session.title}</h1>
          <p className="text-sm text-brand/60 mb-8">
            The backend currently exposes the schedule, room token, and attendance state for this
            session.
          </p>
        </div>

        <aside className="lg:sticky lg:top-10 lg:self-start space-y-4">
          <div className="border border-brand/10 bg-white/40 p-6">
            <div className="flex items-start justify-between mb-4">
              <p className="eyebrow text-brand/45">Session</p>
              {isLive ? (
                <span className="flex items-center gap-1.5 px-2.5 py-1 text-[11px] font-medium text-destructive bg-destructive/10 border border-destructive/30">
                  <Radio className="h-3 w-3 animate-pulse" />
                  Live
                </span>
              ) : (
                <span className="px-2.5 py-1 text-[11px] font-medium border border-brand/15 text-brand/60 capitalize">
                  {session.status}
                </span>
              )}
            </div>

            <ul className="space-y-3 text-sm text-brand/70 mb-6">
              <li className="flex items-center gap-2">
                <Calendar className="h-4 w-4 text-brand/45" />
                {starts.toLocaleDateString(undefined, {
                  weekday: "long",
                  month: "long",
                  day: "numeric",
                  year: "numeric",
                })}
              </li>
              <li className="flex items-center gap-2">
                <Clock className="h-4 w-4 text-brand/45" />
                {starts.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" })}
                {" – "}
                {ends.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" })}
              </li>
              {typeof session.attendees_count === "number" && (
                <li className="flex items-center gap-2">
                  <Users className="h-4 w-4 text-brand/45" />
                  {session.attendees_count} registered
                  {session.capacity ? ` / ${session.capacity}` : ""}
                </li>
              )}
            </ul>

            {isEnded ? (
              <p className="text-xs text-brand/55 text-center py-3">The session has ended.</p>
            ) : isLive ? (
              <button
                onClick={() => join.mutate()}
                disabled={join.isPending}
                className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-destructive text-white text-sm font-medium hover:bg-destructive/90 disabled:opacity-60"
              >
                <Radio className="h-4 w-4" />
                {join.isPending ? "Opening…" : "Join now"}
              </button>
            ) : session.status === "cancelled" ? (
              <p className="text-xs text-brand/55 text-center py-3">This session was cancelled.</p>
            ) : (
              <p className="text-xs text-brand/55 text-center py-3">
                Join is available when the session goes live.
              </p>
            )}
          </div>

          {session.course_id && (
            <Link
              to="/courses/$courseId"
              params={{ courseId: session.course_id }}
              className="block text-center text-xs text-brand/55 hover:text-brand"
            >
              View parent course →
            </Link>
          )}
        </aside>
      </div>
    </AppShell>
  );
}

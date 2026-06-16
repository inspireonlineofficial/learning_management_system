import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Radio } from "lucide-react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { DetailGrid } from "@/components/layout/data-page";
import {
  cancelLiveSession,
  endLiveSession,
  getTeacherLiveSession,
  startLiveSession,
} from "@/lib/api/live";

export const Route = createFileRoute("/_authenticated/teacher/live/$sessionId/")({
  component: Page,
});

function Page() {
  const { sessionId } = Route.useParams();
  const navigate = useNavigate();
  const qc = useQueryClient();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["teacher-live-session", sessionId],
    queryFn: () => getTeacherLiveSession(sessionId),
  });

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ["teacher-live"] });
    qc.invalidateQueries({ queryKey: ["teacher-live-session", sessionId] });
  };

  const start = useMutation({
    mutationFn: () => startLiveSession(sessionId),
    onSuccess: () => {
      invalidate();
      navigate({ to: "/teacher/live/$sessionId/room", params: { sessionId } });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const end = useMutation({
    mutationFn: () => endLiveSession(sessionId),
    onSuccess: () => {
      toast.success("Session ended");
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const cancel = useMutation({
    mutationFn: () => cancelLiveSession(sessionId),
    onSuccess: () => {
      toast.success("Session cancelled");
      invalidate();
      navigate({ to: "/teacher/live" });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (isLoading) {
    return (
      <AppShell eyebrow="Session" title="Loading…">
        <div className="h-40 bg-white/30 border border-brand/10 animate-pulse" />
      </AppShell>
    );
  }
  if (isError || !data) {
    return (
      <AppShell eyebrow="Session" title="Couldn't load session">
        <p className="text-sm text-brand/55">{(error as Error)?.message}</p>
      </AppShell>
    );
  }

  const starts = new Date(data.starts_at);
  const ends = new Date(data.ends_at);
  const isLive = data.status === "live";
  const isEnded = data.status === "ended";
  const isCancelled = data.status === "cancelled";
  const attendees = data.attendees ?? [];

  return (
    <AppShell eyebrow="Session" title={data.title}>
      <Link
        to="/teacher/live"
        className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-6"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Back to live
      </Link>

      <DetailGrid
        items={[
          { label: "Course", value: data.course_title ?? "—" },
          { label: "Starts", value: starts.toLocaleString() },
          { label: "Ends", value: ends.toLocaleString() },
          {
            label: "Status",
            value: <span className="capitalize">{data.status}</span>,
          },
          {
            label: "Registered",
            value: `${data.attendees_count ?? 0}${data.capacity ? ` / ${data.capacity}` : ""}`,
          },
        ]}
      />

      <div className="mt-6 flex flex-wrap gap-3">
        {isLive ? (
          <>
            <Link
              to="/teacher/live/$sessionId/room"
              params={{ sessionId }}
              className="inline-flex items-center gap-2 bg-destructive text-white px-6 py-3 text-sm"
            >
              <Radio className="h-4 w-4 animate-pulse" />
              Open room
            </Link>
            <button
              onClick={() => end.mutate()}
              disabled={end.isPending}
              className="border border-brand/15 px-6 py-3 text-sm hover:bg-brand/[0.03] disabled:opacity-60"
            >
              {end.isPending ? "Ending…" : "End session"}
            </button>
          </>
        ) : isEnded ? (
          data.recording_url ? (
            <a
              href={data.recording_url}
              target="_blank"
              rel="noreferrer"
              className="bg-brand text-white px-6 py-3 text-sm"
            >
              Watch recording
            </a>
          ) : (
            <p className="text-sm text-brand/55">Recording not yet available.</p>
          )
        ) : isCancelled ? (
          <p className="text-sm text-brand/55">Session was cancelled.</p>
        ) : (
          <>
            <button
              onClick={() => start.mutate()}
              disabled={start.isPending}
              className="bg-brand text-white px-6 py-3 text-sm hover:bg-brand/90 disabled:opacity-60"
            >
              {start.isPending ? "Starting…" : "Start session"}
            </button>
            <button
              onClick={() => {
                if (confirm("Cancel this session? Registered students will be notified."))
                  cancel.mutate();
              }}
              disabled={cancel.isPending}
              className="border border-destructive/40 text-destructive px-6 py-3 text-sm hover:bg-destructive/5 disabled:opacity-60"
            >
              Cancel session
            </button>
          </>
        )}
      </div>

      {data.agenda && (
        <section className="mt-10 max-w-2xl">
          <h2 className="font-serif text-xl mb-3">Agenda</h2>
          <p className="text-sm text-brand/75 whitespace-pre-line">{data.agenda}</p>
        </section>
      )}

      <section className="mt-10">
        <h2 className="font-serif text-xl mb-4">Attendees ({attendees.length})</h2>
        {attendees.length === 0 ? (
          <p className="text-sm text-brand/55">No one registered yet.</p>
        ) : (
          <ul className="divide-y divide-brand/10 border-y border-brand/10 max-w-xl">
            {attendees.map((a) => (
              <li key={a.id} className="py-3 px-2 flex justify-between text-sm">
                <span>{a.full_name}</span>
                <span className="text-xs eyebrow text-brand/55">{a.status}</span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </AppShell>
  );
}

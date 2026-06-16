import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { createLiveSession } from "@/lib/api/live";
import { listMyTaughtCourses } from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/live/schedule")({
  component: Page,
});

function toLocalInput(d: Date) {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function Page() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const now = new Date();
  const defaultStart = new Date(now.getTime() + 60 * 60 * 1000);

  const [courseId, setCourseId] = useState("");
  const [title, setTitle] = useState("");
  const [startsAt, setStartsAt] = useState(toLocalInput(defaultStart));
  const [durationMinutes, setDurationMinutes] = useState("60");
  const [recordSession, setRecordSession] = useState(false);

  const { data: courses } = useQuery({
    queryKey: ["teacher-courses-min"],
    queryFn: () => listMyTaughtCourses({ limit: 100 }),
  });

  useEffect(() => {
    if (!courseId && courses?.data?.[0]) setCourseId(courses.data[0].id);
  }, [courses, courseId]);

  const create = useMutation({
    mutationFn: () =>
      createLiveSession({
        course_id: courseId,
        title: title.trim(),
        scheduled_at: new Date(startsAt).toISOString(),
        duration_minutes: Math.max(15, Number(durationMinutes) || 60),
        record_session: recordSession,
      }),
    onSuccess: (session) => {
      toast.success("Session scheduled");
      qc.invalidateQueries({ queryKey: ["teacher-live"] });
      navigate({
        to: "/teacher/live/$sessionId",
        params: { sessionId: session.id },
      });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not schedule"),
  });

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!courseId) return toast.error("Pick a course.");
    if (!title.trim()) return toast.error("Add a title.");
    const s = new Date(startsAt).getTime();
    if (!s) return toast.error("Set a valid start time.");
    if (s < Date.now() - 60_000) return toast.error("Start time is in the past.");
    if (Number(durationMinutes) < 15) return toast.error("Duration must be at least 15 minutes.");
    create.mutate();
  };

  return (
    <AppShell eyebrow="Live" title="Schedule a session">
      <form onSubmit={submit} className="max-w-2xl space-y-5">
        <Field label="Course">
          <select
            value={courseId}
            onChange={(e) => setCourseId(e.target.value)}
            className="w-full p-3 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
          >
            <option value="">Select course…</option>
            {courses?.data.map((c) => (
              <option key={c.id} value={c.id}>
                {c.title}
              </option>
            ))}
          </select>
        </Field>

        <Field label="Title">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            maxLength={200}
            placeholder="e.g. Office hours: portfolio review"
            className="w-full p-3 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
          />
        </Field>

        <div className="grid sm:grid-cols-2 gap-4">
          <Field label="Starts">
            <input
              type="datetime-local"
              value={startsAt}
              onChange={(e) => setStartsAt(e.target.value)}
              className="w-full p-3 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
            />
          </Field>
          <Field label="Duration (minutes)">
            <input
              type="number"
              min={15}
              value={durationMinutes}
              onChange={(e) => setDurationMinutes(e.target.value)}
              className="w-full p-3 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
            />
          </Field>
        </div>
        <label className="flex items-center gap-2 text-sm text-brand/70">
          <input
            type="checkbox"
            checked={recordSession}
            onChange={(e) => setRecordSession(e.target.checked)}
          />
          Record this session
        </label>

        <div className="flex items-center gap-3 pt-2">
          <button
            type="submit"
            disabled={create.isPending}
            className="bg-brand text-white px-6 py-3 text-sm font-medium hover:bg-brand/90 disabled:opacity-60"
          >
            {create.isPending ? "Scheduling…" : "Schedule session"}
          </button>
          <Link
            to="/teacher/live"
            className="text-xs text-brand/55 hover:text-brand underline underline-offset-4"
          >
            Cancel
          </Link>
        </div>
      </form>
    </AppShell>
  );
}

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/55">{label}</span>
      <div className="mt-1.5">{children}</div>
      {hint && <p className="mt-1 text-[11px] text-brand/45">{hint}</p>}
    </label>
  );
}

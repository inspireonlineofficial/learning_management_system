import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { CheckCircle2, ExternalLink, XCircle } from "lucide-react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { DetailGrid } from "@/components/layout/data-page";
import { apiRequest } from "@/lib/api/client";
import { getCourse } from "@/lib/api/courses";

export const Route = createFileRoute("/_authenticated/admin/courses/$courseId/review")({
  component: Page,
});

function Page() {
  const { courseId } = Route.useParams();
  const qc = useQueryClient();
  const [dialog, setDialog] = useState<"approve" | "reject" | null>(null);
  const [note, setNote] = useState("");

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["admin-course-review", courseId],
    queryFn: () => getCourse(courseId),
  });

  const decide = useMutation({
    mutationFn: ({ action, note }: { action: "approve" | "reject"; note?: string }) =>
      apiRequest<{ ok: true }>(`/v1/admin/courses/${courseId}/review`, {
        method: "POST",
        auth: true,
        body: { action, note },
      }),
    onSuccess: () => {
      toast.success("Decision recorded");
      qc.invalidateQueries({ queryKey: ["admin-course-queue"] });
      qc.invalidateQueries({ queryKey: ["admin-course-review", courseId] });
      setDialog(null);
      setNote("");
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (isLoading) {
    return (
      <AppShell eyebrow="Course review" title="Loading…">
        <div className="h-40 bg-white/30 border border-brand/10 animate-pulse" />
      </AppShell>
    );
  }
  if (isError || !data) {
    return (
      <AppShell eyebrow="Course review" title="Couldn't load course">
        <p className="text-sm text-brand/55">{(error as Error)?.message}</p>
      </AppShell>
    );
  }

  const modules = data.modules ?? [];
  const totalLessons = modules.reduce((s, m) => s + m.lessons.length, 0);
  const previewCount = modules.reduce(
    (s, m) => s + m.lessons.filter((l) => l.is_preview).length,
    0,
  );

  return (
    <>
      <AppShell eyebrow="Course review" title={data.title}>
        {data.subtitle && (
          <p className="max-w-2xl text-brand/65 leading-relaxed">{data.subtitle}</p>
        )}

        <div className="mt-8 flex flex-wrap gap-3">
          <Link
            to="/teacher/courses/$courseId/preview"
            params={{ courseId }}
            className="inline-flex items-center gap-2 px-4 py-2 text-xs border border-brand/15 hover:bg-brand/[0.03]"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            Preview as student
          </Link>
        </div>

        <div className="mt-8">
          <DetailGrid
            items={[
              { label: "Teacher", value: data.teacher?.full_name ?? "—" },
              { label: "Category", value: data.category?.name ?? "—" },
              { label: "Level", value: data.level ?? "—" },
              {
                label: "Price",
                value: data.price ? `${data.currency ?? "USD"} ${data.price}` : "Free",
              },
              { label: "Modules", value: String(modules.length) },
              { label: "Lessons", value: `${totalLessons} (${previewCount} preview)` },
            ]}
          />
        </div>

        {data.description && (
          <section className="mt-10 max-w-3xl">
            <h2 className="font-serif text-xl mb-3">Description</h2>
            <p className="text-sm text-brand/75 leading-relaxed whitespace-pre-line">
              {data.description}
            </p>
          </section>
        )}

        {(data.outcomes?.length || data.requirements?.length) && (
          <section className="mt-10 grid md:grid-cols-2 gap-8 max-w-3xl">
            {data.outcomes?.length ? (
              <div>
                <h3 className="eyebrow text-brand/55 mb-3">What students learn</h3>
                <ul className="space-y-1.5 text-sm text-brand/75">
                  {data.outcomes.map((o, i) => (
                    <li key={i} className="flex gap-2">
                      <CheckCircle2 className="h-4 w-4 text-emerald-600 mt-0.5 flex-shrink-0" />
                      {o}
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}
            {data.requirements?.length ? (
              <div>
                <h3 className="eyebrow text-brand/55 mb-3">Requirements</h3>
                <ul className="space-y-1.5 text-sm text-brand/75 list-disc pl-5">
                  {data.requirements.map((r, i) => (
                    <li key={i}>{r}</li>
                  ))}
                </ul>
              </div>
            ) : null}
          </section>
        )}

        <section className="mt-10">
          <h2 className="font-serif text-xl mb-4">Curriculum</h2>
          {modules.length === 0 ? (
            <p className="text-sm text-brand/55">No modules yet — cannot publish.</p>
          ) : (
            <ol className="space-y-3">
              {modules.map((m, mi) => (
                <li key={m.id} className="border border-brand/10 bg-white/40">
                  <div className="px-5 py-3 border-b border-brand/10 flex justify-between text-sm">
                    <span className="font-medium">
                      {String(mi + 1).padStart(2, "0")} · {m.title}
                    </span>
                    <span className="text-xs text-brand/55">
                      {m.lessons.length} lesson{m.lessons.length === 1 ? "" : "s"}
                    </span>
                  </div>
                  <ul className="divide-y divide-brand/5">
                    {m.lessons.map((l) => (
                      <li key={l.id} className="px-5 py-2.5 flex justify-between text-sm">
                        <span className="truncate">
                          {l.title}
                          {l.is_preview && (
                            <span className="ml-2 text-[10px] eyebrow text-accent">preview</span>
                          )}
                        </span>
                        <span className="text-xs text-brand/45 flex-shrink-0 ml-3">
                          {l.type ?? "video"}
                          {l.duration_minutes ? ` · ${l.duration_minutes}m` : ""}
                        </span>
                      </li>
                    ))}
                  </ul>
                </li>
              ))}
            </ol>
          )}
        </section>

        <div className="mt-10 flex flex-wrap gap-3 border-t border-brand/10 pt-6">
          <button
            onClick={() => setDialog("approve")}
            className="bg-brand text-white px-6 py-3 text-sm hover:bg-brand/90"
          >
            Approve & publish
          </button>
          <button
            onClick={() => setDialog("reject")}
            className="border border-destructive/40 text-destructive px-6 py-3 text-sm hover:bg-destructive/5"
          >
            Reject with feedback
          </button>
          <Link to="/admin/courses" className="px-6 py-3 text-sm text-brand/60 hover:text-brand">
            Back to queue
          </Link>
        </div>
      </AppShell>

      {dialog && (
        <div
          className="fixed inset-0 z-50 bg-black/40 grid place-items-center p-4"
          onClick={() => {
            setDialog(null);
            setNote("");
          }}
        >
          <div
            className="bg-white border border-brand/10 max-w-md w-full p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <p className="eyebrow text-brand/55">
              {dialog === "approve" ? "Approve & publish" : "Reject submission"}
            </p>
            <p className="mt-2 font-serif text-xl">{data.title}</p>
            <p className="mt-1 text-xs text-brand/55">
              {data.teacher?.full_name ?? "Unknown teacher"}
            </p>
            <label className="block mt-5">
              <span className="eyebrow text-brand/55">
                {dialog === "approve" ? "Note (optional)" : "Feedback for teacher (required)"}
              </span>
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                rows={5}
                maxLength={2000}
                placeholder={
                  dialog === "approve"
                    ? "Optional note attached to the publish event"
                    : "Explain what needs to change before resubmission"
                }
                className="mt-1 w-full p-3 border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
              />
            </label>
            <div className="mt-5 flex justify-end gap-2">
              <button
                onClick={() => {
                  setDialog(null);
                  setNote("");
                }}
                className="px-4 py-2 text-sm border border-brand/15"
              >
                Cancel
              </button>
              <button
                disabled={decide.isPending || (dialog === "reject" && note.trim().length === 0)}
                onClick={() => decide.mutate({ action: dialog, note: note.trim() || undefined })}
                className={`px-5 py-2 text-sm text-white disabled:opacity-50 ${
                  dialog === "approve" ? "bg-brand" : "bg-destructive"
                }`}
              >
                {decide.isPending
                  ? "Saving…"
                  : dialog === "approve"
                    ? "Approve & publish"
                    : "Send rejection"}
              </button>
            </div>
            {dialog === "reject" && (
              <div className="mt-3 flex items-center gap-1.5 text-xs text-brand/55">
                <XCircle className="h-3.5 w-3.5" />
                Teacher will be notified and can resubmit after edits.
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
}

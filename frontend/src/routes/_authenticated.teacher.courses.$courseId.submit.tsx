import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, AlertCircle, ArrowLeft } from "lucide-react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { apiRequest } from "@/lib/api/client";
import { getTeacherCourse, publishCourse } from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/submit")({
  component: Page,
});

function Page() {
  const { courseId } = Route.useParams();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const { data: course } = useQuery({
    queryKey: ["teacher-course", courseId],
    queryFn: () => getTeacherCourse(courseId),
  });

  const submit = useMutation({
    mutationFn: () =>
      apiRequest<{ ok: true }>(`/v1/teacher/courses/${courseId}/submit`, {
        method: "POST",
        auth: true,
      }),
    onSuccess: () => {
      toast.success("Submitted for admin review");
      qc.invalidateQueries({ queryKey: ["teacher-course", courseId] });
      navigate({ to: "/teacher/courses" });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const publish = useMutation({
    mutationFn: () => publishCourse(courseId),
    onSuccess: () => {
      toast.success("Course published");
      qc.invalidateQueries({ queryKey: ["teacher-course", courseId] });
      navigate({ to: "/teacher/courses" });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const modules = (course as any)?.modules ?? [];
  const lessonCount = modules.reduce((n: number, m: any) => n + (m.lessons?.length ?? 0), 0);
  const checks = [
    { label: "Title set", ok: !!course?.title },
    {
      label: "Subtitle / description",
      ok: !!(course as any)?.subtitle || !!(course as any)?.description,
    },
    { label: "Cover image", ok: !!(course as any)?.cover_url },
    { label: "At least 1 module", ok: modules.length > 0 },
    { label: "At least 3 lessons", ok: lessonCount >= 3 },
  ];
  const ready = checks.every((c) => c.ok);
  const status = (course as any)?.status as string | undefined;

  return (
    <AppShell eyebrow="Submit" title="Submit course for review">
      <Link
        to="/teacher/courses/$courseId/edit"
        params={{ courseId }}
        className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-6"
      >
        <ArrowLeft className="h-3.5 w-3.5" /> Back to editor
      </Link>

      <div className="grid lg:grid-cols-[1fr_320px] gap-10 max-w-4xl">
        <div>
          <p className="text-sm text-brand/65 leading-relaxed mb-8">
            Once submitted, an admin will review your course and either approve it or request
            changes. Make sure your readiness checklist looks complete first.
          </p>

          <h3 className="font-serif text-xl mb-4">Readiness checklist</h3>
          <ul className="border border-brand/10 bg-white/40 divide-y divide-brand/5">
            {checks.map((c) => (
              <li key={c.label} className="flex items-center gap-3 px-4 py-3 text-sm">
                {c.ok ? (
                  <CheckCircle2 className="h-4 w-4 text-emerald-600" />
                ) : (
                  <AlertCircle className="h-4 w-4 text-amber-600" />
                )}
                <span className={c.ok ? "" : "text-brand/55"}>{c.label}</span>
              </li>
            ))}
          </ul>

          <div className="mt-8 flex flex-wrap gap-3">
            <button
              onClick={() => submit.mutate()}
              disabled={!ready || submit.isPending || status === "published"}
              className="bg-brand text-white px-6 py-3 text-sm font-medium disabled:opacity-50"
            >
              {submit.isPending ? "Submitting…" : "Submit for review"}
            </button>
            <button
              onClick={() => publish.mutate()}
              disabled={!ready || publish.isPending || status === "published"}
              className="border border-brand/15 px-6 py-3 text-sm hover:bg-brand/[0.03] disabled:opacity-50"
            >
              {publish.isPending ? "Publishing…" : "Publish directly"}
            </button>
          </div>
          {!ready && (
            <p className="mt-3 text-xs text-amber-700">Complete the checklist before submitting.</p>
          )}
        </div>

        <aside className="border border-brand/10 bg-white/40 p-5 self-start">
          <p className="eyebrow text-brand/45">Current status</p>
          <p className="mt-2 font-serif text-2xl capitalize">{status ?? "draft"}</p>
          <p className="mt-4 text-xs text-brand/55">
            {modules.length} modules · {lessonCount} lessons
          </p>
        </aside>
      </div>
    </AppShell>
  );
}

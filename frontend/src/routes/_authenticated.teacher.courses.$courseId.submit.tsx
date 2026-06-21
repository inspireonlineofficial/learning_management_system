import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, AlertCircle, ArrowLeft } from "lucide-react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { apiRequest } from "@/lib/api/client";
import { getTeacherCourse } from "@/lib/api/teacher";

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

  const readiness = getCoursePublishReadiness(course);
  const modules = (course as any)?.modules ?? [];
  const lessons = getCourseLessons(course);
  const publishedLessons = lessons.filter((lesson: any) => lesson.status === "published");
  const checks = readiness.checks;
  const ready = readiness.ready;
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
            Submit this course for admin review when the required items are complete. After an admin
            approves it, students can see it in the public course catalog.
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
          </div>
          {!ready && (
            <p className="mt-3 text-xs text-amber-700">
              Complete the missing checklist items in the editor before submitting.
            </p>
          )}
        </div>

        <aside className="border border-brand/10 bg-white/40 p-5 self-start">
          <p className="eyebrow text-brand/45">Current status</p>
          <p className="mt-2 font-serif text-2xl capitalize">{status ?? "draft"}</p>
          <p className="mt-4 text-xs text-brand/55">
            {modules.length} modules · {lessons.length} lessons · {publishedLessons.length}{" "}
            published lessons
          </p>
        </aside>
      </div>
    </AppShell>
  );
}

function getCourseLessons(course: any) {
  return ((course as any)?.modules ?? []).flatMap((module: any) =>
    (module.chapters ?? []).flatMap((chapter: any) => chapter.lessons ?? []),
  );
}

function getCoursePublishReadiness(course: any) {
  const modules = (course as any)?.modules ?? [];
  const lessons = getCourseLessons(course);
  const publishedLessons = lessons.filter((lesson: any) => lesson.status === "published");
  const checks = [
    { label: "Title set", ok: Boolean((course as any)?.title?.trim?.() || (course as any)?.title) },
    {
      label: "Subtitle / description",
      ok: Boolean((course as any)?.subtitle?.trim?.() || (course as any)?.description?.trim?.()),
    },
    { label: "At least 1 module", ok: modules.length > 0 },
    { label: "At least 1 published lesson", ok: publishedLessons.length > 0 },
  ];

  return {
    checks,
    ready: checks.every((check) => check.ok),
  };
}

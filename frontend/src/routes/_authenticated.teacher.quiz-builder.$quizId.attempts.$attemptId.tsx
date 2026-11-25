import { createFileRoute, Link, useBlocker } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Check, ChevronLeft, ChevronRight } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import { QUESTION_TYPE_LABEL } from "@/lib/assessment";
import { RichContent, ZoomableImage } from "@/components/assessment/rich-content";
import { AppShell } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { toFrontendType } from "@/lib/api/quizzes";
import {
  getTeacherAttempt,
  gradeAttempt,
  listQuizAttempts,
  type QuestionOutcome,
  type TeacherAttemptDetail,
} from "@/lib/api/teacher-quizzes";

export const Route = createFileRoute(
  "/_authenticated/teacher/quiz-builder/$quizId/attempts/$attemptId",
)({
  component: Page,
});

type Draft = Record<string, { awarded: string; feedback: string }>;

function draftFrom(detail: TeacherAttemptDetail): Draft {
  const d: Draft = {};
  for (const o of detail.results) {
    if (o.type === "long_answer" || o.type === "short_answer") {
      d[o.question_id] = {
        awarded: o.status === "pending" ? "" : String(o.awarded),
        feedback: o.feedback ?? "",
      };
    }
  }
  return d;
}

function Page() {
  const { quizId, attemptId } = Route.useParams();
  const qc = useQueryClient();
  const detailQuery = useQuery({
    queryKey: ["teacher-attempt", quizId, attemptId],
    queryFn: () => getTeacherAttempt(quizId, attemptId),
  });
  // Queue of attempts that still need marking, for "next" navigation.
  const queue = useQuery({
    queryKey: ["quiz-attempts", quizId, { status: "pending_grading", limit: 100 }],
    queryFn: () =>
      listQuizAttempts(quizId, {
        status: "pending_grading",
        limit: 100,
        sort: "submitted_at",
        order: "asc",
      }),
  });

  const detail = detailQuery.data;
  const [draft, setDraft] = useState<Draft>({});
  useEffect(() => {
    if (detail) setDraft(draftFrom(detail));
  }, [detail]);
  const original = useMemo(() => (detail ? draftFrom(detail) : {}), [detail]);
  const dirty = JSON.stringify(draft) !== JSON.stringify(original);

  // Warn before leaving with unsaved marks.
  useBlocker({
    shouldBlockFn: () => dirty && !window.confirm("You have unsaved marks. Leave without saving?"),
    enableBeforeUnload: () => dirty,
  });

  const save = useMutation({
    mutationFn: () => {
      const grades = Object.entries(draft)
        .filter(
          ([id, g]) =>
            g.awarded.trim() !== "" &&
            (g.awarded !== original[id]?.awarded || g.feedback !== original[id]?.feedback),
        )
        .map(([question_id, g]) => ({
          question_id,
          awarded: Number(g.awarded),
          feedback: g.feedback,
        }));
      if (grades.length === 0)
        return Promise.reject(new Error("Enter marks for at least one answer."));
      return gradeAttempt(quizId, attemptId, grades);
    },
    onSuccess: (updated) => {
      qc.setQueryData(["teacher-attempt", quizId, attemptId], updated);
      qc.invalidateQueries({ queryKey: ["quiz-attempts", quizId] });
      qc.invalidateQueries({ queryKey: ["teacher-quiz", quizId] });
      toast.success(
        updated.grading_status === "pending"
          ? "Marks saved — some answers still need marking"
          : "Attempt fully graded",
      );
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (detailQuery.isError) {
    return (
      <AppShell eyebrow="Marking" title="Attempt unavailable">
        <QueryErrorPanel
          error={detailQuery.error}
          title="Couldn't load this attempt"
          onRetry={() => detailQuery.refetch()}
        />
      </AppShell>
    );
  }
  if (!detail) {
    return (
      <AppShell eyebrow="Marking" title="Loading…">
        <div className="h-40 animate-pulse border border-brand/10 bg-white/30" />
      </AppShell>
    );
  }

  const pendingIds = (queue.data?.attempts ?? [])
    .map((a) => a.attempt_id)
    .filter((id) => id !== attemptId);
  const nextId = pendingIds[0];
  const invalid = Object.entries(draft).some(([id, g]) => {
    if (g.awarded.trim() === "") return false;
    const o = detail.results.find((r) => r.question_id === id);
    const n = Number(g.awarded);
    return !o || Number.isNaN(n) || n < 0 || n > o.marks;
  });
  const inProgress = detail.status === "in_progress";

  return (
    <AppShell
      eyebrow={detail.quiz_title}
      title={`${detail.student_name || "Student"} — attempt ${detail.attempt_number}`}
    >
      <div className="mb-6 flex flex-wrap items-center gap-3 text-sm">
        <Link
          to="/teacher/quiz-builder/$quizId/attempts"
          params={{ quizId }}
          className="inline-flex items-center gap-2 text-xs text-brand/70 hover:text-brand"
        >
          <ArrowLeft className="h-3.5 w-3.5" aria-hidden /> All attempts
        </Link>
        <span className="text-xs text-brand/70">{detail.student_email}</span>
        <span className="ml-auto text-xs text-brand/70 tabular-nums">
          {detail.grading_status === "pending"
            ? "Needs marking"
            : detail.grading_status === "graded"
              ? "Graded"
              : inProgress
                ? "In progress"
                : "Auto-graded"}
          {detail.score_points != null && ` · ${detail.score_points}/${detail.max_points} marks`}
          {detail.score_percent != null && ` · ${detail.score_percent}%`}
        </span>
      </div>

      {inProgress && (
        <p className="mb-6 border-l-2 border-sky-500 bg-sky-50 px-4 py-3 text-sm text-sky-900">
          The student is still working on this attempt. Marking opens after submission.
        </p>
      )}
      {!detail.review_available && !inProgress && (
        <p className="mb-6 text-sm text-brand/70">
          This attempt was submitted before per-question records existed; only the total score is
          available.
        </p>
      )}

      <ol className="space-y-5">
        {detail.results.map((o, i) => (
          <OutcomeCard
            key={o.question_id}
            o={o}
            number={i + 1}
            draft={draft[o.question_id]}
            onDraft={(next) => setDraft((d) => ({ ...d, [o.question_id]: next }))}
            disabled={inProgress}
          />
        ))}
      </ol>

      {!inProgress && Object.keys(draft).length > 0 && (
        <div className="sticky bottom-0 mt-8 flex flex-wrap items-center gap-3 border-t border-brand/10 bg-surface/95 py-4 backdrop-blur">
          <button
            type="button"
            onClick={() => save.mutate()}
            disabled={!dirty || invalid || save.isPending}
            className="inline-flex items-center gap-2 bg-brand px-5 py-2.5 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-40"
          >
            <Check className="h-4 w-4" aria-hidden /> {save.isPending ? "Saving…" : "Save marks"}
          </button>
          {invalid && (
            <span className="text-sm text-destructive">
              Marks must be between 0 and the question's maximum.
            </span>
          )}
          {dirty && !invalid && <span className="text-xs text-amber-700">Unsaved changes</span>}
          {nextId && (
            <Link
              to="/teacher/quiz-builder/$quizId/attempts/$attemptId"
              params={{ quizId, attemptId: nextId }}
              className="ml-auto inline-flex items-center gap-1 border border-brand/15 px-4 py-2.5 text-sm hover:border-brand/40"
            >
              Next to mark ({pendingIds.length}) <ChevronRight className="h-4 w-4" aria-hidden />
            </Link>
          )}
          {!nextId && (
            <Link
              to="/teacher/quiz-builder/$quizId/attempts"
              params={{ quizId }}
              className="ml-auto inline-flex items-center gap-1 text-sm text-brand/70 hover:text-brand"
            >
              <ChevronLeft className="h-4 w-4" aria-hidden /> Back to attempts
            </Link>
          )}
        </div>
      )}
    </AppShell>
  );
}

function OutcomeCard({
  o,
  number,
  draft,
  onDraft,
  disabled,
}: {
  o: QuestionOutcome;
  number: number;
  draft?: { awarded: string; feedback: string };
  onDraft: (d: { awarded: string; feedback: string }) => void;
  disabled: boolean;
}) {
  const type = toFrontendType(o.type);
  const manual = Boolean(draft);
  const selected = new Set(o.selected_option_ids);
  const tone =
    o.status === "correct"
      ? "text-emerald-700"
      : o.status === "pending"
        ? "text-accent-ink"
        : o.status === "incorrect"
          ? "text-destructive"
          : "text-brand/70";
  return (
    <li
      className={`border bg-white/60 p-5 ${o.status === "pending" ? "border-accent/40" : "border-brand/10"}`}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="font-serif text-lg">Question {number}</h3>
        <p className="text-xs">
          <span className="text-brand/70">{QUESTION_TYPE_LABEL[type]} · </span>
          <span className={`${tone} capitalize`}>
            {o.status === "pending" ? "needs marking" : o.status}
          </span>
          <span className="tabular-nums text-brand/70">
            {" "}
            · {o.status === "pending" ? "—" : o.awarded}/{o.marks}
          </span>
        </p>
      </div>
      <div className="mt-3">
        <RichContent text={o.body} />
        {o.image_url && (
          <div className="mt-3">
            <ZoomableImage src={o.image_url} alt={`Figure for question ${number}`} />
          </div>
        )}
      </div>

      {o.options && o.options.length > 0 ? (
        <ul className="mt-4 grid gap-1.5 text-sm">
          {o.options
            .slice()
            .sort((a, b) => a.position - b.position)
            .map((opt) => (
              <li
                key={opt.id}
                className={`flex gap-3 border px-3 py-2 ${opt.is_correct ? "border-emerald-600/40 bg-emerald-50/60" : selected.has(opt.id) ? "border-destructive/30 bg-destructive/5" : "border-brand/10"}`}
              >
                <span className="w-28 shrink-0 text-xs text-brand/70">
                  {selected.has(opt.id) ? "Student" : ""}
                  {opt.is_correct ? (selected.has(opt.id) ? " ✓" : "Correct") : ""}
                </span>
                <RichContent text={opt.body} />
              </li>
            ))}
        </ul>
      ) : (
        <div className="mt-4 text-sm">
          <p className="eyebrow mb-1 text-brand/70">Student answer</p>
          {o.answered ? (
            <div className="border border-brand/10 bg-white px-3 py-2">
              {type === "long_answer" ? (
                <RichContent text={o.text_answer} />
              ) : (
                <span className="whitespace-pre-wrap">{o.text_answer}</span>
              )}
              {o.answer_image_url && (
                <div className="mt-2">
                  <ZoomableImage src={o.answer_image_url} alt="Student's attached working" />
                </div>
              )}
            </div>
          ) : (
            <p className="text-brand/70">No answer.</p>
          )}
          {o.accepted_answers && o.accepted_answers.length > 0 && (
            <p className="mt-2 text-xs text-brand/70">Accepted: {o.accepted_answers.join(" · ")}</p>
          )}
          {o.numeric_answer != null && (
            <p className="mt-2 text-xs text-brand/70 tabular-nums">
              Correct value: {o.numeric_answer}
              {o.unit ? ` ${o.unit}` : ""}
              {o.tolerance ? ` (± ${o.tolerance}${o.tolerance_type === "percent" ? "%" : ""})` : ""}
            </p>
          )}
        </div>
      )}

      {o.explanation && (
        <details className="mt-3 text-sm">
          <summary className="cursor-pointer text-xs text-brand/70">
            Model answer / explanation
          </summary>
          <div className="mt-2 border-l-2 border-accent bg-accent/5 px-3 py-2">
            <RichContent text={o.explanation} />
          </div>
        </details>
      )}

      {manual && draft && (
        <div className="mt-4 grid gap-3 border-t border-brand/10 pt-4 sm:grid-cols-[10rem_1fr]">
          <label className="text-sm">
            <span className="font-medium text-brand/80">Marks (0–{o.marks})</span>
            <input
              type="number"
              min={0}
              max={o.marks}
              step={0.5}
              value={draft.awarded}
              disabled={disabled}
              onChange={(e) => onDraft({ ...draft, awarded: e.target.value })}
              className="mt-1.5 w-full border border-brand/15 bg-white px-3 py-2 tabular-nums"
              aria-invalid={
                draft.awarded !== "" &&
                (Number(draft.awarded) < 0 || Number(draft.awarded) > o.marks)
              }
            />
          </label>
          <label className="text-sm">
            <span className="font-medium text-brand/80">Feedback to the student (optional)</span>
            <textarea
              rows={2}
              value={draft.feedback}
              disabled={disabled}
              onChange={(e) => onDraft({ ...draft, feedback: e.target.value })}
              className="mt-1.5 w-full border border-brand/15 bg-white px-3 py-2"
            />
          </label>
          {type === "short_answer" && o.status !== "pending" && (
            <p className="text-xs text-brand/70 sm:col-span-2">
              Auto-marked against the accepted answers; you can override the marks (e.g. for a
              correct spelling variant).
            </p>
          )}
        </div>
      )}
    </li>
  );
}

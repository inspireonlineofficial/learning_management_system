import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import {
  AlertTriangle,
  Calculator,
  Check,
  ChevronLeft,
  ChevronRight,
  Circle,
  CloudOff,
  Flag,
  Keyboard,
  LayoutGrid,
  Loader2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { QuestionView } from "@/components/assessment/question-view";
import { RichContent } from "@/components/assessment/rich-content";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  clearAttemptBackup,
  useAttemptAutosave,
  type SaveStatus,
} from "@/hooks/use-attempt-autosave";
import { ApiError } from "@/lib/api/client";
import {
  checkQuizAnswer,
  isAnswered,
  resumeQuizAttempt,
  startQuizAttempt,
  submitQuizAttempt,
  uploadAnswerImage,
  type QuizAttempt,
} from "@/lib/api/quizzes";

export const Route = createFileRoute("/_authenticated/student/assessments/$quizId/attempt")({
  validateSearch: z.object({ attempt: z.string().optional() }),
  component: QuizRunner,
});

const flagsKey = (attemptId: string) => `inspire:attempt-flags:${attemptId}`;

function readFlags(attemptId: string): Set<string> {
  try {
    return new Set(
      JSON.parse(window.localStorage.getItem(flagsKey(attemptId)) ?? "[]") as string[],
    );
  } catch {
    return new Set();
  }
}

function formatClock(totalSeconds: number) {
  const s = Math.max(0, Math.floor(totalSeconds));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const mm = String(m).padStart(h ? 2 : 1, "0");
  return h ? `${h}:${mm}:${String(sec).padStart(2, "0")}` : `${mm}:${String(sec).padStart(2, "0")}`;
}

function isTypingTarget(target: EventTarget | null) {
  const el = target as HTMLElement | null;
  if (!el) return false;
  const tag = el.tagName;
  if (tag === "TEXTAREA" || el.isContentEditable) return true;
  if (tag === "INPUT") {
    const type = (el as HTMLInputElement).type;
    return !["radio", "checkbox", "button", "submit"].includes(type);
  }
  return false;
}

function QuizRunner() {
  const { quizId } = Route.useParams();
  const { attempt: attemptIdFromUrl } = Route.useSearch();
  const navigate = useNavigate();

  const attemptQuery = useQuery({
    // Keyed by quiz only: adding ?attempt= to the URL after starting must not
    // trigger a second fetch that would replace the loaded attempt.
    queryKey: ["quiz-start", quizId],
    staleTime: Infinity,
    gcTime: 0,
    retry: (count, error) => count < 2 && !(error instanceof ApiError && error.status < 500),
    queryFn: () =>
      attemptIdFromUrl ? resumeQuizAttempt(quizId, attemptIdFromUrl) : startQuizAttempt(quizId),
  });
  const attempt = attemptQuery.data;

  useEffect(() => {
    if (!attempt) return;
    if (attempt.status !== "in_progress") {
      navigate({
        to: "/student/assessments/$quizId/result/$attemptId",
        params: { quizId, attemptId: attempt.id },
        replace: true,
      });
    } else if (!attemptIdFromUrl) {
      navigate({
        to: "/student/assessments/$quizId/attempt",
        params: { quizId },
        search: { attempt: attempt.id },
        replace: true,
      });
    }
  }, [attempt, attemptIdFromUrl, navigate, quizId]);

  if (attemptQuery.isError) {
    const err = attemptQuery.error as Error;
    return (
      <CenteredMessage title="This assessment can't be opened">
        <p className="text-sm text-brand/65">{err.message}</p>
        <Link
          to="/student/assessments/$quizId"
          params={{ quizId }}
          className="mt-6 inline-block bg-brand px-6 py-3 text-sm text-white"
        >
          Back to the assessment
        </Link>
      </CenteredMessage>
    );
  }
  if (!attempt || attempt.status !== "in_progress") {
    return (
      <CenteredMessage title="Preparing your attempt…">
        <Loader2 className="mx-auto h-6 w-6 animate-spin text-brand/70" aria-hidden />
      </CenteredMessage>
    );
  }
  if (attempt.questions.length === 0) {
    return (
      <CenteredMessage title="This assessment has no questions yet">
        <p className="text-sm text-brand/65">Please check back later or contact your teacher.</p>
      </CenteredMessage>
    );
  }
  return <ExamWorkspace key={attempt.id} attempt={attempt} quizId={quizId} />;
}

function CenteredMessage({ title, children }: { title: string; children?: React.ReactNode }) {
  return (
    <main className="min-h-screen grid place-items-center bg-surface px-6 text-brand">
      <div className="max-w-md text-center">
        <h1 className="font-serif text-2xl mb-4">{title}</h1>
        {children}
      </div>
    </main>
  );
}

function ExamWorkspace({ attempt, quizId }: { attempt: QuizAttempt; quizId: string }) {
  const navigate = useNavigate();
  const questions = attempt.questions;
  const practice = attempt.mode === "practice";

  const [idx, setIdx] = useState(0);
  const [visited, setVisited] = useState<Set<number>>(() => new Set([0]));
  const [flags, setFlags] = useState<Set<string>>(() => readFlags(attempt.id));
  const [navOpen, setNavOpen] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [shortcutsOpen, setShortcutsOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [checks, setChecks] = useState<Record<string, { correct: boolean; explanation?: string }>>(
    {},
  );
  const [checking, setChecking] = useState(false);
  const [announcement, setAnnouncement] = useState("");
  const submittedRef = useRef(false);
  const questionTopRef = useRef<HTMLDivElement>(null);

  const goToResult = useCallback(() => {
    clearAttemptBackup(attempt.id);
    navigate({
      to: "/student/assessments/$quizId/result/$attemptId",
      params: { quizId, attemptId: attempt.id },
      replace: true,
    });
  }, [attempt.id, navigate, quizId]);

  const autosave = useAttemptAutosave({
    attemptId: attempt.id,
    serverAnswers: attempt.answers,
    serverRevision: attempt.draft_revision,
    onClosed: (code) => {
      if (submittedRef.current) return;
      submittedRef.current = true;
      toast.info(
        code === "ATTEMPT_EXPIRED"
          ? "Time is up. Your saved answers were submitted."
          : "This attempt has already been submitted.",
      );
      goToResult();
    },
  });
  const { answers, setAnswer } = autosave;

  // ---- Server-synchronised timer ------------------------------------------
  const clockOffset = useMemo(() => {
    const server = attempt.server_now ? new Date(attempt.server_now).getTime() : Date.now();
    return server - Date.now(); // add to Date.now() to get server time
  }, [attempt.server_now]);
  const deadline = attempt.expires_at ? new Date(attempt.expires_at).getTime() : null;
  const [remaining, setRemaining] = useState<number | null>(() =>
    deadline ? (deadline - (Date.now() + clockOffset)) / 1000 : null,
  );
  const warnedRef = useRef<Set<number>>(new Set());

  const submit = useCallback(
    async (reason: "manual" | "timeout") => {
      if (submittedRef.current && reason === "manual" && !submitError) return;
      submittedRef.current = true;
      setSubmitting(true);
      setSubmitError(null);
      try {
        await autosave.saveNow().catch(() => false);
        await submitQuizAttempt(attempt.id, autosave.answersRef.current);
        autosave.markClosed();
        if (reason === "timeout") toast.info("Time is up. Your answers were submitted.");
        goToResult();
      } catch (error) {
        const apiError = error instanceof ApiError ? error : null;
        if (apiError?.code === "ATTEMPT_ALREADY_SUBMITTED") {
          goToResult();
          return;
        }
        // Submission is idempotent server-side, so retrying is always safe.
        submittedRef.current = false;
        setSubmitError(
          typeof navigator !== "undefined" && !navigator.onLine
            ? "You are offline. Your answers are saved on this device — reconnect and submit again."
            : apiError?.message || "Submission failed. Your answers are safe; please try again.",
        );
        setReviewOpen(true);
      } finally {
        setSubmitting(false);
      }
    },
    [attempt.id, autosave, goToResult, submitError],
  );

  useEffect(() => {
    if (!deadline) return;
    const tick = () => {
      const secs = (deadline - (Date.now() + clockOffset)) / 1000;
      setRemaining(secs);
      for (const mark of [300, 60]) {
        if (secs <= mark && secs > 0 && !warnedRef.current.has(mark)) {
          warnedRef.current.add(mark);
          const msg = mark === 300 ? "5 minutes remaining." : "1 minute remaining.";
          toast.warning(msg);
          setAnnouncement(msg);
        }
      }
      if (secs <= 0 && !submittedRef.current) void submit("timeout");
    };
    tick();
    const id = window.setInterval(tick, 1000);
    return () => window.clearInterval(id);
  }, [deadline, clockOffset, submit]);

  // ---- Navigation ---------------------------------------------------------
  const goTo = useCallback(
    (next: number) => {
      const clamped = Math.max(0, Math.min(questions.length - 1, next));
      setIdx(clamped);
      setVisited((v) => new Set(v).add(clamped));
      setNavOpen(false);
      requestAnimationFrame(() => {
        questionTopRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
        questionTopRef.current?.focus({ preventScroll: true });
      });
    },
    [questions.length],
  );

  const toggleFlag = useCallback(
    (questionId: string) => {
      setFlags((current) => {
        const next = new Set(current);
        if (next.has(questionId)) next.delete(questionId);
        else next.add(questionId);
        try {
          window.localStorage.setItem(flagsKey(attempt.id), JSON.stringify([...next]));
        } catch {
          /* flags are a convenience; ignore storage errors */
        }
        return next;
      });
    },
    [attempt.id],
  );

  const question = questions[idx];
  const answeredCount = questions.filter((q) => isAnswered(answers[q.id])).length;
  const unanswered = questions.length - answeredCount;
  const flaggedCount = questions.filter((q) => flags.has(q.id)).length;

  // Keyboard shortcuts (ignored while typing in a text field).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (reviewOpen || shortcutsOpen || e.ctrlKey || e.metaKey || e.altKey) return;
      if (isTypingTarget(e.target)) return;
      const key = e.key.toLowerCase();
      if (key === "arrowright" || key === "n") {
        e.preventDefault();
        goTo(idx + 1);
      } else if (key === "arrowleft" || key === "p") {
        e.preventDefault();
        goTo(idx - 1);
      } else if (key === "f") {
        e.preventDefault();
        toggleFlag(question.id);
      } else if (key === "g") {
        e.preventDefault();
        setNavOpen((o) => !o);
      } else if (key === "?") {
        setShortcutsOpen(true);
      } else if (/^[1-9]$/.test(key) && question.options?.length) {
        const opt = question.options[Number(key) - 1];
        if (!opt) return;
        e.preventDefault();
        if (question.type === "multi_select") {
          const current = Array.isArray(answers[question.id])
            ? (answers[question.id] as string[])
            : [];
          setAnswer(
            question.id,
            current.includes(opt.id) ? current.filter((id) => id !== opt.id) : [...current, opt.id],
          );
        } else {
          setAnswer(question.id, opt.id);
        }
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [answers, goTo, idx, question, reviewOpen, setAnswer, shortcutsOpen, toggleFlag]);

  const runCheck = async () => {
    const value = answers[question.id];
    if (!isAnswered(value)) {
      toast.info("Answer the question first.");
      return;
    }
    setChecking(true);
    try {
      const result = await checkQuizAnswer(attempt.id, question.id, value);
      setChecks((c) => ({
        ...c,
        [question.id]: { correct: result.correct, explanation: result.explanation },
      }));
    } catch (error) {
      toast.error((error as Error).message || "Could not check the answer.");
    } finally {
      setChecking(false);
    }
  };

  const timerUrgent = remaining != null && remaining <= 300;

  return (
    <div className="min-h-screen bg-surface text-brand">
      <a
        href="#question"
        className="sr-only focus:not-sr-only focus:absolute focus:left-2 focus:top-2 focus:z-50 focus:bg-white focus:px-3 focus:py-2"
      >
        Skip to question
      </a>
      <p className="sr-only" aria-live="assertive">
        {announcement}
      </p>

      {/* Header */}
      <header className="sticky top-0 z-30 border-b border-brand/10 bg-white/95 backdrop-blur supports-[backdrop-filter]:bg-white/85">
        <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-3 sm:px-6">
          <div className="min-w-0 flex-1">
            <p className="truncate font-serif text-base sm:text-lg" title={attempt.quiz_title}>
              {attempt.quiz_title ?? "Assessment"}
            </p>
            <p className="text-xs text-brand/70 tabular-nums">
              Question {idx + 1} of {questions.length}
              <span className="hidden sm:inline">
                {" "}
                · {answeredCount} answered · {unanswered} unanswered
                {flaggedCount > 0 && ` · ${flaggedCount} flagged`}
              </span>
              {practice && (
                <span className="ml-2 border border-accent/40 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-accent-ink">
                  Practice
                </span>
              )}
            </p>
          </div>
          <SaveIndicator status={autosave.status} />
          {remaining != null && (
            <div
              className={`flex items-center gap-1.5 border px-2.5 py-1.5 text-sm font-medium tabular-nums ${
                timerUrgent
                  ? "border-destructive/40 bg-destructive/5 text-destructive"
                  : "border-brand/15"
              }`}
              role="timer"
              aria-label={`Time remaining ${formatClock(remaining)}`}
            >
              <span aria-hidden>{formatClock(remaining)}</span>
            </div>
          )}
          <button
            type="button"
            onClick={() => setNavOpen((o) => !o)}
            className="lg:hidden inline-flex items-center gap-1.5 border border-brand/15 px-2.5 py-1.5 text-sm"
            aria-expanded={navOpen}
            aria-controls="question-navigator"
          >
            <LayoutGrid className="h-4 w-4" aria-hidden />
            <span className="sr-only sm:not-sr-only">Questions</span>
          </button>
          <button
            type="button"
            onClick={() => setReviewOpen(true)}
            disabled={submitting}
            className="bg-brand px-3.5 py-2 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-60 sm:px-5"
          >
            {submitting ? "Submitting…" : "Submit"}
          </button>
        </div>
        <div
          className="h-1 bg-brand/10"
          role="progressbar"
          aria-label="Questions answered"
          aria-valuemin={0}
          aria-valuemax={questions.length}
          aria-valuenow={answeredCount}
        >
          <div
            className="h-full bg-brand transition-[width] duration-300 motion-reduce:transition-none"
            style={{ width: `${(answeredCount / questions.length) * 100}%` }}
          />
        </div>
      </header>

      {autosave.status === "conflict" && (
        <div
          role="alert"
          className="border-b border-amber-300 bg-amber-50 px-4 py-2 text-center text-sm text-amber-900"
        >
          This attempt was updated in another tab or on another device, so saving here has paused.{" "}
          <button
            type="button"
            className="font-medium underline underline-offset-2"
            onClick={() => window.location.reload()}
          >
            Reload to continue
          </button>{" "}
          with the latest saved answers.
        </div>
      )}
      {autosave.status === "auth" && (
        <div
          role="alert"
          className="border-b border-amber-300 bg-amber-50 px-4 py-2 text-center text-sm text-amber-900"
        >
          Your session has expired. Your answers are kept on this device — sign in again in another
          tab, then come back; saving resumes automatically.
        </div>
      )}

      <div className="mx-auto grid max-w-6xl gap-8 px-4 py-6 sm:px-6 lg:grid-cols-[minmax(0,1fr)_17rem] lg:py-10">
        <main id="question" className="min-w-0">
          {idx === 0 &&
            (attempt.instructions ||
              attempt.calculator_policy !== "none" ||
              attempt.reference_material) && (
              <details
                className="mb-6 border border-brand/10 bg-white/60 px-4 py-3 text-sm"
                open={!autosave.lastSavedAt}
              >
                <summary className="cursor-pointer font-medium">Instructions</summary>
                <div className="mt-3 space-y-3 text-brand/75">
                  {attempt.instructions && <RichContent text={attempt.instructions} />}
                  {attempt.calculator_policy !== "none" && (
                    <p className="flex items-center gap-2">
                      <Calculator className="h-4 w-4" aria-hidden /> A {attempt.calculator_policy}{" "}
                      calculator is permitted.
                    </p>
                  )}
                  {attempt.reference_material && (
                    <div>
                      <p className="eyebrow text-brand/70 mb-1">Permitted resources</p>
                      <RichContent text={attempt.reference_material} />
                    </div>
                  )}
                </div>
              </details>
            )}

          <div ref={questionTopRef} tabIndex={-1} className="scroll-mt-24 focus:outline-none">
            <QuestionView
              key={question.id}
              question={question}
              number={idx + 1}
              value={answers[question.id]}
              onChange={(value) => setAnswer(question.id, value)}
              showHint={practice}
              feedback={practice ? checks[question.id] : null}
              onUploadImage={(file) => uploadAnswerImage(attempt.id, file)}
            />
          </div>

          <div className="mt-8 flex flex-wrap items-center gap-3 border-t border-brand/10 pt-5">
            <button
              type="button"
              onClick={() => toggleFlag(question.id)}
              aria-pressed={flags.has(question.id)}
              className={`inline-flex items-center gap-2 border px-3.5 py-2.5 text-sm ${
                flags.has(question.id)
                  ? "border-amber-500 bg-amber-50 text-amber-900"
                  : "border-brand/15 hover:border-brand/40"
              }`}
            >
              <Flag
                className="h-4 w-4"
                aria-hidden
                fill={flags.has(question.id) ? "currentColor" : "none"}
              />
              {flags.has(question.id) ? "Flagged for review" : "Flag for review"}
            </button>
            {practice && (
              <button
                type="button"
                onClick={runCheck}
                disabled={checking}
                className="inline-flex items-center gap-2 border border-accent/50 px-3.5 py-2.5 text-sm text-accent-ink hover:bg-accent/5 disabled:opacity-60"
              >
                {checking ? (
                  <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                ) : (
                  <Check className="h-4 w-4" aria-hidden />
                )}
                Check answer
              </button>
            )}
            <div className="ml-auto flex gap-2">
              <button
                type="button"
                onClick={() => goTo(idx - 1)}
                disabled={idx === 0}
                className="inline-flex items-center gap-1 border border-brand/15 px-4 py-2.5 text-sm hover:border-brand/40 disabled:opacity-40"
              >
                <ChevronLeft className="h-4 w-4" aria-hidden /> Previous
              </button>
              {idx < questions.length - 1 ? (
                <button
                  type="button"
                  onClick={() => goTo(idx + 1)}
                  className="inline-flex items-center gap-1 bg-brand px-4 py-2.5 text-sm text-white hover:bg-brand/90"
                >
                  Next <ChevronRight className="h-4 w-4" aria-hidden />
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => setReviewOpen(true)}
                  className="inline-flex items-center gap-1 bg-brand px-4 py-2.5 text-sm text-white hover:bg-brand/90"
                >
                  Review &amp; submit
                </button>
              )}
            </div>
          </div>

          <button
            type="button"
            onClick={() => setShortcutsOpen(true)}
            className="mt-6 hidden items-center gap-2 text-xs text-brand/70 hover:text-brand md:inline-flex"
          >
            <Keyboard className="h-3.5 w-3.5" aria-hidden /> Keyboard shortcuts (?)
          </button>
        </main>

        {/* Navigator: sidebar on desktop, overlay panel on small screens */}
        <aside
          id="question-navigator"
          aria-label="Question navigator"
          className={`${
            navOpen
              ? "fixed inset-x-0 bottom-0 z-40 max-h-[75vh] overflow-y-auto border-t border-brand/15 bg-white p-5 shadow-lg"
              : "hidden"
          } lg:static lg:block lg:max-h-none lg:border-0 lg:bg-transparent lg:p-0 lg:shadow-none`}
        >
          <div className="lg:sticky lg:top-24">
            <div className="flex items-center justify-between">
              <h2 className="font-serif text-lg">Questions</h2>
              <button
                type="button"
                className="p-2 lg:hidden"
                onClick={() => setNavOpen(false)}
                aria-label="Close question navigator"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <p className="mt-1 text-xs text-brand/70 tabular-nums">
              {answeredCount} of {questions.length} answered
            </p>
            <ol className="mt-4 grid grid-cols-6 gap-2 sm:grid-cols-8 lg:grid-cols-5">
              {questions.map((q, i) => {
                const answered = isAnswered(answers[q.id]);
                const flagged = flags.has(q.id);
                const current = i === idx;
                const state = answered
                  ? "answered"
                  : visited.has(i)
                    ? "visited, unanswered"
                    : "not visited";
                return (
                  <li key={q.id}>
                    <button
                      type="button"
                      onClick={() => goTo(i)}
                      aria-current={current ? "step" : undefined}
                      aria-label={`Question ${i + 1}: ${state}${flagged ? ", flagged" : ""}`}
                      className={`relative grid h-11 w-full place-items-center border text-sm tabular-nums transition-colors ${
                        current ? "ring-2 ring-accent ring-offset-1" : ""
                      } ${answered ? "border-brand bg-brand text-white" : visited.has(i) ? "border-brand/40 bg-white" : "border-brand/15 bg-white/60 text-brand/70"}`}
                    >
                      {i + 1}
                      {flagged && (
                        <Flag
                          className="absolute -right-1 -top-1 h-3.5 w-3.5 fill-amber-500 text-amber-600"
                          aria-hidden
                        />
                      )}
                    </button>
                  </li>
                );
              })}
            </ol>
            <ul className="mt-5 space-y-1.5 text-xs text-brand/65">
              <li className="flex items-center gap-2">
                <span className="grid h-4 w-4 place-items-center bg-brand text-white">
                  <Check className="h-3 w-3" aria-hidden />
                </span>{" "}
                Answered
              </li>
              <li className="flex items-center gap-2">
                <span className="h-4 w-4 border border-brand/40 bg-white" /> Visited, unanswered
              </li>
              <li className="flex items-center gap-2">
                <Circle className="h-4 w-4 text-brand/30" aria-hidden /> Not visited
              </li>
              <li className="flex items-center gap-2">
                <Flag className="h-4 w-4 fill-amber-500 text-amber-600" aria-hidden /> Flagged for
                review
              </li>
            </ul>
            <button
              type="button"
              onClick={() => setReviewOpen(true)}
              className="mt-6 w-full border border-brand px-4 py-2.5 text-sm font-medium hover:bg-brand hover:text-white"
            >
              Review &amp; submit
            </button>
            <Link
              to="/student/assessments/$quizId"
              params={{ quizId }}
              className="mt-3 block text-center text-xs text-brand/70 underline underline-offset-4 hover:text-brand"
            >
              Leave (answers are saved)
            </Link>
          </div>
        </aside>
      </div>

      {/* Review before submit */}
      <Dialog open={reviewOpen} onOpenChange={(o) => !submitting && setReviewOpen(o)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="font-serif text-2xl">Submit your answers?</DialogTitle>
            <DialogDescription>Once submitted, answers can't be changed.</DialogDescription>
          </DialogHeader>
          <dl className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
            <Stat label="Questions" value={questions.length} />
            <Stat label="Answered" value={answeredCount} />
            <Stat label="Unanswered" value={unanswered} warn={unanswered > 0} />
            <Stat label="Flagged" value={flaggedCount} warn={flaggedCount > 0} />
          </dl>
          {remaining != null && (
            <p className="text-sm text-brand/70">
              Time remaining:{" "}
              <span className="tabular-nums font-medium">{formatClock(remaining)}</span>
            </p>
          )}
          {(unanswered > 0 || flaggedCount > 0) && (
            <div className="max-h-40 overflow-y-auto border border-brand/10 p-3">
              <p className="text-xs text-brand/70 mb-2">Go back to:</p>
              <div className="flex flex-wrap gap-2">
                {questions.map((q, i) =>
                  !isAnswered(answers[q.id]) || flags.has(q.id) ? (
                    <button
                      key={q.id}
                      type="button"
                      onClick={() => {
                        setReviewOpen(false);
                        goTo(i);
                      }}
                      className="inline-flex items-center gap-1 border border-brand/20 px-2.5 py-1 text-xs hover:border-brand"
                    >
                      {i + 1}
                      {!isAnswered(answers[q.id]) && (
                        <span className="text-destructive">· unanswered</span>
                      )}
                      {flags.has(q.id) && (
                        <Flag
                          className="h-3 w-3 fill-amber-500 text-amber-600"
                          aria-label="flagged"
                        />
                      )}
                    </button>
                  ) : null,
                )}
              </div>
            </div>
          )}
          {submitError && (
            <p
              role="alert"
              className="flex items-start gap-2 border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"
            >
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden /> {submitError}
            </p>
          )}
          <DialogFooter className="gap-2 sm:gap-0">
            <button
              type="button"
              onClick={() => setReviewOpen(false)}
              disabled={submitting}
              className="border border-brand/15 px-5 py-2.5 text-sm hover:border-brand/40"
            >
              Keep working
            </button>
            <button
              type="button"
              onClick={() => void submit("manual")}
              disabled={submitting}
              className="inline-flex items-center justify-center gap-2 bg-brand px-5 py-2.5 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-60"
            >
              {submitting && <Loader2 className="h-4 w-4 animate-spin" aria-hidden />}
              {submitting ? "Submitting…" : submitError ? "Try again" : "Submit now"}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={shortcutsOpen} onOpenChange={setShortcutsOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>Keyboard shortcuts</DialogTitle>
            <DialogDescription>
              Shortcuts are paused while you type in an answer box.
            </DialogDescription>
          </DialogHeader>
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
            {[
              ["→ or N", "Next question"],
              ["← or P", "Previous question"],
              ["1 – 9", "Choose option 1–9"],
              ["F", "Flag / unflag"],
              ["G", "Open question navigator"],
              ["?", "Show this list"],
            ].map(([k, v]) => (
              <div key={k} className="contents">
                <dt>
                  <kbd className="border border-brand/20 bg-brand/[0.04] px-1.5 py-0.5 font-mono text-xs">
                    {k}
                  </kbd>
                </dt>
                <dd className="text-brand/70">{v}</dd>
              </div>
            ))}
          </dl>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function Stat({ label, value, warn = false }: { label: string; value: number; warn?: boolean }) {
  return (
    <div className="border border-brand/10 px-3 py-2">
      <dt className="text-xs text-brand/70">{label}</dt>
      <dd className={`font-serif text-2xl tabular-nums ${warn ? "text-destructive" : ""}`}>
        {value}
      </dd>
    </div>
  );
}

function SaveIndicator({ status }: { status: SaveStatus }) {
  const map: Record<SaveStatus, { text: string; icon: React.ReactNode; tone: string }> = {
    idle: {
      text: "All changes saved",
      icon: <Check className="h-3.5 w-3.5" aria-hidden />,
      tone: "text-brand/70",
    },
    saved: {
      text: "Saved",
      icon: <Check className="h-3.5 w-3.5" aria-hidden />,
      tone: "text-emerald-700",
    },
    pending: {
      text: "Saving…",
      icon: <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />,
      tone: "text-brand/70",
    },
    saving: {
      text: "Saving…",
      icon: <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />,
      tone: "text-brand/70",
    },
    offline: {
      text: "Offline — kept on this device",
      icon: <CloudOff className="h-3.5 w-3.5" aria-hidden />,
      tone: "text-amber-700",
    },
    retrying: {
      text: "Save failed — retrying",
      icon: <AlertTriangle className="h-3.5 w-3.5" aria-hidden />,
      tone: "text-amber-700",
    },
    auth: {
      text: "Session expired",
      icon: <AlertTriangle className="h-3.5 w-3.5" aria-hidden />,
      tone: "text-destructive",
    },
    conflict: {
      text: "Open in another tab",
      icon: <AlertTriangle className="h-3.5 w-3.5" aria-hidden />,
      tone: "text-amber-700",
    },
    closed: {
      text: "Submitted",
      icon: <Check className="h-3.5 w-3.5" aria-hidden />,
      tone: "text-brand/70",
    },
  };
  const s = map[status];
  return (
    <p
      className={`hidden items-center gap-1.5 text-xs md:flex ${s.tone}`}
      role="status"
      aria-live="polite"
    >
      {s.icon}
      <span>{s.text}</span>
    </p>
  );
}

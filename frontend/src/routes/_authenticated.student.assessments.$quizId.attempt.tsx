import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { z } from "zod";

import {
  getQuizAttempt,
  saveQuizAnswers,
  startQuizAttempt,
  submitQuizAttempt,
  type QuizAttempt,
  type QuizQuestion,
} from "@/lib/api/quizzes";

export const Route = createFileRoute("/_authenticated/student/assessments/$quizId/attempt")({
  validateSearch: z.object({ attempt: z.string().optional() }),
  component: QuizRunner,
});

type Answers = Record<string, string[] | string>;

function QuizRunner() {
  const { quizId } = Route.useParams();
  const { attempt: attemptIdFromUrl } = Route.useSearch();
  const navigate = useNavigate();

  // Resume or start the attempt
  const { data: startedAttempt } = useQuery({
    queryKey: ["quiz-start", quizId, attemptIdFromUrl],
    queryFn: () =>
      attemptIdFromUrl
        ? (getQuizAttempt(attemptIdFromUrl) as Promise<QuizAttempt>)
        : startQuizAttempt(quizId),
  });

  const [answers, setAnswers] = useState<Answers>({});
  const [idx, setIdx] = useState(0);
  const [remaining, setRemaining] = useState<number | null>(null);
  const initializedRef = useRef(false);

  useEffect(() => {
    if (!startedAttempt || initializedRef.current) return;
    initializedRef.current = true;
    setAnswers(startedAttempt.answers ?? {});
    setIdx(startedAttempt.current_question_index ?? 0);
    if (startedAttempt.expires_at) {
      const ms = new Date(startedAttempt.expires_at).getTime() - Date.now();
      setRemaining(Math.max(0, Math.floor(ms / 1000)));
    }
    // Ensure URL has the attempt id so refresh resumes
    if (!attemptIdFromUrl) {
      navigate({
        to: "/student/assessments/$quizId/attempt",
        params: { quizId },
        search: { attempt: startedAttempt.id },
        replace: true,
      });
    }
  }, [startedAttempt, attemptIdFromUrl, navigate, quizId]);

  // Countdown
  useEffect(() => {
    if (remaining == null) return;
    if (remaining <= 0) {
      submitNow(true);
      return;
    }
    const t = setTimeout(() => setRemaining((r) => (r == null ? r : r - 1)), 1000);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [remaining]);

  // Autosave
  const saveMutation = useMutation({
    mutationFn: (next: Answers) => saveQuizAnswers(startedAttempt!.id, next),
  });
  useEffect(() => {
    if (!startedAttempt || !initializedRef.current) return;
    const t = setTimeout(() => saveMutation.mutate(answers), 800);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [answers]);

  const submitMutation = useMutation({
    mutationFn: () => submitQuizAttempt(startedAttempt!.id, answers),
    onSuccess: (result) => {
      toast.success(result.passed ? "Quiz passed." : "Quiz submitted.");
      navigate({
        to: "/student/assessments/result/$attemptId",
        params: { attemptId: result.id },
        replace: true,
      });
    },
    onError: (e: Error) => toast.error(e.message ?? "Submission failed"),
  });

  const submitNow = (auto = false) => {
    if (submitMutation.isPending) return;
    if (!auto) {
      const unanswered = (startedAttempt?.questions ?? []).filter(
        (q) => !isAnswered(answers[q.id]),
      ).length;
      if (
        unanswered > 0 &&
        !window.confirm(
          `You have ${unanswered} unanswered question${unanswered === 1 ? "" : "s"}. Submit anyway?`,
        )
      )
        return;
    }
    submitMutation.mutate();
  };

  const questions = startedAttempt?.questions ?? [];
  const q = questions[idx];
  const answeredCount = useMemo(
    () => questions.filter((qq) => isAnswered(answers[qq.id])).length,
    [questions, answers],
  );

  if (!startedAttempt || !q) {
    return (
      <div className="min-h-screen grid place-items-center bg-surface text-brand font-sans">
        <p className="text-sm text-brand/55">Preparing your quiz…</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      {/* Header */}
      <header className="border-b border-brand/10 px-6 md:px-12 py-5 flex items-center justify-between gap-4">
        <div>
          <p className="eyebrow text-accent">
            Question {idx + 1} of {questions.length}
          </p>
          <p className="text-xs text-brand/50 mt-1">
            {answeredCount} answered · {saveMutation.isPending ? "saving…" : "saved"}
          </p>
        </div>
        {remaining != null && (
          <div
            className={`px-4 py-2 font-mono text-sm border ${
              remaining < 60
                ? "border-destructive/40 text-destructive bg-destructive/5"
                : "border-brand/15 text-brand"
            }`}
          >
            {formatTime(remaining)}
          </div>
        )}
      </header>

      <div className="grid lg:grid-cols-[1fr_280px] gap-10 px-6 md:px-12 py-10 max-w-6xl mx-auto">
        <article>
          <h1 className="font-serif text-2xl lg:text-3xl leading-snug text-balance">{q.prompt}</h1>
          <p className="mt-2 text-xs text-brand/45">
            {q.points} point{q.points === 1 ? "" : "s"}
          </p>

          <div className="mt-8">
            <QuestionInput
              question={q}
              value={answers[q.id]}
              onChange={(v) => setAnswers((prev) => ({ ...prev, [q.id]: v }))}
            />
          </div>

          <div className="mt-10 flex items-center justify-between pt-6 border-t border-brand/10">
            <button
              disabled={idx === 0}
              onClick={() => setIdx((i) => Math.max(0, i - 1))}
              className="px-5 py-3 border border-brand/15 text-sm disabled:opacity-40 hover:bg-brand/[0.03]"
            >
              Previous
            </button>
            {idx < questions.length - 1 ? (
              <button
                onClick={() => setIdx((i) => Math.min(questions.length - 1, i + 1))}
                className="bg-brand text-white px-6 py-3 text-sm font-medium hover:bg-brand/90"
              >
                Next
              </button>
            ) : (
              <button
                onClick={() => submitNow()}
                disabled={submitMutation.isPending}
                className="bg-accent text-white px-6 py-3 text-sm font-medium hover:bg-accent/90 disabled:opacity-60"
              >
                {submitMutation.isPending ? "Submitting…" : "Submit quiz"}
              </button>
            )}
          </div>
        </article>

        {/* Question navigator */}
        <aside className="lg:sticky lg:top-8 lg:self-start">
          <p className="eyebrow text-brand/40 mb-3">Navigator</p>
          <div className="grid grid-cols-6 lg:grid-cols-5 gap-2">
            {questions.map((qq, i) => {
              const answered = isAnswered(answers[qq.id]);
              const active = i === idx;
              return (
                <button
                  key={qq.id}
                  onClick={() => setIdx(i)}
                  className={`aspect-square text-xs font-medium border transition-colors ${
                    active
                      ? "bg-brand text-white border-brand"
                      : answered
                        ? "bg-accent/10 text-brand border-accent/40"
                        : "bg-white text-brand/55 border-brand/15 hover:border-brand/40"
                  }`}
                >
                  {i + 1}
                </button>
              );
            })}
          </div>
          <button
            onClick={() => submitNow()}
            disabled={submitMutation.isPending}
            className="mt-6 w-full bg-brand text-white py-3 text-sm font-medium hover:bg-brand/90 disabled:opacity-60"
          >
            Submit quiz
          </button>
        </aside>
      </div>
    </div>
  );
}

function isAnswered(v: unknown) {
  if (v == null) return false;
  if (typeof v === "string") return v.trim() !== "";
  if (Array.isArray(v)) return v.length > 0;
  return true;
}

function formatTime(s: number) {
  const m = Math.floor(s / 60);
  const r = s % 60;
  return `${String(m).padStart(2, "0")}:${String(r).padStart(2, "0")}`;
}

function QuestionInput({
  question,
  value,
  onChange,
}: {
  question: QuizQuestion;
  value: string[] | string | undefined;
  onChange: (v: string[] | string) => void;
}) {
  if (question.type === "short_answer") {
    return (
      <textarea
        value={typeof value === "string" ? value : ""}
        onChange={(e) => onChange(e.target.value)}
        rows={6}
        maxLength={2000}
        placeholder="Type your answer…"
        className="w-full p-4 bg-white border border-brand/15 focus:border-brand/40 focus:outline-none text-sm leading-relaxed"
      />
    );
  }

  if (question.type === "true_false") {
    const v = Array.isArray(value) ? value[0] : value;
    return (
      <div className="space-y-2">
        {["true", "false"].map((opt) => (
          <label
            key={opt}
            className={`flex items-center gap-3 p-4 border cursor-pointer transition-colors ${
              v === opt
                ? "border-brand bg-brand/[0.04]"
                : "border-brand/15 hover:border-brand/40 bg-white"
            }`}
          >
            <input
              type="radio"
              checked={v === opt}
              onChange={() => onChange(opt)}
              className="accent-brand"
            />
            <span className="text-sm capitalize">{opt}</span>
          </label>
        ))}
      </div>
    );
  }

  const selected = new Set(Array.isArray(value) ? value : value ? [value] : []);
  const isMulti = question.type === "multi_select";

  return (
    <div className="space-y-2">
      {(question.options ?? []).map((opt) => {
        const active = selected.has(opt.id);
        return (
          <label
            key={opt.id}
            className={`flex items-start gap-3 p-4 border cursor-pointer transition-colors ${
              active
                ? "border-brand bg-brand/[0.04]"
                : "border-brand/15 hover:border-brand/40 bg-white"
            }`}
          >
            <input
              type={isMulti ? "checkbox" : "radio"}
              checked={active}
              onChange={() => {
                if (isMulti) {
                  const next = new Set(selected);
                  if (active) next.delete(opt.id);
                  else next.add(opt.id);
                  onChange(Array.from(next));
                } else {
                  onChange([opt.id]);
                }
              }}
              className="mt-1 accent-brand"
            />
            <span className="text-sm leading-relaxed">{opt.text}</span>
          </label>
        );
      })}
    </div>
  );
}

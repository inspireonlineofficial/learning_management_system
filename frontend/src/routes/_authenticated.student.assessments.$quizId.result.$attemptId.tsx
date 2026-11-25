import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { CheckCircle2, Clock3, XCircle } from "lucide-react";

import { AttemptReview } from "@/components/assessment/attempt-review";
import { AppShell } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { getQuizResult, type QuizResult } from "@/lib/api/quizzes";

export const Route = createFileRoute(
  "/_authenticated/student/assessments/$quizId/result/$attemptId",
)({
  component: QuizResultPage,
});

function formatDuration(seconds?: number | null) {
  if (seconds == null) return "—";
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return m ? `${m} min ${s} s` : `${s} s`;
}

function QuizResultPage() {
  const { quizId, attemptId } = Route.useParams();
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["quiz-attempt", quizId, attemptId],
    queryFn: () => getQuizResult(quizId, attemptId),
    // Pending manual grading: refresh occasionally while the page is open.
    refetchInterval: (query) => (query.state.data?.grading_status === "pending" ? 60_000 : false),
  });

  if (isLoading) {
    return (
      <AppShell eyebrow="Result" title="Loading your result…">
        <div className="h-40 animate-pulse border border-brand/10 bg-white/30" />
      </AppShell>
    );
  }
  if (isError || !data) {
    return (
      <AppShell eyebrow="Result" title="Result unavailable">
        <QueryErrorPanel
          error={error}
          title="Couldn't load this result"
          onRetry={() => refetch()}
        />
      </AppShell>
    );
  }
  return <ResultView result={data} quizId={quizId} />;
}

function ResultView({ result, quizId }: { result: QuizResult; quizId: string }) {
  const pending = result.grading_status === "pending";
  const percent =
    result.score_percent ??
    (result.max_points
      ? Math.round(((result.score_points ?? 0) / result.max_points) * 1000) / 10
      : 0);
  const counts = result.review.reduce(
    (acc, q) => {
      if (q.pending) acc.pending++;
      else if (!q.answered) acc.unanswered++;
      else if (q.status === "correct") acc.correct++;
      else if (q.status === "incorrect") acc.incorrect++;
      else if (q.status === "partial") acc.partial++;
      return acc;
    },
    { correct: 0, incorrect: 0, partial: 0, unanswered: 0, pending: 0 },
  );

  return (
    <AppShell eyebrow="Result" title={result.quiz_title ?? "Assessment result"}>
      <section
        className={`border p-6 sm:p-8 ${
          pending
            ? "border-brand/15 bg-white/60"
            : result.passed
              ? "border-emerald-600/30 bg-emerald-50/60"
              : "border-amber-600/30 bg-amber-50/50"
        }`}
        aria-labelledby="score-heading"
      >
        <div className="flex flex-wrap items-start gap-5">
          {pending ? (
            <Clock3 className="h-10 w-10 text-brand/70" aria-hidden />
          ) : result.passed ? (
            <CheckCircle2 className="h-10 w-10 text-emerald-700" aria-hidden />
          ) : (
            <XCircle className="h-10 w-10 text-amber-700" aria-hidden />
          )}
          <div className="min-w-0 flex-1">
            <h2 id="score-heading" className="font-serif text-3xl sm:text-4xl">
              {pending
                ? "Submitted — awaiting marking"
                : result.passed
                  ? "Passed"
                  : "Not passed yet"}
            </h2>
            <p className="mt-2 text-sm text-brand/70">
              {pending
                ? "Some written answers are marked by your teacher. Your final score appears here once marking is complete."
                : result.passing_score_percent != null
                  ? `Passing score: ${result.passing_score_percent}%`
                  : null}
            </p>
          </div>
          <div className="text-right">
            <p className="font-serif text-5xl tabular-nums">{percent}%</p>
            <p className="text-sm tabular-nums text-brand/65">
              {result.score_points ?? 0} / {result.max_points ?? 0} marks{pending ? " so far" : ""}
            </p>
          </div>
        </div>
        <dl className="mt-6 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
          <div>
            <dt className="text-xs text-brand/70">Time taken</dt>
            <dd className="tabular-nums">{formatDuration(result.time_taken_seconds)}</dd>
          </div>
          <div>
            <dt className="text-xs text-brand/70">Submitted</dt>
            <dd>{result.submitted_at ? new Date(result.submitted_at).toLocaleString() : "—"}</dd>
          </div>
          {result.answers_released && (
            <div>
              <dt className="text-xs text-brand/70">Correct / partly</dt>
              <dd className="tabular-nums">
                {counts.correct} / {counts.partial}
              </dd>
            </div>
          )}
          <div>
            <dt className="text-xs text-brand/70">Unanswered</dt>
            <dd className="tabular-nums">{counts.unanswered}</dd>
          </div>
          {result.points_awarded > 0 && (
            <div>
              <dt className="text-xs text-brand/70">Points earned</dt>
              <dd className="tabular-nums">+{result.points_awarded}</dd>
            </div>
          )}
        </dl>
        {result.status === "auto_submitted" && (
          <p className="mt-4 text-xs text-brand/70">
            Submitted automatically when the time ran out, using your saved answers.
          </p>
        )}
      </section>

      <section className="mt-10">
        <h2 className="font-serif text-2xl">Review</h2>
        {!result.answers_released && (
          <p className="mt-2 text-sm text-brand/65">
            Your teacher has not released correct answers for this assessment. You can see your own
            answers below.
          </p>
        )}
        {result.review_available ? (
          <div className="mt-5">
            <AttemptReview review={result.review} released={result.answers_released} />
          </div>
        ) : (
          <p className="mt-4 text-sm text-brand/70">
            A question-by-question review isn't available for attempts submitted before the
            assessment upgrade.
          </p>
        )}
      </section>

      <div className="mt-10 flex flex-wrap gap-3">
        <Link
          to="/student/assessments/$quizId"
          params={{ quizId }}
          className="bg-brand px-6 py-3 text-sm font-medium text-white hover:bg-brand/90"
        >
          Back to the assessment
        </Link>
        <Link
          to="/student/assessments"
          className="border border-brand/15 px-6 py-3 text-sm hover:border-brand/40"
        >
          All assessments
        </Link>
      </div>
    </AppShell>
  );
}

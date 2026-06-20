import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { CheckCircle2, XCircle } from "lucide-react";

import { AppShell } from "@/components/layout/app-shell";
import { getQuizAttempt, type QuizResult } from "@/lib/api/quizzes";

export const Route = createFileRoute(
  "/_authenticated/student/assessments/$quizId/result/$attemptId",
)({
  component: QuizResultPage,
});

function QuizResultPage() {
  const { quizId, attemptId } = Route.useParams();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["quiz-attempt", quizId, attemptId],
    queryFn: () => getQuizAttempt(attemptId),
  });

  if (isLoading) {
    return (
      <AppShell eyebrow="Result" title="Loading your result...">
        <div className="h-40 bg-white/30 border border-brand/10 animate-pulse" />
      </AppShell>
    );
  }

  if (isError || !data) {
    return (
      <AppShell eyebrow="Result" title="Couldn't load this result">
        <p className="text-sm text-brand/55">{(error as Error)?.message}</p>
      </AppShell>
    );
  }

  const result = data as QuizResult;
  const hasGrade = typeof result.percentage === "number";

  return (
    <AppShell
      eyebrow="Quiz result"
      title={
        hasGrade ? (result.passed ? "You passed." : "Not passing yet.") : "Submission received"
      }
    >
      {hasGrade && (
        <div className="grid sm:grid-cols-3 gap-4 max-w-2xl">
          <Stat label="Score" value={`${result.score} / ${result.max_score}`} />
          <Stat label="Percentage" value={`${Math.round(result.percentage)}%`} />
          <Stat
            label="Outcome"
            value={result.passed ? "Passed" : "Failed"}
            tone={result.passed ? "good" : "bad"}
          />
        </div>
      )}

      {result.feedback && (
        <section className="mt-10 max-w-2xl">
          <h2 className="font-serif text-xl mb-2">Feedback</h2>
          <p className="text-sm text-brand/70 leading-relaxed whitespace-pre-line">
            {result.feedback}
          </p>
        </section>
      )}

      <div className="mt-10 flex gap-3">
        <Link to="/student/assessments" className="bg-brand text-white px-6 py-3 text-sm">
          Back to assessments
        </Link>
        <Link
          to="/student/assessments/$quizId"
          params={{ quizId }}
          className="border border-brand/15 px-6 py-3 text-sm"
        >
          Quiz details
        </Link>
      </div>

      {result.questions && result.questions.length > 0 && (
        <section className="mt-12">
          <h2 className="font-serif text-2xl mb-6">Review</h2>
          <ol className="space-y-5">
            {result.questions.map((question, index) => {
              const perQuestion = result.per_question?.find(
                (item) => item.question_id === question.id,
              );
              return (
                <li key={question.id} className="border border-brand/10 bg-white/50 p-5">
                  <div className="flex items-start gap-3">
                    {perQuestion?.correct ? (
                      <CheckCircle2 className="h-5 w-5 text-emerald-600 mt-0.5 shrink-0" />
                    ) : (
                      <XCircle className="h-5 w-5 text-destructive mt-0.5 shrink-0" />
                    )}
                    <div>
                      <p className="text-xs text-brand/45">
                        Q{index + 1} · {perQuestion?.awarded ?? 0} /{" "}
                        {perQuestion?.max ?? question.points}
                      </p>
                      <p className="mt-1 font-serif text-lg">{question.prompt}</p>
                      {question.image_url && (
                        <img
                          src={question.image_url}
                          alt=""
                          className="mt-3 max-h-64 w-full max-w-xl object-contain border border-brand/10 bg-white"
                        />
                      )}
                      {question.explanation && (
                        <p className="mt-3 text-xs text-brand/60">{question.explanation}</p>
                      )}
                    </div>
                  </div>
                </li>
              );
            })}
          </ol>
        </section>
      )}
    </AppShell>
  );
}

function Stat({ label, value, tone }: { label: string; value: string; tone?: "good" | "bad" }) {
  const toneClass =
    tone === "good" ? "text-emerald-700" : tone === "bad" ? "text-destructive" : "text-brand";
  return (
    <div className="border border-brand/10 bg-white/40 p-5">
      <p className="eyebrow text-brand/45">{label}</p>
      <p className={`mt-2 font-serif text-3xl ${toneClass}`}>{value}</p>
    </div>
  );
}

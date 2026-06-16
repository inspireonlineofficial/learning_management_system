import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Clock, FileQuestion, RotateCw } from "lucide-react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { getQuiz, startQuizAttempt } from "@/lib/api/quizzes";

export const Route = createFileRoute("/_authenticated/student/assessments/$quizId/")({
  component: QuizOverview,
});

function QuizOverview() {
  const { quizId } = Route.useParams();
  const navigate = useNavigate();

  const {
    data: quiz,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ["quiz", quizId],
    queryFn: () => getQuiz(quizId),
  });

  const startMutation = useMutation({
    mutationFn: () => startQuizAttempt(quizId),
    onSuccess: (attempt) => {
      navigate({
        to: "/student/assessments/$quizId/attempt",
        params: { quizId },
        search: { attempt: attempt.id } as never,
      });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not start the quiz"),
  });

  if (isLoading) {
    return (
      <AppShell eyebrow="Quiz" title="Loading…">
        <div className="h-40 bg-white/30 border border-brand/10 animate-pulse" />
      </AppShell>
    );
  }
  if (isError || !quiz) {
    return (
      <AppShell eyebrow="Quiz" title="Couldn't load this quiz">
        <p className="text-sm text-brand/55">{(error as Error)?.message}</p>
      </AppShell>
    );
  }

  const attemptsLeft =
    quiz.attempts_allowed == null
      ? null
      : Math.max(0, quiz.attempts_allowed - (quiz.attempts_used ?? 0));

  return (
    <AppShell eyebrow={quiz.course_title ?? "Quiz"} title={quiz.title}>
      {quiz.description && (
        <p className="text-brand/70 leading-relaxed max-w-2xl">{quiz.description}</p>
      )}

      <div className="mt-10 grid sm:grid-cols-2 lg:grid-cols-4 gap-4 max-w-3xl">
        <Meta icon={FileQuestion} label="Questions" value={String(quiz.total_questions)} />
        <Meta icon={RotateCw} label="Points" value={String(quiz.total_points)} />
        <Meta
          icon={Clock}
          label="Time limit"
          value={quiz.time_limit_minutes ? `${quiz.time_limit_minutes} min` : "Untimed"}
        />
        <Meta
          icon={RotateCw}
          label="Attempts"
          value={
            quiz.attempts_allowed == null
              ? "Unlimited"
              : `${quiz.attempts_used ?? 0} / ${quiz.attempts_allowed}`
          }
        />
      </div>

      {quiz.instructions && (
        <section className="mt-10 max-w-2xl">
          <h2 className="font-serif text-xl mb-3">Instructions</h2>
          <p className="text-sm text-brand/70 leading-relaxed whitespace-pre-line">
            {quiz.instructions}
          </p>
        </section>
      )}

      <div className="mt-10 flex flex-wrap gap-3 items-center">
        <button
          disabled={startMutation.isPending || attemptsLeft === 0}
          onClick={() => startMutation.mutate()}
          className="bg-brand text-white px-7 py-4 text-sm font-medium hover:bg-brand/90 transition-colors disabled:opacity-50"
        >
          {startMutation.isPending
            ? "Preparing your attempt…"
            : quiz.status === "in_progress"
              ? "Resume attempt"
              : "Begin attempt"}
        </button>
        <Link
          to="/student/assessments"
          className="text-xs text-brand/55 hover:text-brand underline underline-offset-4"
        >
          Back to assessments
        </Link>
        {attemptsLeft === 0 && <p className="text-xs text-destructive">No attempts remaining.</p>}
      </div>
    </AppShell>
  );
}

function Meta({ icon: Icon, label, value }: { icon: typeof Clock; label: string; value: string }) {
  return (
    <div className="border border-brand/10 bg-white/40 p-5">
      <Icon className="h-4 w-4 text-brand/40 mb-3" />
      <p className="eyebrow text-brand/45">{label}</p>
      <p className="mt-2 font-serif text-2xl">{value}</p>
    </div>
  );
}

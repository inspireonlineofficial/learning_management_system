import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Navigate } from "@tanstack/react-router";

import { AppShell } from "@/components/layout/app-shell";
import { getQuizAttempt } from "@/lib/api/quizzes";

export const Route = createFileRoute("/_authenticated/student/assessments/result/$attemptId")({
  component: QuizResultRedirectPage,
});

function QuizResultRedirectPage() {
  const { attemptId } = Route.useParams();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["quiz-attempt-redirect", attemptId],
    queryFn: () => getQuizAttempt(attemptId),
  });

  if (data?.quiz_id) {
    return (
      <Navigate
        to="/student/assessments/$quizId/result/$attemptId"
        params={{ quizId: data.quiz_id, attemptId }}
        replace
      />
    );
  }

  return (
    <AppShell eyebrow="Result" title={isLoading ? "Loading your result..." : "Result unavailable"}>
      {isError && <p className="text-sm text-brand/55">{(error as Error)?.message}</p>}
    </AppShell>
  );
}

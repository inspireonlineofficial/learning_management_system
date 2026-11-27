import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { AppShell, SectionHeading } from "@/components/layout/app-shell";
import { deleteQuiz, listTeacherQuizzes } from "@/lib/api/teacher-quizzes";

export const Route = createFileRoute("/_authenticated/teacher/quiz-builder/")({
  component: Page,
});

function Page() {
  const qc = useQueryClient();

  const quizzes = useQuery({
    queryKey: ["teacher-quizzes"],
    queryFn: () => listTeacherQuizzes({ limit: 100 }),
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteQuiz(id),
    onSuccess: () => {
      toast.success("Deleted");
      qc.invalidateQueries({ queryKey: ["teacher-quizzes"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <AppShell eyebrow="Quizzes" title="Quiz builder">
      <SectionHeading
        title="Your quizzes"
        action={
          <Link
            to="/teacher/quiz-builder/new"
            className="inline-flex items-center gap-2 px-3 py-2 text-xs font-medium border border-brand/15 hover:bg-brand/[0.03]"
          >
            <Plus className="h-3.5 w-3.5" />
            New quiz
          </Link>
        }
      />

      {quizzes.isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-14 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : !quizzes.data || quizzes.data.data.length === 0 ? (
        <p className="text-sm text-slatebody rounded-lg border border-borderui bg-white p-8 text-center">
          No quizzes yet.
        </p>
      ) : (
        <ul className="border border-brand/10 bg-white/40 divide-y divide-brand/10">
          {quizzes.data.data.map((q) => (
            <li key={q.id} className="p-4 flex items-center gap-3">
              <Link
                to="/teacher/quiz-builder/$quizId/edit"
                params={{ quizId: q.id }}
                className="flex-1 hover:text-accent-ink"
              >
                <p className="font-serif text-lg">{q.title}</p>
                <p className="text-xs text-brand/70">
                  {q.course_title ?? "Course"} · {q.total_questions} questions · {q.total_points}{" "}
                  marks · passing {q.passing_score}%
                </p>
                <p className="mt-1 text-[10px] uppercase tracking-[0.12em] text-brand/70">
                  {q.lifecycle} · {q.mode} · {q.is_free ? "Free" : "Paid"} · {q.submitted_count}{" "}
                  submitted
                  {q.in_progress_count > 0 ? ` · ${q.in_progress_count} in progress` : ""}
                </p>
              </Link>
              {q.pending_grading_count > 0 && (
                <Link
                  to="/teacher/quiz-builder/$quizId/attempts"
                  params={{ quizId: q.id }}
                  search={{ status: "pending_grading" }}
                  className="bg-accent-ink px-2 py-1 text-xs font-medium text-white tabular-nums"
                >
                  {q.pending_grading_count} to mark
                </Link>
              )}
              <Link
                to="/teacher/quiz-builder/$quizId/attempts"
                params={{ quizId: q.id }}
                className="border border-brand/15 px-2.5 py-1.5 text-xs hover:bg-brand/[0.03]"
              >
                Attempts
              </Link>
              <button
                onClick={() => {
                  if (
                    confirm(
                      "Delete this quiz and all its questions? Quizzes with student attempts can only be unpublished.",
                    )
                  )
                    remove.mutate(q.id);
                }}
                className="p-1.5 text-brand/70 hover:text-destructive"
                aria-label={`Delete ${q.title}`}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </AppShell>
  );
}

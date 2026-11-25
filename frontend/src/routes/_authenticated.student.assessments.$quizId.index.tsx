import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Award, Clock, FileQuestion, RotateCw } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { RichContent } from "@/components/assessment/rich-content";
import { AppShell } from "@/components/layout/app-shell";
import { useAuth } from "@/context/auth-context";
import {
  createCourseComment,
  deleteCourseComment,
  listCourseComments,
  updateCourseComment,
  type CourseComment,
} from "@/lib/api/courses";
import { getQuiz, startQuizAttempt } from "@/lib/api/quizzes";

export const Route = createFileRoute("/_authenticated/student/assessments/$quizId/")({
  component: QuizOverview,
});

function QuizOverview() {
  const { quizId } = Route.useParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { user } = useAuth();

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

  const comments = useQuery({
    queryKey: ["quiz-comments", quiz?.course_id, quizId],
    queryFn: () => listCourseComments(quiz!.course_id, { limit: 100 }),
    enabled: Boolean(quiz?.course_id),
  });

  const createComment = useMutation({
    mutationFn: (input: { content: string; parent_comment_id?: string }) =>
      createCourseComment(quiz!.course_id, { ...input, quiz_id: quizId }),
    onSuccess: () => {
      toast.success("Comment posted.");
      qc.invalidateQueries({ queryKey: ["quiz-comments", quiz?.course_id, quizId] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not post comment"),
  });

  const updateComment = useMutation({
    mutationFn: (input: { commentId: string; content?: string; is_pinned?: boolean }) =>
      updateCourseComment(input.commentId, {
        content: input.content,
        is_pinned: input.is_pinned,
      }),
    onSuccess: () => {
      toast.success("Comment updated.");
      qc.invalidateQueries({ queryKey: ["quiz-comments", quiz?.course_id, quizId] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not update comment"),
  });

  const deleteComment = useMutation({
    mutationFn: (commentId: string) => deleteCourseComment(commentId),
    onSuccess: () => {
      toast.success("Comment deleted.");
      qc.invalidateQueries({ queryKey: ["quiz-comments", quiz?.course_id, quizId] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not delete comment"),
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
        <p className="text-sm text-brand/70">{(error as Error)?.message}</p>
      </AppShell>
    );
  }

  const attemptsLeft = quiz.remaining_attempts ?? null;
  const resuming = Boolean(quiz.in_progress_attempt_id);
  const blocked = !resuming && (attemptsLeft === 0 || quiz.availability !== "open");
  const fmt = (iso?: string | null) => (iso ? new Date(iso).toLocaleString() : null);

  return (
    <AppShell eyebrow={quiz.course_title ?? "Assessment"} title={quiz.title}>
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <span
          className={`border px-2 py-0.5 uppercase tracking-wide ${quiz.mode === "practice" ? "border-accent/40 text-accent-ink" : "border-brand/20 text-brand/70"}`}
        >
          {quiz.mode === "practice" ? "Practice — hints and answer checks available" : "Exam"}
        </span>
        {quiz.status === "pending_grading" && (
          <span className="border border-brand/20 px-2 py-0.5 text-brand/70">
            Last attempt awaiting marking
          </span>
        )}
        {quiz.best_score != null && (
          <span className="border border-brand/20 px-2 py-0.5 text-brand/70 tabular-nums">
            Best score {quiz.best_score}%
          </span>
        )}
      </div>
      {quiz.description && (
        <div className="mt-4 max-w-2xl text-brand/75">
          <RichContent text={quiz.description} />
        </div>
      )}

      <dl className="mt-8 grid max-w-4xl grid-cols-2 gap-4 lg:grid-cols-4">
        <Meta icon={FileQuestion} label="Questions" value={String(quiz.total_questions)} />
        <Meta icon={Award} label="Total marks" value={String(quiz.total_points)} />
        <Meta
          icon={Clock}
          label="Time limit"
          value={quiz.time_limit_minutes ? `${quiz.time_limit_minutes} min` : "Untimed"}
        />
        <Meta
          icon={RotateCw}
          label="Attempts left"
          value={
            quiz.attempts_allowed == null
              ? "Unlimited"
              : `${attemptsLeft ?? 0} of ${quiz.attempts_allowed}`
          }
        />
      </dl>

      <dl className="mt-6 grid max-w-4xl gap-x-8 gap-y-3 text-sm sm:grid-cols-2">
        <Row label="Passing score" value={`${quiz.passing_score}%`} />
        <Row label="Opens" value={fmt(quiz.available_from) ?? "Now"} />
        <Row label="Closes" value={fmt(quiz.available_until) ?? "No closing date"} />
        <Row
          label="Calculator"
          value={
            quiz.calculator_policy === "none"
              ? "Not permitted"
              : `${quiz.calculator_policy[0].toUpperCase()}${quiz.calculator_policy.slice(1)} calculator permitted`
          }
        />
        <Row
          label="Answers after submission"
          value={
            quiz.show_answers_after_submission
              ? "Correct answers are shown"
              : "Only your own answers are shown"
          }
        />
      </dl>

      {quiz.instructions && (
        <section className="mt-10 max-w-2xl">
          <h2 className="font-serif text-xl mb-3">Instructions</h2>
          <div className="text-sm text-brand/75 leading-relaxed">
            <RichContent text={quiz.instructions} />
          </div>
        </section>
      )}
      {quiz.reference_material && (
        <section className="mt-6 max-w-2xl">
          <h2 className="font-serif text-xl mb-3">Permitted resources</h2>
          <div className="text-sm text-brand/75 leading-relaxed">
            <RichContent text={quiz.reference_material} />
          </div>
        </section>
      )}
      <ul className="mt-6 max-w-2xl list-disc space-y-1 pl-5 text-xs text-brand/70">
        <li>
          Your answers are saved automatically as you work, including if you lose your connection.
        </li>
        {quiz.time_limit_minutes ? (
          <li>
            The timer keeps running if you leave; when it ends, your saved answers are submitted.
          </li>
        ) : null}
        <li>You can flag questions and review everything before you submit.</li>
      </ul>

      <div className="mt-10 flex flex-wrap items-center gap-3">
        <button
          disabled={startMutation.isPending || blocked}
          onClick={() => startMutation.mutate()}
          className="bg-brand px-7 py-4 text-sm font-medium text-white transition-colors hover:bg-brand/90 disabled:opacity-50"
        >
          {startMutation.isPending
            ? "Preparing your attempt…"
            : resuming
              ? "Resume attempt"
              : "Start assessment"}
        </button>
        {quiz.latest_attempt_id && quiz.status !== "in_progress" && (
          <Link
            to="/student/assessments/$quizId/result/$attemptId"
            params={{ quizId, attemptId: quiz.latest_attempt_id }}
            className="border border-brand/15 px-6 py-4 text-sm hover:border-brand/40"
          >
            View last result
          </Link>
        )}
        <Link
          to="/student/assessments"
          className="text-xs text-brand/70 underline underline-offset-4 hover:text-brand"
        >
          Back to assessments
        </Link>
        {!resuming && quiz.availability === "not_open" && (
          <p className="text-xs text-brand/70">Opens {fmt(quiz.available_from)}.</p>
        )}
        {!resuming && quiz.availability === "closed" && (
          <p className="text-xs text-destructive">This assessment has closed.</p>
        )}
        {!resuming && attemptsLeft === 0 && (
          <p className="text-xs text-destructive">No attempts remaining.</p>
        )}
      </div>

      <QuizDiscussion
        comments={(comments.data?.comments ?? []).filter((comment) => comment.quiz_id === quizId)}
        userId={user?.id}
        userRole={user?.role}
        saving={createComment.isPending || updateComment.isPending || deleteComment.isPending}
        onPost={(content) => createComment.mutate({ content })}
        onReply={(parentId, content) =>
          createComment.mutate({ content, parent_comment_id: parentId })
        }
        onUpdate={(commentId, content) => updateComment.mutate({ commentId, content })}
        onPin={(commentId, isPinned) => updateComment.mutate({ commentId, is_pinned: isPinned })}
        onDelete={(commentId) => deleteComment.mutate(commentId)}
      />
    </AppShell>
  );
}

function Meta({ icon: Icon, label, value }: { icon: typeof Clock; label: string; value: string }) {
  return (
    <div className="border border-brand/10 bg-white/40 p-5">
      <Icon className="h-4 w-4 text-brand/70 mb-3" aria-hidden />
      <dt className="eyebrow text-brand/70">{label}</dt>
      <dd className="mt-2 font-serif text-2xl tabular-nums">{value}</dd>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4 border-b border-brand/10 pb-2">
      <dt className="text-brand/70">{label}</dt>
      <dd className="text-right">{value}</dd>
    </div>
  );
}

function QuizDiscussion({
  comments,
  userId,
  userRole,
  saving,
  onPost,
  onReply,
  onUpdate,
  onPin,
  onDelete,
}: {
  comments: CourseComment[];
  userId?: string;
  userRole?: string;
  saving: boolean;
  onPost: (content: string) => void;
  onReply: (parentId: string, content: string) => void;
  onUpdate: (commentId: string, content: string) => void;
  onPin: (commentId: string, isPinned: boolean) => void;
  onDelete: (commentId: string) => void;
}) {
  const [content, setContent] = useState("");
  const [replyTo, setReplyTo] = useState<string | null>(null);
  const [reply, setReply] = useState("");
  const [editing, setEditing] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const roots = comments.filter((comment) => !comment.parent_comment_id);
  const replies = comments.reduce<Record<string, CourseComment[]>>((acc, comment) => {
    if (!comment.parent_comment_id) return acc;
    acc[comment.parent_comment_id] = [...(acc[comment.parent_comment_id] ?? []), comment];
    return acc;
  }, {});

  return (
    <section className="mt-12 max-w-3xl border-t border-brand/10 pt-8">
      <h2 className="font-serif text-xl">Quiz discussion</h2>
      <form
        className="mt-4"
        onSubmit={(event) => {
          event.preventDefault();
          if (!content.trim()) return;
          onPost(content.trim());
          setContent("");
        }}
      >
        <textarea
          value={content}
          onChange={(event) => setContent(event.target.value)}
          rows={3}
          placeholder="Ask a question about this quiz."
          className="w-full border border-brand/15 bg-white px-3 py-2 text-sm"
        />
        <button
          disabled={saving || !content.trim()}
          className="mt-3 bg-brand px-4 py-2 text-sm text-white disabled:opacity-50"
        >
          Post comment
        </button>
      </form>
      <ul className="mt-5 space-y-3">
        {roots.length === 0 && <li className="text-sm text-brand/70">No quiz comments yet.</li>}
        {roots.map((comment) => {
          const canEdit = comment.user_id === userId;
          const canModerate = userRole === "teacher" || userRole === "admin";
          const canDelete = canEdit || canModerate;
          return (
            <li key={comment.id} className="border border-brand/10 bg-white/50 p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1">
                  {comment.is_pinned && (
                    <p className="mb-1 text-[10px] uppercase tracking-[0.18em] text-accent-ink">
                      Pinned
                    </p>
                  )}
                  {editing === comment.id ? (
                    <textarea
                      value={draft}
                      onChange={(event) => setDraft(event.target.value)}
                      rows={3}
                      className="w-full border border-brand/15 bg-white px-3 py-2 text-sm"
                    />
                  ) : (
                    <p className="text-sm text-brand/75 whitespace-pre-wrap">{comment.content}</p>
                  )}
                  <p className="mt-2 text-[11px] text-brand/70">
                    {new Date(comment.created_at).toLocaleString()}
                  </p>
                  <div className="mt-3 flex flex-wrap gap-3 text-xs">
                    <button type="button" onClick={() => setReplyTo(comment.id)}>
                      Reply
                    </button>
                    {canEdit &&
                      (editing === comment.id ? (
                        <>
                          <button
                            type="button"
                            disabled={saving || !draft.trim()}
                            onClick={() => {
                              onUpdate(comment.id, draft.trim());
                              setEditing(null);
                            }}
                          >
                            Save
                          </button>
                          <button type="button" onClick={() => setEditing(null)}>
                            Cancel
                          </button>
                        </>
                      ) : (
                        <button
                          type="button"
                          onClick={() => {
                            setEditing(comment.id);
                            setDraft(comment.content);
                          }}
                        >
                          Edit
                        </button>
                      ))}
                    {canModerate && (
                      <button
                        type="button"
                        disabled={saving}
                        onClick={() => onPin(comment.id, !comment.is_pinned)}
                      >
                        {comment.is_pinned ? "Unpin" : "Pin"}
                      </button>
                    )}
                    {canDelete && (
                      <button
                        type="button"
                        disabled={saving}
                        onClick={() => onDelete(comment.id)}
                        className="text-brand/70 hover:text-destructive"
                      >
                        Delete
                      </button>
                    )}
                  </div>
                  {replyTo === comment.id && (
                    <form
                      className="mt-3"
                      onSubmit={(event) => {
                        event.preventDefault();
                        if (!reply.trim()) return;
                        onReply(comment.id, reply.trim());
                        setReply("");
                        setReplyTo(null);
                      }}
                    >
                      <textarea
                        value={reply}
                        onChange={(event) => setReply(event.target.value)}
                        rows={2}
                        className="w-full border border-brand/15 bg-white px-3 py-2 text-sm"
                      />
                      <button
                        disabled={saving || !reply.trim()}
                        className="mt-2 bg-brand px-3 py-1.5 text-xs text-white disabled:opacity-50"
                      >
                        Post reply
                      </button>
                    </form>
                  )}
                  {(replies[comment.id] ?? []).length > 0 && (
                    <ul className="mt-4 space-y-2 border-l border-brand/10 pl-4">
                      {(replies[comment.id] ?? []).map((child) => (
                        <li key={child.id} className="bg-white/60 p-3">
                          <p className="text-sm text-brand/70 whitespace-pre-wrap">
                            {child.content}
                          </p>
                          <p className="mt-2 text-[11px] text-brand/70">
                            {new Date(child.created_at).toLocaleString()}
                          </p>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

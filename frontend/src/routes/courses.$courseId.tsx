import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Clock, Lock, PlayCircle, Star, Users } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { PreviewPlayerDialog } from "@/components/course/preview-player-dialog";
import { BrandLogo } from "@/components/layout/brand-logo";
import { useAuth } from "@/context/auth-context";
import {
  createCourseComment,
  deleteCourseComment,
  deleteMyCourseReview,
  getCourse,
  updateCourseComment,
  upsertCourseReview,
  type CourseComment,
  type Module,
} from "@/lib/api/courses";
import { enroll } from "@/lib/api/student";

export const Route = createFileRoute("/courses/$courseId")({
  component: CourseDetailPage,
  notFoundComponent: () => (
    <div className="min-h-screen grid place-items-center bg-surface text-brand font-sans px-6">
      <div className="max-w-md text-center">
        <p className="eyebrow text-accent">404</p>
        <h1 className="mt-4 font-serif text-4xl">Course not found</h1>
        <Link to="/courses" className="mt-6 inline-block bg-brand text-white px-6 py-3">
          Back to catalog
        </Link>
      </div>
    </div>
  ),
  errorComponent: ({ error }) => (
    <div className="min-h-screen grid place-items-center bg-surface text-brand font-sans px-6">
      <div className="max-w-md text-center">
        <p className="eyebrow text-destructive">Error</p>
        <h1 className="mt-4 font-serif text-3xl">Couldn't load this course</h1>
        <p className="mt-2 text-sm text-brand/55">{error.message}</p>
      </div>
    </div>
  ),
});

function CourseDetailPage() {
  const { courseId } = Route.useParams();
  const { isAuthenticated, isHydrated, user } = useAuth();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [previewLesson, setPreviewLesson] = useState<{
    id: string;
    title: string;
    duration?: number;
  } | null>(null);

  const {
    data: course,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ["course", courseId],
    queryFn: () => getCourse(courseId),
  });

  const enrollMutation = useMutation({
    mutationFn: () => enroll(course!.id),
    onSuccess: () => {
      toast.success("Enrolled — let's begin.");
      qc.invalidateQueries({ queryKey: ["course", courseId] });
      qc.invalidateQueries({ queryKey: ["enrollments"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
      navigate({ to: "/student/player/$courseId", params: { courseId: course!.id } });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not enroll"),
  });

  const reviewMutation = useMutation({
    mutationFn: (input: { rating: number; comment: string }) =>
      upsertCourseReview(course!.id, input),
    onSuccess: () => {
      toast.success("Review saved.");
      qc.invalidateQueries({ queryKey: ["course", courseId] });
      qc.invalidateQueries({ queryKey: ["course-reviews", courseId] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not save review"),
  });

  const deleteReviewMutation = useMutation({
    mutationFn: () => deleteMyCourseReview(course!.id),
    onSuccess: () => {
      toast.success("Review deleted.");
      qc.invalidateQueries({ queryKey: ["course", courseId] });
      qc.invalidateQueries({ queryKey: ["course-reviews", courseId] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not delete review"),
  });

  const commentMutation = useMutation({
    mutationFn: (input: {
      content: string;
      module_id?: string;
      lesson_id?: string;
      quiz_id?: string;
      parent_comment_id?: string;
    }) => createCourseComment(course!.id, input),
    onSuccess: () => {
      toast.success("Comment posted.");
      qc.invalidateQueries({ queryKey: ["course", courseId] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not post comment"),
  });

  const deleteCommentMutation = useMutation({
    mutationFn: (commentId: string) => deleteCourseComment(commentId),
    onSuccess: () => {
      toast.success("Comment deleted.");
      qc.invalidateQueries({ queryKey: ["course", courseId] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not delete comment"),
  });

  const updateCommentMutation = useMutation({
    mutationFn: (input: { commentId: string; content?: string; is_pinned?: boolean }) =>
      updateCourseComment(input.commentId, {
        content: input.content,
        is_pinned: input.is_pinned,
      }),
    onSuccess: () => {
      toast.success("Comment updated.");
      qc.invalidateQueries({ queryKey: ["course", courseId] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not update comment"),
  });

  if (isLoading) {
    return (
      <div className="min-h-screen bg-surface text-brand font-sans">
        <div className="px-6 md:px-12 lg:px-20 py-16 animate-pulse">
          <div className="h-3 w-24 bg-brand/10" />
          <div className="mt-6 h-12 w-2/3 bg-brand/10" />
          <div className="mt-4 h-4 w-1/2 bg-brand/10" />
        </div>
      </div>
    );
  }

  if (isError || !course) {
    return (
      <div className="min-h-screen grid place-items-center bg-surface text-brand font-sans">
        <p className="text-sm text-brand/55">{(error as Error)?.message ?? "Not found"}</p>
      </div>
    );
  }

  const totalLessons = course.modules?.reduce((n, m) => n + m.lessons.length, 0) ?? 0;
  const canEnrollAsStudent = isHydrated && isAuthenticated && user?.role === "student";

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      <header className="px-6 md:px-12 lg:px-20 py-6 flex items-center justify-between border-b border-brand/10">
        <BrandLogo imageClassName="max-h-14 max-w-[220px]" />
        <nav className="flex items-center gap-6 text-sm">
          <Link to="/courses" className="text-brand/60 hover:text-brand transition-colors">
            Catalog
          </Link>
          <Link to="/bookshop" className="text-brand/60 hover:text-brand transition-colors">
            Bookshop
          </Link>
          {!isAuthenticated && (
            <Link to="/login" className="text-brand/60 hover:text-brand transition-colors">
              Sign in
            </Link>
          )}
        </nav>
      </header>

      <div className="px-6 md:px-12 lg:px-20 py-12 lg:py-16 grid lg:grid-cols-[1fr_380px] gap-12">
        <article>
          {course.category?.name && (
            <p className="eyebrow text-accent mb-4">{course.category.name}</p>
          )}
          <h1 className="font-serif text-4xl lg:text-6xl leading-[1.05] text-balance">
            {course.title}
          </h1>
          {course.subtitle && (
            <p className="mt-6 text-xl text-brand/65 leading-relaxed max-w-2xl">
              {course.subtitle}
            </p>
          )}

          <div className="mt-8 flex flex-wrap items-center gap-6 text-sm text-brand/60">
            {course.teacher?.full_name && (
              <span>
                Taught by <span className="text-brand">{course.teacher.full_name}</span>
              </span>
            )}
            {typeof course.rating === "number" && (
              <span className="inline-flex items-center gap-1.5">
                <Star className="h-3.5 w-3.5 fill-accent text-accent" />
                {course.rating.toFixed(1)} rating
              </span>
            )}
            {typeof course.enrollment_count === "number" && (
              <span className="inline-flex items-center gap-1.5">
                <Users className="h-3.5 w-3.5" />
                {course.enrollment_count.toLocaleString()} enrolled
              </span>
            )}
            {typeof course.duration_minutes === "number" && (
              <span className="inline-flex items-center gap-1.5">
                <Clock className="h-3.5 w-3.5" />
                {Math.round(course.duration_minutes / 60)} hours
              </span>
            )}
          </div>

          {course.description && (
            <section className="mt-12">
              <h2 className="font-serif text-2xl mb-4">About this course</h2>
              <p className="text-brand/70 leading-relaxed whitespace-pre-line">
                {course.description}
              </p>
            </section>
          )}

          {course.outcomes && course.outcomes.length > 0 && (
            <section className="mt-12">
              <h2 className="font-serif text-2xl mb-6">What you'll learn</h2>
              <ul className="grid sm:grid-cols-2 gap-3">
                {course.outcomes.map((o, i) => (
                  <li key={i} className="flex gap-3 text-sm text-brand/75">
                    <span className="text-accent mt-1">✦</span>
                    <span>{o}</span>
                  </li>
                ))}
              </ul>
            </section>
          )}

          {course.modules && course.modules.length > 0 && (
            <section className="mt-12">
              <h2 className="font-serif text-2xl mb-2">Syllabus</h2>
              <p className="text-sm text-brand/55 mb-6">
                {course.modules.length} module{course.modules.length === 1 ? "" : "s"} ·{" "}
                {totalLessons} lesson{totalLessons === 1 ? "" : "s"}
              </p>
              <div className="space-y-4">
                {course.modules.map((m, idx) => (
                  <details
                    key={m.id}
                    open={idx === 0}
                    className="border border-brand/10 bg-white/40 group"
                  >
                    <summary className="px-5 py-4 cursor-pointer flex items-center justify-between hover:bg-brand/[0.02]">
                      <div>
                        <p className="eyebrow text-brand/40">Module {idx + 1}</p>
                        <p className="font-serif text-lg mt-1">{m.title}</p>
                      </div>
                      <span className="text-xs text-brand/45">
                        {m.lessons.length} lesson{m.lessons.length === 1 ? "" : "s"}
                      </span>
                    </summary>
                    <ul className="border-t border-brand/10 divide-y divide-brand/5">
                      {m.lessons.map((l) => {
                        const canPreview = l.is_preview && !course.is_enrolled;
                        return (
                          <li
                            key={l.id}
                            className="px-5 py-3 flex items-center justify-between text-sm gap-3"
                          >
                            <span className="flex items-center gap-3 text-brand/75 min-w-0">
                              {l.is_preview ? (
                                <PlayCircle className="h-4 w-4 text-accent shrink-0" />
                              ) : course.is_enrolled ? (
                                <PlayCircle className="h-4 w-4 text-brand/30 shrink-0" />
                              ) : (
                                <Lock className="h-3.5 w-3.5 text-brand/30 shrink-0" />
                              )}
                              <span className="truncate">{l.title}</span>
                              {l.is_preview && (
                                <span className="text-[10px] uppercase tracking-wider text-accent shrink-0">
                                  Preview
                                </span>
                              )}
                            </span>
                            <span className="flex items-center gap-3 shrink-0">
                              {typeof l.duration_minutes === "number" && (
                                <span className="text-xs text-brand/40">
                                  {l.duration_minutes} min
                                </span>
                              )}
                              {canPreview && (
                                <button
                                  type="button"
                                  onClick={() =>
                                    setPreviewLesson({
                                      id: l.id,
                                      title: l.title,
                                      duration: l.duration_minutes,
                                    })
                                  }
                                  className="text-xs font-medium text-accent hover:underline underline-offset-4"
                                >
                                  Watch preview
                                </button>
                              )}
                            </span>
                          </li>
                        );
                      })}
                    </ul>
                  </details>
                ))}
              </div>
            </section>
          )}

          <CourseReviewsSection
            courseId={course.id}
            isEnrolled={Boolean(course.is_enrolled)}
            userId={user?.id}
            rating={course.rating}
            onSave={(rating, comment) => reviewMutation.mutate({ rating, comment })}
            onDelete={() => deleteReviewMutation.mutate()}
            saving={reviewMutation.isPending || deleteReviewMutation.isPending}
          />

          <CourseCommentsSection
            comments={course.comments ?? []}
            modules={course.modules ?? []}
            canComment={Boolean(
              course.is_enrolled || user?.role === "teacher" || user?.role === "admin",
            )}
            userId={user?.id}
            userRole={user?.role}
            onPost={(input) => commentMutation.mutate(input)}
            onUpdate={(commentId, content) => updateCommentMutation.mutate({ commentId, content })}
            onPin={(commentId, isPinned) =>
              updateCommentMutation.mutate({ commentId, is_pinned: isPinned })
            }
            onDelete={(commentId) => deleteCommentMutation.mutate(commentId)}
            saving={
              commentMutation.isPending ||
              updateCommentMutation.isPending ||
              deleteCommentMutation.isPending
            }
          />
        </article>

        {/* Enrollment card */}
        <aside className="lg:sticky lg:top-8 lg:self-start">
          <div className="border border-brand/10 bg-white/60 overflow-hidden">
            {course.cover_url ? (
              <img
                src={course.cover_url}
                alt={course.title}
                className="aspect-[16/10] w-full object-cover"
              />
            ) : (
              <div className="aspect-[16/10] bg-brand/5 grid place-items-center font-serif italic text-5xl text-brand/20">
                {course.title.slice(0, 1)}
              </div>
            )}
            <div className="p-6">
              <p className="font-serif text-3xl">
                {course.price === 0
                  ? "Free"
                  : `${course.currency ?? "BDT"} ${course.price?.toLocaleString() ?? "—"}`}
              </p>
              {course.is_enrolled ? (
                <Link
                  to="/student/player/$courseId"
                  params={{ courseId: course.id }}
                  className="mt-6 block text-center bg-brand text-white py-4 text-sm font-medium hover:bg-brand/90 transition-colors"
                >
                  Continue learning
                </Link>
              ) : canEnrollAsStudent ? (
                course.price && course.price > 0 ? (
                  <Link
                    to="/student/courses/$courseId/request-access"
                    params={{ courseId: course.id }}
                    className="mt-6 block text-center bg-brand text-white py-4 text-sm font-medium hover:bg-brand/90 transition-colors"
                  >
                    Request approval
                  </Link>
                ) : (
                  <button
                    onClick={() => enrollMutation.mutate()}
                    disabled={enrollMutation.isPending}
                    className="mt-6 w-full bg-brand text-white py-4 text-sm font-medium hover:bg-brand/90 transition-colors disabled:opacity-60"
                  >
                    {enrollMutation.isPending ? "Enrolling…" : "Enroll for free"}
                  </button>
                )
              ) : (
                <Link
                  to="/login"
                  search={{ return: `/courses/${courseId}` } as never}
                  className="mt-6 block text-center bg-brand text-white py-4 text-sm font-medium hover:bg-brand/90 transition-colors"
                >
                  {isHydrated && isAuthenticated
                    ? "Sign in as student to enroll"
                    : "Sign in to enroll"}
                </Link>
              )}
              <ul className="mt-6 space-y-2 text-xs text-brand/60">
                {typeof course.duration_minutes === "number" && (
                  <li>· {Math.round(course.duration_minutes / 60)} hours of content</li>
                )}
                {totalLessons > 0 && <li>· {totalLessons} lessons</li>}
                {course.level && <li>· {course.level} level</li>}
                <li>· Certificate on completion</li>
              </ul>
            </div>
          </div>
        </aside>
      </div>

      {previewLesson && (
        <PreviewPlayerDialog
          courseId={course.id}
          lessonId={previewLesson.id}
          lessonTitle={previewLesson.title}
          durationMinutes={previewLesson.duration}
          open={!!previewLesson}
          onOpenChange={(o) => !o && setPreviewLesson(null)}
        />
      )}
    </div>
  );
}

function CourseReviewsSection({
  isEnrolled,
  userId,
  rating,
  onSave,
  onDelete,
  saving,
}: {
  courseId: string;
  isEnrolled: boolean;
  userId?: string;
  rating?: number;
  onSave: (rating: number, comment: string) => void;
  onDelete: () => void;
  saving: boolean;
}) {
  const [draftRating, setDraftRating] = useState(5);
  const [comment, setComment] = useState("");
  return (
    <section className="mt-12 border-t border-brand/10 pt-10">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="font-serif text-2xl">Ratings and reviews</h2>
          <p className="mt-1 text-sm text-brand/55">
            Average rating {typeof rating === "number" ? rating.toFixed(1) : "not available yet"}.
          </p>
        </div>
        {isEnrolled && userId && (
          <button
            type="button"
            onClick={onDelete}
            disabled={saving}
            className="border border-brand/15 px-3 py-2 text-xs text-brand/60 hover:text-destructive disabled:opacity-50"
          >
            Delete my review
          </button>
        )}
      </div>
      {isEnrolled ? (
        <form
          className="mt-5 border border-brand/10 bg-white/50 p-4"
          onSubmit={(event) => {
            event.preventDefault();
            onSave(draftRating, comment);
          }}
        >
          <label className="text-xs font-semibold uppercase tracking-[0.18em] text-brand/45">
            Rating
          </label>
          <select
            value={draftRating}
            onChange={(event) => setDraftRating(Number(event.target.value))}
            className="mt-2 block border border-brand/15 bg-white px-3 py-2 text-sm"
          >
            {[5, 4, 3, 2, 1].map((value) => (
              <option key={value} value={value}>
                {value} star{value === 1 ? "" : "s"}
              </option>
            ))}
          </select>
          <textarea
            value={comment}
            onChange={(event) => setComment(event.target.value)}
            rows={4}
            placeholder="Share what helped you learn."
            className="mt-3 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
          />
          <button
            type="submit"
            disabled={saving}
            className="mt-3 bg-brand px-4 py-2 text-sm text-white disabled:opacity-50"
          >
            {saving ? "Saving..." : "Save review"}
          </button>
        </form>
      ) : (
        <p className="mt-4 text-sm text-brand/55">
          Enroll or get approved access to add a course review.
        </p>
      )}
    </section>
  );
}

function CourseCommentsSection({
  comments,
  modules,
  canComment,
  userId,
  userRole,
  onPost,
  onUpdate,
  onPin,
  onDelete,
  saving,
}: {
  comments: CourseComment[];
  modules: Module[];
  canComment: boolean;
  userId?: string;
  userRole?: string;
  onPost: (input: {
    content: string;
    module_id?: string;
    lesson_id?: string;
    quiz_id?: string;
    parent_comment_id?: string;
  }) => void;
  onUpdate: (commentId: string, content: string) => void;
  onPin: (commentId: string, isPinned: boolean) => void;
  onDelete: (commentId: string) => void;
  saving: boolean;
}) {
  const [content, setContent] = useState("");
  const [target, setTarget] = useState("course");
  const sorted = [...comments].sort((a, b) => {
    if (a.is_pinned !== b.is_pinned) return a.is_pinned ? -1 : 1;
    return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
  });
  const roots = sorted.filter((comment) => !comment.parent_comment_id);
  const repliesByParent = sorted.reduce<Record<string, CourseComment[]>>((acc, comment) => {
    if (!comment.parent_comment_id) return acc;
    acc[comment.parent_comment_id] = [...(acc[comment.parent_comment_id] ?? []), comment];
    return acc;
  }, {});
  const targetLabel = (comment: CourseComment) => {
    if (comment.module_id) {
      return `Module: ${modules.find((module) => module.id === comment.module_id)?.title ?? "Module"}`;
    }
    if (comment.lesson_id) return "Lesson discussion";
    if (comment.quiz_id) return "Quiz discussion";
    return "Course discussion";
  };
  return (
    <section className="mt-12 border-t border-brand/10 pt-10">
      <h2 className="font-serif text-2xl">Discussion</h2>
      {canComment ? (
        <form
          className="mt-5"
          onSubmit={(event) => {
            event.preventDefault();
            if (!content.trim()) return;
            onPost({
              content: content.trim(),
              module_id: target.startsWith("module:") ? target.slice("module:".length) : undefined,
            });
            setContent("");
          }}
        >
          {modules.length > 0 && (
            <select
              value={target}
              onChange={(event) => setTarget(event.target.value)}
              className="mb-3 w-full border border-brand/15 bg-white px-3 py-2 text-sm sm:w-72"
            >
              <option value="course">Course discussion</option>
              {modules.map((module) => (
                <option key={module.id} value={`module:${module.id}`}>
                  Module: {module.title}
                </option>
              ))}
            </select>
          )}
          <textarea
            value={content}
            onChange={(event) => setContent(event.target.value)}
            rows={3}
            placeholder="Ask a question or share a note."
            className="w-full border border-brand/15 bg-white px-3 py-2 text-sm"
          />
          <button
            type="submit"
            disabled={saving || !content.trim()}
            className="mt-3 bg-brand px-4 py-2 text-sm text-white disabled:opacity-50"
          >
            {saving ? "Posting..." : "Post comment"}
          </button>
        </form>
      ) : (
        <p className="mt-4 text-sm text-brand/55">
          Discussion unlocks after enrollment or approved access.
        </p>
      )}
      <ul className="mt-6 space-y-3">
        {comments.length === 0 && <li className="text-sm text-brand/45">No comments yet.</li>}
        {roots.map((comment) => (
          <ThreadedComment
            key={comment.id}
            comment={comment}
            replies={repliesByParent[comment.id] ?? []}
            canComment={canComment}
            canPin={userRole === "teacher" || userRole === "admin"}
            canEdit={comment.user_id === userId}
            canDelete={comment.user_id === userId || userRole === "teacher" || userRole === "admin"}
            saving={saving}
            targetLabel={targetLabel(comment)}
            onReply={(reply) => onPost({ content: reply, parent_comment_id: comment.id })}
            onUpdate={(updated) => onUpdate(comment.id, updated)}
            onPin={(isPinned) => onPin(comment.id, isPinned)}
            onDelete={() => onDelete(comment.id)}
          />
        ))}
      </ul>
    </section>
  );
}

function ThreadedComment({
  comment,
  replies,
  canComment,
  canPin,
  canEdit,
  canDelete,
  saving,
  targetLabel,
  onReply,
  onUpdate,
  onPin,
  onDelete,
}: {
  comment: CourseComment;
  replies: CourseComment[];
  canComment: boolean;
  canPin: boolean;
  canEdit: boolean;
  canDelete: boolean;
  saving: boolean;
  targetLabel: string;
  onReply: (content: string) => void;
  onUpdate: (content: string) => void;
  onPin: (isPinned: boolean) => void;
  onDelete: () => void;
}) {
  const [replying, setReplying] = useState(false);
  const [editing, setEditing] = useState(false);
  const [reply, setReply] = useState("");
  const [draft, setDraft] = useState(comment.content);

  return (
    <li className="border border-brand/10 bg-white/50 p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="mb-1 flex flex-wrap items-center gap-2">
            {comment.is_pinned && (
              <span className="text-[10px] uppercase tracking-[0.18em] text-accent">Pinned</span>
            )}
            <span className="text-[10px] uppercase tracking-[0.18em] text-brand/35">
              {targetLabel}
            </span>
          </div>
          {editing ? (
            <textarea
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              rows={3}
              className="w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          ) : (
            <p className="text-sm text-brand/75 whitespace-pre-wrap">{comment.content}</p>
          )}
          <p className="mt-2 text-[11px] text-brand/40">
            {new Date(comment.created_at).toLocaleString()}
          </p>
          <div className="mt-3 flex flex-wrap gap-3 text-xs">
            {canComment && (
              <button type="button" onClick={() => setReplying((value) => !value)}>
                Reply
              </button>
            )}
            {canEdit &&
              (editing ? (
                <>
                  <button
                    type="button"
                    disabled={saving || !draft.trim()}
                    onClick={() => {
                      onUpdate(draft.trim());
                      setEditing(false);
                    }}
                  >
                    Save
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setDraft(comment.content);
                      setEditing(false);
                    }}
                  >
                    Cancel
                  </button>
                </>
              ) : (
                <button type="button" onClick={() => setEditing(true)}>
                  Edit
                </button>
              ))}
            {canPin && (
              <button type="button" disabled={saving} onClick={() => onPin(!comment.is_pinned)}>
                {comment.is_pinned ? "Unpin" : "Pin"}
              </button>
            )}
            {canDelete && (
              <button
                type="button"
                onClick={onDelete}
                disabled={saving}
                className="text-brand/45 hover:text-destructive disabled:opacity-50"
              >
                Delete
              </button>
            )}
          </div>
        </div>
      </div>
      {replying && (
        <form
          className="mt-4"
          onSubmit={(event) => {
            event.preventDefault();
            if (!reply.trim()) return;
            onReply(reply.trim());
            setReply("");
            setReplying(false);
          }}
        >
          <textarea
            value={reply}
            onChange={(event) => setReply(event.target.value)}
            rows={2}
            className="w-full border border-brand/15 bg-white px-3 py-2 text-sm"
          />
          <button
            type="submit"
            disabled={saving || !reply.trim()}
            className="mt-2 bg-brand px-3 py-1.5 text-xs text-white disabled:opacity-50"
          >
            Post reply
          </button>
        </form>
      )}
      {replies.length > 0 && (
        <ul className="mt-4 space-y-2 border-l border-brand/10 pl-4">
          {replies.map((replyComment) => (
            <li key={replyComment.id} className="bg-white/60 p-3">
              <p className="text-sm text-brand/70 whitespace-pre-wrap">{replyComment.content}</p>
              <p className="mt-2 text-[11px] text-brand/40">
                {new Date(replyComment.created_at).toLocaleString()}
              </p>
            </li>
          ))}
        </ul>
      )}
    </li>
  );
}

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
  CheckCircle2,
  ChevronLeft,
  Circle,
  Download,
  FileText,
  Lock,
  PlayCircle,
  Trash2,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import { ApiError } from "@/lib/api/client";
import {
  createCourseComment,
  deleteCourseComment,
  getCourse,
  updateCourseComment,
  type CourseComment,
} from "@/lib/api/courses";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { useAuth } from "@/context/auth-context";
import { completeLesson, getCourseProgress, getLesson } from "@/lib/api/student";
import { useVideoReadyPolling } from "@/hooks/use-video-ready-polling";
import { downloadLesson, isLessonDownloaded, removeOfflineLesson } from "@/lib/offline-lessons";

export const Route = createFileRoute("/_authenticated/student/player/$courseId")({
  component: PlayerPage,
});

function PlayerPage() {
  const { courseId } = Route.useParams();
  const { user } = useAuth();
  const qc = useQueryClient();

  const { data: course } = useQuery({
    queryKey: ["course", courseId],
    queryFn: () => getCourse(courseId),
  });

  const { data: progress } = useQuery({
    queryKey: ["progress", courseId],
    queryFn: () => getCourseProgress(courseId),
  });

  const modules = progress?.modules ?? course?.modules ?? [];
  const allLessons = useMemo(() => modules.flatMap((m) => m.lessons), [modules]);
  const completedSet = useMemo(
    () => new Set(progress?.completed_lessons ?? []),
    [progress?.completed_lessons],
  );

  const [activeLessonId, setActiveLessonId] = useState<string | null>(null);

  useEffect(() => {
    if (activeLessonId || allLessons.length === 0) return;
    const initial =
      progress?.current_lesson_id ??
      allLessons.find((l) => !completedSet.has(l.id))?.id ??
      allLessons[0]?.id ??
      null;
    setActiveLessonId(initial);
  }, [activeLessonId, allLessons, completedSet, progress?.current_lesson_id]);

  const {
    data: lesson,
    isLoading: lessonLoading,
    isError: lessonError,
    error: lessonErrorObj,
    refetch: refetchLesson,
  } = useQuery({
    queryKey: ["lesson", courseId, activeLessonId],
    queryFn: () => getLesson(courseId, activeLessonId!),
    enabled: Boolean(activeLessonId),
  });

  const {
    retrying: videoPolling,
    secondsUntilNextRetry,
    cancel: cancelVideoPolling,
  } = useVideoReadyPolling({
    error: lessonErrorObj,
    enabled: Boolean(activeLessonId),
    refetch: () => refetchLesson(),
  });

  const completeMutation = useMutation({
    mutationFn: (lessonId: string) => completeLesson(courseId, lessonId),
    onSuccess: (_data, lessonId) => {
      toast.success("Lesson complete.");
      qc.invalidateQueries({ queryKey: ["progress", courseId] });
      qc.invalidateQueries({ queryKey: ["enrollments"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
      // advance to next lesson
      const idx = allLessons.findIndex((l) => l.id === lessonId);
      const next = allLessons[idx + 1];
      if (next) setActiveLessonId(next.id);
    },
    onError: (e: Error) => toast.error(e.message ?? "Couldn't mark complete"),
  });

  const activeCompleted = activeLessonId ? completedSet.has(activeLessonId) : false;
  const activeLessonSummary = useMemo(
    () => allLessons.find((item) => item.id === activeLessonId) ?? null,
    [activeLessonId, allLessons],
  );
  const courseTitle = course?.title ?? "Loading…";
  const pct = progress?.progress_percent ?? 0;
  const activeNotes = (course?.notes ?? []).filter(
    (note) => !note.lesson_id || note.lesson_id === activeLessonId,
  );
  const activeComments = (course?.comments ?? []).filter(
    (comment) => !comment.lesson_id || comment.lesson_id === activeLessonId,
  );

  const commentMutation = useMutation({
    mutationFn: (input: { content: string; parent_comment_id?: string }) =>
      createCourseComment(courseId, {
        ...input,
        lesson_id: activeLessonId ?? undefined,
      }),
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

  const [downloaded, setDownloaded] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [downloadPct, setDownloadPct] = useState(0);
  useEffect(() => {
    setDownloaded(activeLessonId ? isLessonDownloaded(activeLessonId) : false);
    setDownloadPct(0);
  }, [activeLessonId]);

  async function handleDownload() {
    if (!lesson || !lesson.video_url || !activeLessonId) return;
    setDownloading(true);
    setDownloadPct(0);
    try {
      await downloadLesson(
        {
          courseId,
          courseTitle: course?.title ?? "Course",
          lessonId: activeLessonId,
          lessonTitle: lesson.title,
          url: lesson.video_url,
        },
        ({ received, total }) => {
          if (total > 0) setDownloadPct(Math.round((received / total) * 100));
        },
      );
      setDownloaded(true);
      setDownloadPct(100);
      toast.success("Saved for offline viewing");
    } catch (e) {
      toast.error(
        `${(e as Error).message ?? "Download failed"} — we'll retry when you're back online.`,
      );
    } finally {
      setDownloading(false);
    }
  }

  async function handleRemoveDownload() {
    if (!activeLessonId) return;
    await removeOfflineLesson(activeLessonId);
    setDownloaded(false);
    toast.success("Removed from offline library");
  }

  return (
    <div className="min-h-screen bg-surface text-brand font-sans flex flex-col lg:flex-row">
      {/* Sidebar */}
      <aside className="lg:w-80 lg:fixed lg:inset-y-0 lg:flex lg:flex-col border-b lg:border-b-0 lg:border-r border-brand/10 bg-white/40">
        <div className="px-5 py-5 border-b border-brand/10">
          <Link
            to="/student/my-courses"
            className="inline-flex items-center gap-1.5 text-xs text-brand/55 hover:text-brand"
          >
            <ChevronLeft className="h-3 w-3" /> My courses
          </Link>
          <p className="mt-3 font-serif text-lg leading-snug">{courseTitle}</p>
          <div className="mt-4">
            <div className="h-1 bg-brand/10">
              <div
                className="h-full bg-accent transition-all"
                style={{ width: `${Math.min(100, pct)}%` }}
              />
            </div>
            <p className="mt-2 text-[11px] text-brand/45">{Math.round(pct)}% complete</p>
          </div>
        </div>
        <nav className="overflow-y-auto flex-1">
          {modules.map((m, idx) => (
            <div key={m.id} className="border-b border-brand/5">
              <div className="px-5 pt-5 pb-2">
                <p className="eyebrow text-brand/40">Module {idx + 1}</p>
                <p className="font-serif text-sm mt-1">{m.title}</p>
              </div>
              <ul>
                {m.lessons.map((l) => {
                  const done = completedSet.has(l.id);
                  const active = l.id === activeLessonId;
                  return (
                    <li key={l.id}>
                      <button
                        onClick={() => setActiveLessonId(l.id)}
                        className={`w-full text-left px-5 py-2.5 flex items-start gap-3 text-sm transition-colors ${
                          active
                            ? "bg-brand/[0.05] text-brand"
                            : "text-brand/70 hover:bg-brand/[0.03] hover:text-brand"
                        }`}
                      >
                        {done ? (
                          <CheckCircle2 className="h-4 w-4 text-accent flex-shrink-0 mt-0.5" />
                        ) : (
                          <Circle className="h-4 w-4 text-brand/25 flex-shrink-0 mt-0.5" />
                        )}
                        <span className="flex-1 min-w-0">
                          <span className="block">{l.title}</span>
                          {typeof l.duration_minutes === "number" && (
                            <span className="text-[11px] text-brand/40">
                              {l.duration_minutes} min
                            </span>
                          )}
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </nav>
      </aside>

      {/* Main */}
      <main className="flex-1 lg:ml-80">
        {!activeLessonId ? (
          <div className="min-h-[60vh] grid place-items-center px-6">
            <p className="text-sm text-brand/55">Select a lesson to begin.</p>
          </div>
        ) : lessonLoading ? (
          <div className="px-6 md:px-10 lg:px-16 py-10 animate-pulse">
            <div className="aspect-video bg-brand/10" />
            <div className="mt-8 h-8 w-2/3 bg-brand/10" />
            <div className="mt-4 h-4 w-1/2 bg-brand/10" />
          </div>
        ) : lessonError ? (
          <article className="max-w-4xl mx-auto px-6 md:px-10 lg:px-16 py-10">
            <div className="aspect-video bg-brand/10 text-brand/45 grid place-items-center">
              <PlayCircle className="h-12 w-12 opacity-60" />
            </div>
            <header className="mt-8">
              <p className="eyebrow text-accent">Lesson</p>
              <h1 className="mt-3 font-serif text-3xl lg:text-4xl text-balance">
                {activeLessonSummary?.title ?? "Lesson"}
              </h1>
              {typeof activeLessonSummary?.duration_minutes === "number" && (
                <p className="mt-2 text-sm text-brand/55">
                  {activeLessonSummary.duration_minutes} minutes
                </p>
              )}
            </header>
            <QueryErrorPanel
              error={lessonErrorObj}
              title={lessonErrorTitle(lessonErrorObj)}
              onRetry={
                videoPolling
                  ? cancelVideoPolling
                  : () => {
                      refetchLesson();
                    }
              }
              retryLabel={videoPolling ? "Stop auto-retry" : "Try again"}
            >
              {videoPolling && (
                <p className="mt-3 text-xs text-brand/55">
                  Auto-retrying every 15 seconds (next try in {secondsUntilNextRetry}s).{" "}
                  <button
                    type="button"
                    onClick={cancelVideoPolling}
                    className="underline underline-offset-2"
                  >
                    Stop
                  </button>
                </p>
              )}
            </QueryErrorPanel>
          </article>
        ) : !lesson ? (
          <div className="min-h-[60vh] grid place-items-center px-6">
            <p className="text-sm text-brand/55">No lesson selected.</p>
          </div>
        ) : (
          <article className="max-w-4xl mx-auto px-6 md:px-10 lg:px-16 py-10">
            {/* Player surface */}
            {lesson.video_url ? (
              <div className="aspect-video bg-black overflow-hidden">
                <video key={lesson.id} src={lesson.video_url} controls className="h-full w-full" />
              </div>
            ) : (
              <div className="aspect-video bg-brand text-white grid place-items-center">
                <PlayCircle className="h-12 w-12 opacity-60" />
              </div>
            )}

            <header className="mt-8">
              <p className="eyebrow text-accent">Lesson</p>
              <h1 className="mt-3 font-serif text-3xl lg:text-4xl text-balance">{lesson.title}</h1>
              {typeof lesson.duration_minutes === "number" && (
                <p className="mt-2 text-sm text-brand/55">{lesson.duration_minutes} minutes</p>
              )}
            </header>

            {lesson.body_html && (
              <div
                className="mt-8 prose prose-sm max-w-none text-brand/80 leading-relaxed"
                dangerouslySetInnerHTML={{ __html: lesson.body_html }}
              />
            )}

            {lesson.resources && lesson.resources.length > 0 && (
              <section className="mt-10 border-t border-brand/10 pt-8">
                <h2 className="font-serif text-xl mb-4">Resources</h2>
                <ul className="space-y-2 text-sm">
                  {lesson.resources.map((r) => (
                    <li key={r.id}>
                      <a
                        href={r.url}
                        target="_blank"
                        rel="noreferrer"
                        className="text-accent hover:underline underline-offset-4"
                      >
                        {r.title} →
                      </a>
                    </li>
                  ))}
                </ul>
              </section>
            )}

            {activeNotes.length > 0 && (
              <section className="mt-10 border-t border-brand/10 pt-8">
                <h2 className="font-serif text-xl mb-4">Notes</h2>
                <ul className="space-y-3">
                  {activeNotes.map((note) => {
                    const locked = note.is_locked ?? (!note.is_free && !course?.is_enrolled);
                    return (
                      <li key={note.id} className="border border-brand/10 bg-white/50 p-4">
                        <div className="flex items-start gap-3">
                          {locked ? (
                            <Lock className="h-4 w-4 mt-0.5 text-brand/35" />
                          ) : (
                            <FileText className="h-4 w-4 mt-0.5 text-accent" />
                          )}
                          <div className="flex-1 min-w-0">
                            <p className="font-medium text-sm">{note.title}</p>
                            {locked ? (
                              <p className="mt-1 text-xs text-brand/50">
                                Locked until admin approval.
                              </p>
                            ) : (
                              <>
                                {note.content && (
                                  <p className="mt-2 text-sm text-brand/70 whitespace-pre-wrap">
                                    {note.content}
                                  </p>
                                )}
                                {note.file_url && (
                                  <a
                                    href={note.file_url}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="mt-3 inline-block text-xs text-accent hover:underline"
                                  >
                                    Open attachment
                                  </a>
                                )}
                              </>
                            )}
                          </div>
                        </div>
                      </li>
                    );
                  })}
                </ul>
              </section>
            )}

            <LessonDiscussion
              comments={activeComments}
              userId={user?.id}
              userRole={user?.role}
              saving={
                commentMutation.isPending ||
                updateCommentMutation.isPending ||
                deleteCommentMutation.isPending
              }
              onPost={(content) => commentMutation.mutate({ content })}
              onReply={(parentId, content) =>
                commentMutation.mutate({ content, parent_comment_id: parentId })
              }
              onUpdate={(commentId, content) =>
                updateCommentMutation.mutate({ commentId, content })
              }
              onPin={(commentId, isPinned) =>
                updateCommentMutation.mutate({ commentId, is_pinned: isPinned })
              }
              onDelete={(commentId) => deleteCommentMutation.mutate(commentId)}
            />

            <footer className="mt-12 pt-8 border-t border-brand/10 flex flex-wrap items-center justify-between gap-4">
              <p className="text-xs text-brand/45">
                {activeCompleted ? "Completed" : "Mark this lesson complete to advance."}
              </p>
              <div className="flex items-center gap-3">
                {lesson.video_url &&
                  (downloaded ? (
                    <button
                      onClick={handleRemoveDownload}
                      className="inline-flex items-center gap-1.5 px-3 py-2 text-xs border border-brand/20 text-brand/70 hover:text-destructive"
                    >
                      <Trash2 className="h-3.5 w-3.5" /> Remove offline
                    </button>
                  ) : (
                    <button
                      onClick={handleDownload}
                      disabled={downloading}
                      className="inline-flex items-center gap-1.5 px-3 py-2 text-xs border border-brand/20 text-brand/70 hover:text-brand disabled:opacity-50"
                    >
                      <Download className="h-3.5 w-3.5" />
                      {downloading
                        ? downloadPct > 0
                          ? `Saving ${downloadPct}%`
                          : "Saving…"
                        : "Save offline"}
                    </button>
                  ))}
                <button
                  disabled={activeCompleted || completeMutation.isPending}
                  onClick={() => completeMutation.mutate(activeLessonId)}
                  className="bg-brand text-white px-6 py-3 text-sm font-medium hover:bg-brand/90 transition-colors disabled:opacity-50"
                >
                  {activeCompleted
                    ? "Completed"
                    : completeMutation.isPending
                      ? "Saving…"
                      : "Mark complete & continue"}
                </button>
              </div>
            </footer>
          </article>
        )}
      </main>
    </div>
  );
}

function lessonErrorTitle(error: unknown) {
  if (!(error instanceof ApiError)) return "Couldn't load this lesson";
  if (error.code === "VIDEO_NOT_READY") return "Video is still processing";
  if (error.code === "NO_VIDEO") return "No video is attached to this lesson";
  return "Couldn't load this lesson";
}

function LessonDiscussion({
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
    <section className="mt-10 border-t border-brand/10 pt-8">
      <h2 className="font-serif text-xl mb-4">Discussion</h2>
      <form
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
          placeholder="Ask a question about this lesson."
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
      <ul className="mt-5 space-y-3">
        {roots.length === 0 && <li className="text-sm text-brand/45">No comments yet.</li>}
        {roots.map((comment) => {
          const canEdit = comment.user_id === userId;
          const canModerate = userRole === "teacher" || userRole === "admin";
          const canDelete = canEdit || canModerate;
          return (
            <li key={comment.id} className="border border-brand/10 bg-white/50 p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1">
                  {comment.is_pinned && (
                    <p className="mb-1 text-[10px] uppercase tracking-[0.18em] text-accent">
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
                  <p className="mt-2 text-[11px] text-brand/40">
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
                        onClick={() => onDelete(comment.id)}
                        disabled={saving}
                        className="text-brand/45 hover:text-destructive disabled:opacity-50"
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
                        type="submit"
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
                          <p className="mt-2 text-[11px] text-brand/40">
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

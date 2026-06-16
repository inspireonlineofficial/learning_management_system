import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { CheckCircle2, ChevronLeft, Circle, Download, PlayCircle, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import { getCourse } from "@/lib/api/courses";
import { completeLesson, getCourseProgress, getLesson } from "@/lib/api/student";
import { downloadLesson, isLessonDownloaded, removeOfflineLesson } from "@/lib/offline-lessons";

export const Route = createFileRoute("/_authenticated/student/player/$courseId")({
  component: PlayerPage,
});

function PlayerPage() {
  const { courseId } = Route.useParams();
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

  const { data: lesson, isLoading: lessonLoading } = useQuery({
    queryKey: ["lesson", courseId, activeLessonId],
    queryFn: () => getLesson(courseId, activeLessonId!),
    enabled: Boolean(activeLessonId),
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
  const courseTitle = course?.title ?? "Loading…";
  const pct = progress?.progress_percent ?? 0;

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
        ) : lessonLoading || !lesson ? (
          <div className="px-6 md:px-10 lg:px-16 py-10 animate-pulse">
            <div className="aspect-video bg-brand/10" />
            <div className="mt-8 h-8 w-2/3 bg-brand/10" />
            <div className="mt-4 h-4 w-1/2 bg-brand/10" />
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

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Clock, Lock, PlayCircle, Star, Users } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { PreviewPlayerDialog } from "@/components/course/preview-player-dialog";
import { useAuth } from "@/context/auth-context";
import { getCourse } from "@/lib/api/courses";
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
  const { isAuthenticated } = useAuth();
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

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      <header className="px-6 md:px-12 lg:px-20 py-6 flex items-center justify-between border-b border-brand/10">
        <Link to="/" className="font-serif italic text-2xl text-accent tracking-tight">
          Inspire LMS
        </Link>
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
              ) : isAuthenticated ? (
                course.price && course.price > 0 ? (
                  <Link
                    to="/student/checkout/$courseId"
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
                  Sign in to enroll
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

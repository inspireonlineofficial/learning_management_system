import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { FileText, Lock, PlayCircle } from "lucide-react";
import { useState } from "react";

import { AppShell } from "@/components/layout/app-shell";
import { getTeacherCourse } from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/preview")({
  component: Page,
});

type PreviewMode = "public" | "enrolled" | "paid";

function Page() {
  const { courseId } = Route.useParams();
  const [mode, setMode] = useState<PreviewMode>("public");
  const { data, isLoading } = useQuery({
    queryKey: ["teacher-course", courseId],
    queryFn: () => getTeacherCourse(courseId),
  });

  const hasFullAccess = mode === "enrolled" || mode === "paid";

  return (
    <AppShell eyebrow="Preview" title={data?.title ?? "Course preview"}>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <p className="max-w-xl text-sm text-brand/65">
          Preview the course as a public visitor, enrolled student, or paid/approved student.
        </p>
        <div className="inline-flex border border-brand/15 bg-white">
          {(
            [
              ["public", "Public visitor"],
              ["enrolled", "Enrolled student"],
              ["paid", "Paid/approved"],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => setMode(value)}
              className={`px-3 py-2 text-xs ${
                mode === value ? "bg-brand text-white" : "text-brand/65 hover:bg-brand/[0.04]"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <div className="mt-8 h-48 border border-brand/10 bg-brand/5 animate-pulse" />
      ) : (
        <div className="mt-8 grid lg:grid-cols-[1fr_320px] gap-6">
          <section>
            <h3 className="font-serif text-xl">Curriculum</h3>
            <div className="mt-4 space-y-3">
              {(data as any)?.modules?.map((module: any, moduleIndex: number) => {
                const moduleLocked = !module.is_free && !hasFullAccess;
                return (
                  <article key={module.id} className="border border-brand/10 bg-white/50">
                    <div className="p-4 flex items-start justify-between gap-3">
                      <div>
                        <p className="eyebrow text-brand/40">Module {moduleIndex + 1}</p>
                        <h4 className="mt-1 font-serif text-lg">{module.title}</h4>
                        {module.description && (
                          <p className="mt-1 text-sm text-brand/55">{module.description}</p>
                        )}
                      </div>
                      <AccessBadge locked={moduleLocked} free={module.is_free} />
                    </div>
                    <ul className="border-t border-brand/10 divide-y divide-brand/5">
                      {module.lessons?.map((lesson: any) => {
                        const lessonLocked =
                          moduleLocked || (!lesson.is_free && !lesson.is_preview && !hasFullAccess);
                        return (
                          <li
                            key={lesson.id}
                            className="p-4 flex items-center justify-between gap-3"
                          >
                            <span className="flex min-w-0 items-center gap-3 text-sm">
                              {lessonLocked ? (
                                <Lock className="h-4 w-4 text-brand/35" />
                              ) : (
                                <PlayCircle className="h-4 w-4 text-accent" />
                              )}
                              <span className="truncate">{lesson.title}</span>
                            </span>
                            <AccessBadge
                              locked={lessonLocked}
                              free={lesson.is_free || lesson.is_preview}
                            />
                          </li>
                        );
                      })}
                    </ul>
                  </article>
                );
              })}
            </div>
          </section>

          <aside className="border border-brand/10 bg-white/50 p-4 self-start">
            <h3 className="font-serif text-xl">Notes visibility</h3>
            <ul className="mt-4 space-y-3">
              {((data as any)?.notes ?? []).length === 0 && (
                <li className="text-sm text-brand/45">No notes yet.</li>
              )}
              {((data as any)?.notes ?? []).map((note: any) => {
                const locked = !note.is_free && !hasFullAccess;
                return (
                  <li key={note.id} className="flex items-start justify-between gap-3 text-sm">
                    <span className="flex min-w-0 items-center gap-2">
                      {locked ? (
                        <Lock className="h-4 w-4 text-brand/35" />
                      ) : (
                        <FileText className="h-4 w-4 text-accent" />
                      )}
                      <span className="truncate">{note.title}</span>
                    </span>
                    <AccessBadge locked={locked} free={note.is_free} />
                  </li>
                );
              })}
            </ul>
            <Link
              to="/teacher/courses/$courseId/edit"
              params={{ courseId }}
              className="mt-6 inline-block border border-brand/15 px-3 py-2 text-xs text-brand/70 hover:text-brand"
            >
              Back to builder
            </Link>
          </aside>
        </div>
      )}
    </AppShell>
  );
}

function AccessBadge({ locked, free }: { locked: boolean; free?: boolean }) {
  if (locked) {
    return (
      <span className="shrink-0 bg-brand/10 px-2 py-1 text-[10px] uppercase tracking-[0.16em] text-brand/55">
        Locked
      </span>
    );
  }
  return (
    <span className="shrink-0 bg-accent/10 px-2 py-1 text-[10px] uppercase tracking-[0.16em] text-accent">
      {free ? "Free" : "Approved"}
    </span>
  );
}

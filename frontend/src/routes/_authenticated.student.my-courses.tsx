import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { listMyEnrollments } from "@/lib/api/student";

export const Route = createFileRoute("/_authenticated/student/my-courses")({
  component: MyCoursesPage,
});

function MyCoursesPage() {
  const [status, setStatus] = useState<"active" | "completed">("active");

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["enrollments", status],
    queryFn: () => listMyEnrollments({ status, limit: 50 }),
  });

  return (
    <AppShell eyebrow="My courses" title="The work in progress.">
      <div className="flex gap-2 mb-8">
        <Tab active={status === "active"} onClick={() => setStatus("active")}>
          In progress
        </Tab>
        <Tab active={status === "completed"} onClick={() => setStatus("completed")}>
          Completed
        </Tab>
      </div>

      {isError && (
        <QueryErrorPanel
          error={error}
          title="Couldn't load your courses"
          onRetry={() => refetch()}
        />
      )}

      {isLoading && (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-48 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      )}

      {!isLoading && data && data.data.length === 0 && (
        <EmptyState
          title={status === "active" ? "No courses in progress" : "No completed courses yet"}
          description={
            status === "active"
              ? "Enroll in a course to see it here."
              : "Finish a course and it'll appear here with your certificate."
          }
          action={
            <Link to="/courses" className="bg-brand text-white px-6 py-3 text-sm">
              Browse catalog
            </Link>
          }
        />
      )}

      {!isLoading &&
        data &&
        (() => {
          // An enrollment's course can be null when the underlying course was
          // soft-deleted (e.g. by an admin). Hide those rows so a stale
          // enrollment does not crash the page.
          const visible = data.data.filter((e) => e.course != null);
          if (visible.length === 0) {
            return (
              <EmptyState
                title={status === "active" ? "No courses in progress" : "No completed courses yet"}
                description={
                  status === "active"
                    ? "Enroll in a course to see it here."
                    : "Finish a course and it'll appear here with your certificate."
                }
                action={
                  <Link to="/courses" className="bg-brand text-white px-6 py-3 text-sm">
                    Browse catalog
                  </Link>
                }
              />
            );
          }
          return (
            <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
              {visible.map((e) => {
                const course = e.course!;
                return (
                  <Link
                    key={e.id}
                    to="/student/player/$courseId"
                    params={{ courseId: course.id }}
                    className="group flex flex-col border border-brand/10 bg-white/50 hover:bg-white transition-colors"
                  >
                    <div className="aspect-[16/10] bg-brand/5 overflow-hidden">
                      {course.cover_url ? (
                        <img
                          src={course.cover_url}
                          alt={course.title}
                          loading="lazy"
                          className="h-full w-full object-cover group-hover:scale-[1.02] transition-transform duration-700"
                        />
                      ) : (
                        <div className="h-full w-full grid place-items-center font-serif italic text-3xl text-brand/20">
                          {course.title.slice(0, 1)}
                        </div>
                      )}
                    </div>
                    <div className="p-5 flex flex-col flex-1">
                      {course.category?.name && (
                        <p className="eyebrow text-accent mb-2">{course.category.name}</p>
                      )}
                      <h3 className="font-serif text-lg leading-snug">{course.title}</h3>
                      {e.next_lesson?.title && status === "active" && (
                        <p className="mt-3 text-xs text-brand/55">
                          Next · <span className="text-brand/75">{e.next_lesson.title}</span>
                        </p>
                      )}
                      <div className="mt-auto pt-5">
                        <div className="h-1 bg-brand/10">
                          <div
                            className="h-full bg-accent transition-all"
                            style={{ width: `${Math.min(100, e.progress_percent)}%` }}
                          />
                        </div>
                        <p className="mt-2 text-[11px] text-brand/45">
                          {Math.round(e.progress_percent)}% complete
                        </p>
                      </div>
                    </div>
                  </Link>
                );
              })}
            </div>
          );
        })()}
    </AppShell>
  );
}

function Tab({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`px-5 py-2 text-sm font-medium transition-colors ${
        active
          ? "bg-brand text-white"
          : "border border-brand/15 text-brand/70 hover:text-brand hover:bg-brand/[0.03]"
      }`}
    >
      {children}
    </button>
  );
}

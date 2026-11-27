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
      <div className="flex flex-wrap gap-2 mb-8">
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
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="h-56 rounded-xl border border-borderui bg-white animate-pulse"
            />
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
            <Link
              to="/courses"
              className="inline-flex items-center justify-center min-h-11 rounded-lg bg-navy text-white px-6 py-2.5 text-sm font-semibold hover:bg-navy-light shadow-sm transition-colors"
            >
              Browse catalog
            </Link>
          }
        />
      )}

      {!isLoading &&
        data &&
        (() => {
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
                  <Link
                    to="/courses"
                    className="inline-flex items-center justify-center min-h-11 rounded-lg bg-navy text-white px-6 py-2.5 text-sm font-semibold hover:bg-navy-light shadow-sm transition-colors"
                  >
                    Browse catalog
                  </Link>
                }
              />
            );
          }
          return (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
              {visible.map((e) => {
                const course = e.course!;
                return (
                  <Link
                    key={e.id}
                    to="/student/player/$courseId"
                    params={{ courseId: course.id }}
                    className="group flex flex-col rounded-xl overflow-hidden border border-borderui bg-white shadow-sm hover:border-navy transition-all focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                  >
                    <div className="aspect-[16/10] bg-stoneband overflow-hidden">
                      {course.cover_url ? (
                        <img
                          src={course.cover_url}
                          alt={course.title}
                          loading="lazy"
                          className="h-full w-full object-cover group-hover:scale-[1.02] transition-transform duration-700"
                        />
                      ) : (
                        <div className="h-full w-full grid place-items-center font-serif italic text-3xl text-navy/20">
                          {course.title.slice(0, 1)}
                        </div>
                      )}
                    </div>
                    <div className="p-5 flex flex-col flex-1">
                      {course.category?.name && (
                        <p className="text-xs font-bold tracking-[0.12em] uppercase text-bronze mb-2">
                          {course.category.name}
                        </p>
                      )}
                      <h3 className="font-serif text-xl font-bold text-navy leading-snug group-hover:text-bronze transition-colors">
                        {course.title}
                      </h3>
                      {e.next_lesson?.title && status === "active" && (
                        <p className="mt-2.5 text-xs text-slatebody">
                          Next &bull;{" "}
                          <span className="font-medium text-navy">{e.next_lesson.title}</span>
                        </p>
                      )}
                      <div className="mt-auto pt-5">
                        <div className="h-2 rounded-full bg-stoneband overflow-hidden">
                          <div
                            className="h-full rounded-full bg-navy transition-all"
                            style={{ width: `${Math.min(100, e.progress_percent)}%` }}
                          />
                        </div>
                        <p className="mt-2 text-xs font-semibold text-bronze">
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
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`inline-flex items-center justify-center min-h-[42px] px-5 py-2 rounded-lg text-sm font-semibold border transition-colors ${
        active
          ? "bg-navy text-white border-navy"
          : "bg-white text-navy-light border-borderui hover:border-navy hover:bg-stoneband"
      }`}
    >
      {children}
    </button>
  );
}

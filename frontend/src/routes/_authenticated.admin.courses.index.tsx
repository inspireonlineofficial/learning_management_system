import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { ListTable } from "@/components/layout/data-page";
import { apiRequest } from "@/lib/api/client";

type AdminCourse = {
  id: string;
  title: string;
  teacher_id: string;
  status: "draft" | "pending" | "published" | "rejected" | string;
  price_type?: "free" | "paid" | string;
  price?: number;
  currency?: string;
  total_enrolled?: number;
  rating_average?: number;
  updated_at?: string;
  submitted_at?: string;
};

type CoursesResponse = {
  data?: AdminCourse[];
  items?: AdminCourse[];
  courses?: AdminCourse[];
  meta?: { total?: number };
};

export const Route = createFileRoute("/_authenticated/admin/courses/")({
  component: Page,
});

function Page() {
  const [status, setStatus] = useState("");
  const [teacherId, setTeacherId] = useState("");

  const courses = useQuery({
    queryKey: ["admin-courses", status, teacherId],
    queryFn: () =>
      apiRequest<CoursesResponse>("/v1/admin/courses", {
        auth: true,
        query: {
          status: status || undefined,
          teacher_id: teacherId.trim() || undefined,
          limit: 100,
        },
      }).then((result) => ({
        items: result.items ?? result.data ?? result.courses ?? [],
        meta: result.meta,
      })),
  });

  return (
    <AppShell eyebrow="Courses" title="Course management">
      <div className="mt-6 flex flex-wrap gap-3">
        <select
          value={status}
          onChange={(event) => setStatus(event.target.value)}
          className="border border-brand/15 bg-white px-3 py-2 text-sm"
        >
          <option value="">All statuses</option>
          <option value="draft">Draft</option>
          <option value="pending">Pending review</option>
          <option value="published">Published</option>
          <option value="rejected">Rejected</option>
        </select>
        <input
          value={teacherId}
          onChange={(event) => setTeacherId(event.target.value)}
          placeholder="Filter by teacher UUID"
          className="min-w-72 border border-brand/15 bg-white px-3 py-2 text-sm"
        />
      </div>

      {courses.isLoading && (
        <div className="mt-10 grid gap-3">
          {Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="h-16 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      )}

      {courses.isError && (
        <div className="mt-10 border border-destructive/20 bg-destructive/5 p-6 text-sm">
          <p className="font-medium text-destructive">Couldn't load courses</p>
          <p className="mt-1 text-brand/60">{(courses.error as Error).message}</p>
          <button
            onClick={() => courses.refetch()}
            className="mt-3 bg-brand px-4 py-2 text-xs text-white"
          >
            Try again
          </button>
        </div>
      )}

      {courses.data && !courses.isLoading && !courses.isError && (
        <div className="mt-10">
          {courses.data.items.length === 0 ? (
            <EmptyState title="No courses found" description="Adjust filters to view courses." />
          ) : (
            <ListTable<AdminCourse>
              rows={courses.data.items}
              columns={[
                {
                  key: "title",
                  label: "Course",
                  render: (course) => (
                    <Link
                      to="/admin/courses/$courseId/review"
                      params={{ courseId: course.id }}
                      className="font-medium text-brand hover:text-accent"
                    >
                      {course.title}
                    </Link>
                  ),
                },
                {
                  key: "teacher",
                  label: "Teacher",
                  render: (course) => (
                    <span className="font-mono text-xs text-brand/55">{course.teacher_id}</span>
                  ),
                },
                {
                  key: "status",
                  label: "Status",
                  render: (course) => (
                    <span className="eyebrow text-brand/55">{course.status}</span>
                  ),
                },
                {
                  key: "access",
                  label: "Access",
                  render: (course) =>
                    course.price_type === "paid"
                      ? `${course.currency ?? "BDT"} ${course.price ?? 0}`
                      : "Free",
                },
                {
                  key: "students",
                  label: "Approved",
                  render: (course) => String(course.total_enrolled ?? 0),
                },
                {
                  key: "rating",
                  label: "Rating",
                  render: (course) => (course.rating_average ?? 0).toFixed(1),
                },
                {
                  key: "updated",
                  label: "Updated",
                  render: (course) =>
                    course.updated_at ? new Date(course.updated_at).toLocaleDateString() : "—",
                },
              ]}
            />
          )}
        </div>
      )}
    </AppShell>
  );
}

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { ListTable } from "@/components/layout/data-page";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
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
  const [pendingDelete, setPendingDelete] = useState<AdminCourse | null>(null);
  const qc = useQueryClient();

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

  const deleteCourse = useMutation({
    mutationFn: (courseId: string) =>
      apiRequest<{ ok: true }>(`/v1/admin/courses/${encodeURIComponent(courseId)}`, {
        method: "DELETE",
        auth: true,
      }),
    onSuccess: () => {
      toast.success("Course deleted");
      qc.invalidateQueries({ queryKey: ["admin-courses"] });
      setPendingDelete(null);
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not delete course"),
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
        <QueryErrorPanel
          error={courses.error}
          title="Couldn't load courses"
          onRetry={() => courses.refetch()}
          className="mt-10"
        />
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
                {
                  key: "actions",
                  label: "",
                  width: "1%",
                  render: (course) => (
                    <button
                      type="button"
                      onClick={(event) => {
                        event.stopPropagation();
                        setPendingDelete(course);
                      }}
                      className="inline-flex items-center gap-1.5 px-2 py-1 text-xs text-brand/55 hover:text-destructive border border-transparent hover:border-destructive/30 transition-colors"
                      aria-label={`Delete ${course.title}`}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                      Delete
                    </button>
                  ),
                },
              ]}
            />
          )}
        </div>
      )}

      {pendingDelete && (
        <div
          className="fixed inset-0 z-50 bg-black/40 grid place-items-center p-4"
          onClick={() => !deleteCourse.isPending && setPendingDelete(null)}
        >
          <div
            className="bg-white border border-brand/10 max-w-md w-full p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <p className="eyebrow text-destructive">Delete course</p>
            <p className="mt-2 font-serif text-xl">{pendingDelete.title}</p>
            <p className="mt-3 text-sm text-brand/70 leading-relaxed">
              This hides the course from the public catalog and the admin queue. Existing
              enrollments and progress are preserved; the teacher can no longer edit the course.
              Continue?
            </p>
            <div className="mt-5 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setPendingDelete(null)}
                disabled={deleteCourse.isPending}
                className="px-4 py-2 text-sm border border-brand/15 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => deleteCourse.mutate(pendingDelete.id)}
                disabled={deleteCourse.isPending}
                className="px-5 py-2 text-sm text-white bg-destructive hover:bg-destructive/90 disabled:opacity-50"
              >
                {deleteCourse.isPending ? "Deleting…" : "Delete course"}
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}

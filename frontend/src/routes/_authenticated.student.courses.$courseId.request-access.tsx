import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { DetailGrid } from "@/components/layout/data-page";
import { listMyCourseAccessRequests, requestCourseAccess } from "@/lib/api/access-requests";
import { getCourse } from "@/lib/api/courses";

export const Route = createFileRoute("/_authenticated/student/courses/$courseId/request-access")({
  component: RequestAccessPage,
});

function RequestAccessPage() {
  const { courseId } = Route.useParams();
  const navigate = useNavigate();
  const [note, setNote] = useState("");

  const course = useQuery({ queryKey: ["course", courseId], queryFn: () => getCourse(courseId) });
  const requests = useQuery({
    queryKey: ["my-course-access-requests", courseId],
    queryFn: () => listMyCourseAccessRequests({ limit: 100 }),
  });

  const current = useMemo(
    () => requests.data?.data.find((request) => request.item_id === courseId),
    [courseId, requests.data],
  );

  const request = useMutation({
    mutationFn: () => requestCourseAccess(courseId, note.trim()),
    onSuccess: (result) => {
      toast.success(
        result.status === "approved" ? "Access already approved" : "Access request submitted",
      );
      if (result.status === "approved") {
        navigate({ to: "/student/player/$courseId", params: { courseId } });
      } else {
        navigate({ to: "/student/access-requests" });
      }
    },
    onError: (error: Error) => toast.error(error.message),
  });

  if (current?.status === "approved") {
    return (
      <AppShell eyebrow="Approved Access" title={course.data?.title ?? "Course access approved"}>
        <p className="mb-6 text-sm text-brand/65">Your admin-approved access is active.</p>
        <Link
          to="/student/player/$courseId"
          params={{ courseId }}
          className="bg-brand px-5 py-2.5 text-sm text-white"
        >
          Open course
        </Link>
      </AppShell>
    );
  }

  return (
    <AppShell eyebrow="Request Access" title={course.data?.title ?? "Request course access"}>
      <div className="max-w-2xl">
        <DetailGrid
          items={[
            { label: "Course", value: course.data?.title ?? "..." },
            { label: "Teacher", value: course.data?.teacher?.full_name ?? "..." },
            {
              label: "Access",
              value: current ? statusLabel(current.status) : "Admin approval required",
            },
          ]}
        />

        {current?.status === "pending" && (
          <div className="mt-6 border border-brand/10 bg-white/50 p-5">
            <p className="font-medium text-brand">Pending admin approval</p>
            <p className="mt-1 text-sm text-brand/60">
              Your request was submitted on {new Date(current.created_at).toLocaleString()}.
            </p>
          </div>
        )}

        {current?.status === "rejected" && (
          <div className="mt-6 border border-destructive/20 bg-destructive/5 p-5">
            <p className="font-medium text-destructive">Access rejected</p>
            {current.rejection_reason && (
              <p className="mt-1 text-sm text-brand/65">{current.rejection_reason}</p>
            )}
          </div>
        )}

        {!current || current.status === "rejected" ? (
          <div className="mt-6 border border-brand/10 bg-white/50 p-5">
            <label className="block">
              <span className="eyebrow text-brand/45">Note for admin</span>
              <textarea
                value={note}
                onChange={(event) => setNote(event.target.value)}
                rows={4}
                className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              />
            </label>
            <button
              onClick={() => request.mutate()}
              disabled={request.isPending}
              className="mt-4 bg-brand px-6 py-2.5 text-sm text-white disabled:opacity-50"
            >
              {request.isPending ? "Submitting..." : "Request access"}
            </button>
          </div>
        ) : null}
      </div>
    </AppShell>
  );
}

function statusLabel(status: string) {
  return (
    (
      {
        pending: "Pending admin approval",
        approved: "Approved access",
        rejected: "Rejected access",
      } as Record<string, string>
    )[status] ?? status
  );
}

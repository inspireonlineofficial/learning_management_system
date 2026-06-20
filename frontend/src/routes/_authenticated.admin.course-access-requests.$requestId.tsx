import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { DetailGrid } from "@/components/layout/data-page";
import {
  approveCourseAccessRequest,
  listAdminCourseAccessRequests,
  rejectCourseAccessRequest,
} from "@/lib/api/access-requests";

export const Route = createFileRoute("/_authenticated/admin/course-access-requests/$requestId")({
  component: Page,
});

function Page() {
  const { requestId } = Route.useParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [reason, setReason] = useState("");
  const requests = useQuery({
    queryKey: ["admin-course-access-request", requestId],
    queryFn: () => listAdminCourseAccessRequests({ limit: 100 }),
  });
  const request = useMemo(
    () => requests.data?.data.find((item) => item.id === requestId),
    [requestId, requests.data],
  );

  const onSuccess = () => {
    qc.invalidateQueries({ queryKey: ["admin-course-access-requests"] });
    qc.invalidateQueries({ queryKey: ["admin-course-access-request", requestId] });
  };

  const approve = useMutation({
    mutationFn: () => approveCourseAccessRequest(requestId),
    onSuccess: () => {
      toast.success("Access approved");
      onSuccess();
    },
    onError: (error: Error) => toast.error(error.message),
  });
  const reject = useMutation({
    mutationFn: () => rejectCourseAccessRequest(requestId, reason.trim() || "Rejected by admin"),
    onSuccess: () => {
      toast.success("Access rejected");
      onSuccess();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  if (requests.isLoading) {
    return <AppShell eyebrow="Course Access" title="Loading request..." />;
  }

  if (!request) {
    return (
      <AppShell eyebrow="Course Access" title="Request unavailable">
        <EmptyState
          title="Request not found"
          action={
            <Link
              to="/admin/course-access-requests"
              className="bg-brand px-4 py-2 text-xs text-white"
            >
              Back to requests
            </Link>
          }
        />
      </AppShell>
    );
  }

  return (
    <AppShell eyebrow="Course Access" title={request.item_title}>
      <button
        onClick={() => navigate({ to: "/admin/course-access-requests" })}
        className="mb-6 text-xs text-brand/55"
      >
        Back to requests
      </button>
      <DetailGrid
        items={[
          { label: "Student", value: `${request.student_name} (${request.student_email})` },
          { label: "Course", value: request.item_title },
          { label: "Teacher", value: request.item_subtitle || "-" },
          { label: "Status", value: request.status },
          { label: "Requested", value: new Date(request.created_at).toLocaleString() },
          {
            label: "Reviewed",
            value: request.reviewed_at ? new Date(request.reviewed_at).toLocaleString() : "-",
          },
        ]}
      />
      {request.status === "pending" && (
        <div className="mt-6 max-w-xl border border-brand/10 bg-white/50 p-5">
          <label className="block">
            <span className="eyebrow text-brand/45">Rejection reason</span>
            <textarea
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              rows={3}
              className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
          <div className="mt-4 flex gap-2">
            <button
              onClick={() => approve.mutate()}
              disabled={approve.isPending || reject.isPending}
              className="bg-brand px-5 py-2 text-sm text-white disabled:opacity-50"
            >
              Approve
            </button>
            <button
              onClick={() => reject.mutate()}
              disabled={approve.isPending || reject.isPending}
              className="border border-destructive/30 px-5 py-2 text-sm text-destructive disabled:opacity-50"
            >
              Reject
            </button>
          </div>
        </div>
      )}
      {request.rejection_reason && (
        <p className="mt-6 max-w-xl text-sm text-brand/65">{request.rejection_reason}</p>
      )}
    </AppShell>
  );
}

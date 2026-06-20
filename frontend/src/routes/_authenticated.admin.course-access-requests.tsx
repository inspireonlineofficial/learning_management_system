import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";

import { DataPage, ListTable } from "@/components/layout/data-page";
import {
  approveCourseAccessRequest,
  listAdminCourseAccessRequests,
  rejectCourseAccessRequest,
  type CourseAccessRequest,
  type CourseAccessRequestStatus,
} from "@/lib/api/access-requests";

export const Route = createFileRoute("/_authenticated/admin/course-access-requests")({
  component: Page,
});

const FILTERS: Array<"all" | CourseAccessRequestStatus> = [
  "all",
  "pending",
  "approved",
  "rejected",
];

function Page() {
  const [status, setStatus] = useState<"all" | CourseAccessRequestStatus>("pending");
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: ["admin-course-access-requests"] });

  const approve = useMutation({
    mutationFn: approveCourseAccessRequest,
    onSuccess: () => {
      toast.success("Access approved");
      invalidate();
    },
    onError: (error: Error) => toast.error(error.message),
  });
  const reject = useMutation({
    mutationFn: (requestId: string) => rejectCourseAccessRequest(requestId, "Rejected by admin"),
    onSuccess: () => {
      toast.success("Access rejected");
      invalidate();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <DataPage
      eyebrow="Course Access"
      title="Course access requests"
      queryKey={["admin-course-access-requests", status]}
      queryFn={() => listAdminCourseAccessRequests(status === "all" ? {} : { status })}
      empty={{ title: "No course access requests" }}
      toolbar={
        <div className="flex flex-wrap gap-2">
          {FILTERS.map((filter) => (
            <button
              key={filter}
              onClick={() => setStatus(filter)}
              className={`border px-3 py-1.5 text-xs capitalize ${
                status === filter ? "border-brand bg-brand text-white" : "border-brand/15"
              }`}
            >
              {filter}
            </button>
          ))}
        </div>
      }
    >
      {(data: { data: CourseAccessRequest[] }) => (
        <ListTable
          rows={data.data}
          columns={[
            {
              key: "student",
              label: "Student",
              render: (request) => (
                <div>
                  <p className="font-medium">{request.student_name}</p>
                  <p className="text-xs text-brand/50">{request.student_email}</p>
                </div>
              ),
            },
            { key: "course", label: "Course", render: (request) => request.item_title },
            { key: "teacher", label: "Teacher", render: (request) => request.item_subtitle || "-" },
            {
              key: "status",
              label: "Status",
              render: (request) => <span className="eyebrow text-brand/55">{request.status}</span>,
            },
            {
              key: "requested",
              label: "Requested",
              render: (request) => new Date(request.created_at).toLocaleDateString(),
            },
            {
              key: "actions",
              label: "Actions",
              render: (request) => (
                <div className="flex flex-wrap gap-2">
                  <Link
                    to="/admin/course-access-requests/$requestId"
                    params={{ requestId: request.id }}
                    className="border border-brand/15 px-3 py-1.5 text-xs"
                  >
                    View
                  </Link>
                  {request.status === "pending" && (
                    <>
                      <button
                        onClick={() => approve.mutate(request.id)}
                        className="bg-brand px-3 py-1.5 text-xs text-white"
                      >
                        Approve
                      </button>
                      <button
                        onClick={() => reject.mutate(request.id)}
                        className="border border-destructive/30 px-3 py-1.5 text-xs text-destructive"
                      >
                        Reject
                      </button>
                    </>
                  )}
                </div>
              ),
            },
          ]}
        />
      )}
    </DataPage>
  );
}

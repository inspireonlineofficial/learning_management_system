import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { DetailGrid } from "@/components/layout/data-page";
import { createPurchaseRequest } from "@/lib/api/approvals";
import { getCourse } from "@/lib/api/courses";

export const Route = createFileRoute("/_authenticated/student/checkout/$courseId")({
  component: CourseApprovalRequestPage,
});

function CourseApprovalRequestPage() {
  const { courseId } = Route.useParams();
  const navigate = useNavigate();
  const [fileName, setFileName] = useState("");
  const [note, setNote] = useState("");
  const { data } = useQuery({ queryKey: ["course", courseId], queryFn: () => getCourse(courseId) });

  const request = useMutation({
    mutationFn: () =>
      createPurchaseRequest({
        item_type: "course",
        item_id: courseId,
        note: [fileName ? `File: ${fileName}` : "", note].filter(Boolean).join("\n"),
      }),
    onSuccess: () => {
      toast.success("Approval request submitted");
      navigate({ to: "/student/bookshop/requests" });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <AppShell eyebrow="Approval request" title={data?.title ?? "Request course approval"}>
      <div className="max-w-2xl">
        <DetailGrid
          items={[
            { label: "Course", value: data?.title ?? "—" },
            { label: "Teacher", value: data?.teacher?.full_name ?? "—" },
            {
              label: "Price",
              value: data ? (data.price ? `${data.currency ?? "USD"} ${data.price}` : "Free") : "—",
            },
            {
              label: "Duration",
              value: data?.duration_minutes ? `${Math.round(data.duration_minutes / 60)}h` : "—",
            },
          ]}
        />

        <div className="mt-6 border border-brand/10 bg-white/50 p-5 space-y-4">
          <label className="block">
            <span className="eyebrow text-brand/45">Optional file name</span>
            <input
              value={fileName}
              onChange={(event) => setFileName(event.target.value)}
              placeholder="receipt.pdf, guardian-note.jpg"
              className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
          <label className="block">
            <span className="eyebrow text-brand/45">Note for admin</span>
            <textarea
              value={note}
              onChange={(event) => setNote(event.target.value)}
              rows={4}
              className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
        </div>

        <div className="mt-6 flex flex-wrap gap-3">
          <button
            onClick={() => request.mutate()}
            disabled={request.isPending}
            className="bg-brand text-white px-8 py-3 text-sm disabled:opacity-50"
          >
            {request.isPending ? "Submitting..." : "Submit approval request"}
          </button>
          <Link
            to="/student/bookshop/requests"
            className="border border-brand/15 px-8 py-3 text-sm"
          >
            View request history
          </Link>
        </div>
      </div>
    </AppShell>
  );
}

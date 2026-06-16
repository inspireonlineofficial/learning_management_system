import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { DataPage } from "@/components/layout/data-page";
import { approve, listApprovals, reject, type ApprovalItem } from "@/lib/api/approvals";

export const Route = createFileRoute("/_authenticated/admin/approvals")({
  component: Page,
});

type Filter = "all" | ApprovalItem["kind"];
const FILTERS: { value: Filter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "course_publish", label: "Course publish" },
  { value: "book_purchase", label: "Book purchase" },
  { value: "refund", label: "Refund" },
  { value: "role_change", label: "Role change" },
];

function Page() {
  const qc = useQueryClient();
  const [filter, setFilter] = useState<Filter>("all");
  const [decision, setDecision] = useState<{
    item: ApprovalItem;
    action: "approve" | "reject";
  } | null>(null);
  const [note, setNote] = useState("");

  const invalidate = () => qc.invalidateQueries({ queryKey: ["approvals"] });

  const ap = useMutation({
    mutationFn: ({ id, note }: { id: string; note?: string }) => approve(id, note),
    onSuccess: () => {
      toast.success("Approved");
      invalidate();
      closeDialog();
    },
    onError: (e: Error) => toast.error(e.message),
  });
  const rj = useMutation({
    mutationFn: ({ id, note }: { id: string; note?: string }) => reject(id, note),
    onSuccess: () => {
      toast.success("Rejected");
      invalidate();
      closeDialog();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  function closeDialog() {
    setDecision(null);
    setNote("");
  }

  function confirm() {
    if (!decision) return;
    const payload = { id: decision.item.id, note: note.trim() || undefined };
    if (decision.action === "approve") ap.mutate(payload);
    else rj.mutate(payload);
  }

  return (
    <>
      <DataPage
        eyebrow="Approvals"
        title="Pending approvals"
        queryKey={["approvals"]}
        queryFn={listApprovals}
        empty={{ title: "Nothing pending" }}
        toolbar={
          <div className="flex flex-wrap gap-2">
            {FILTERS.map((f) => (
              <button
                key={f.value}
                onClick={() => setFilter(f.value)}
                className={`px-4 py-1.5 text-xs font-medium border transition-colors ${
                  filter === f.value
                    ? "bg-brand text-white border-brand"
                    : "border-brand/15 text-brand/65 hover:text-brand hover:bg-brand/[0.03]"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>
        }
      >
        {(data: { items: ApprovalItem[] }) => {
          const filtered =
            filter === "all" ? data.items : data.items.filter((it) => it.kind === filter);
          if (filtered.length === 0) {
            return <p className="text-sm text-brand/55">No items in this category.</p>;
          }
          return (
            <ul className="space-y-3">
              {filtered.map((it) => (
                <li
                  key={it.id}
                  className="border border-brand/10 bg-white/40 p-5 flex flex-col sm:flex-row sm:justify-between sm:items-start gap-4"
                >
                  <div className="min-w-0">
                    <p className="eyebrow text-brand/55">{kindLabel(it.kind)}</p>
                    <p className="mt-1 text-sm">{it.payload_summary}</p>
                    <p className="mt-1 text-xs text-brand/45">
                      By {it.requester.full_name} · {new Date(it.created_at).toLocaleString()}
                    </p>
                  </div>
                  <div className="flex gap-2 flex-shrink-0">
                    <button
                      onClick={() => setDecision({ item: it, action: "approve" })}
                      className="text-xs bg-brand text-white px-3 py-1.5 hover:bg-brand/90"
                    >
                      Approve
                    </button>
                    <button
                      onClick={() => setDecision({ item: it, action: "reject" })}
                      className="text-xs border border-destructive/40 text-destructive px-3 py-1.5 hover:bg-destructive/5"
                    >
                      Reject
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          );
        }}
      </DataPage>

      {decision && (
        <div
          className="fixed inset-0 z-50 bg-black/40 grid place-items-center p-4"
          onClick={closeDialog}
        >
          <div
            className="bg-white border border-brand/10 max-w-md w-full p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <p className="eyebrow text-brand/55">
              {decision.action === "approve" ? "Approve request" : "Reject request"}
            </p>
            <p className="mt-2 font-serif text-xl">{decision.item.payload_summary}</p>
            <p className="mt-1 text-xs text-brand/55">
              {kindLabel(decision.item.kind)} · {decision.item.requester.full_name}
            </p>
            <label className="block mt-5">
              <span className="eyebrow text-brand/55">
                Note {decision.action === "reject" ? "(recommended)" : "(optional)"}
              </span>
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                rows={4}
                maxLength={1000}
                placeholder={
                  decision.action === "reject"
                    ? "Tell the requester why this was rejected…"
                    : "Optional note attached to the decision"
                }
                className="mt-1 w-full p-3 border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
              />
            </label>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={closeDialog} className="px-4 py-2 text-sm border border-brand/15">
                Cancel
              </button>
              <button
                onClick={confirm}
                disabled={ap.isPending || rj.isPending}
                className={`px-5 py-2 text-sm text-white disabled:opacity-60 ${
                  decision.action === "approve" ? "bg-brand" : "bg-destructive"
                }`}
              >
                {ap.isPending || rj.isPending
                  ? "Saving…"
                  : decision.action === "approve"
                    ? "Approve"
                    : "Reject"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function kindLabel(k: ApprovalItem["kind"]) {
  return (
    {
      course_publish: "Course publish",
      book_purchase: "Book purchase",
      refund: "Refund",
      role_change: "Role change",
    } as const
  )[k];
}

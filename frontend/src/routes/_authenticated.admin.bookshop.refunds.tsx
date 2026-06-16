import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { DataPage } from "@/components/layout/data-page";
import { listRefunds, decideRefund, type RefundRequest } from "@/lib/api/refunds";

export const Route = createFileRoute("/_authenticated/admin/bookshop/refunds")({
  component: Page,
});

function Page() {
  const qc = useQueryClient();
  const [notes, setNotes] = useState<Record<string, string>>({});
  const mut = useMutation({
    mutationFn: (args: { id: string; action: "approve" | "reject"; note?: string }) =>
      decideRefund(args.id, args.action, args.note),
    onSuccess: () => {
      toast.success("Done");
      qc.invalidateQueries({ queryKey: ["refunds"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <DataPage
      eyebrow="Refunds"
      title="Refund queue"
      queryKey={["refunds"]}
      queryFn={listRefunds}
      empty={{ title: "No pending refunds" }}
    >
      {(data) => (
        <ul className="space-y-3">
          {data.items.map((r: RefundRequest) => (
            <li key={r.id} className="border border-brand/10 bg-white/40 p-5 space-y-3">
              <div className="flex justify-between items-start gap-4">
                <div className="min-w-0">
                  <p className="eyebrow text-brand/55">Order {r.order_id.slice(0, 8)}</p>
                  <p className="mt-1 text-sm">{r.reason}</p>
                  <p className="mt-1 font-serif text-lg">
                    {r.currency} {r.amount.toFixed(2)}
                  </p>
                  <p className="text-xs text-brand/45">
                    By {r.requester.full_name} · {new Date(r.created_at).toLocaleDateString()} ·{" "}
                    <span className="capitalize">{r.status}</span>
                  </p>
                </div>
              </div>
              {r.status === "pending" && (
                <>
                  <textarea
                    value={notes[r.id] ?? ""}
                    onChange={(e) => setNotes({ ...notes, [r.id]: e.target.value })}
                    placeholder="Internal note (optional)…"
                    rows={2}
                    className="w-full text-sm border border-brand/15 bg-white p-2"
                  />
                  <div className="flex gap-2">
                    <button
                      onClick={() => mut.mutate({ id: r.id, action: "approve", note: notes[r.id] })}
                      disabled={mut.isPending}
                      className="text-xs bg-brand text-white px-4 py-2 disabled:opacity-50"
                    >
                      Approve
                    </button>
                    <button
                      onClick={() => mut.mutate({ id: r.id, action: "reject", note: notes[r.id] })}
                      disabled={mut.isPending}
                      className="text-xs border border-brand/15 px-4 py-2 hover:bg-brand/[0.03] disabled:opacity-50"
                    >
                      Reject
                    </button>
                  </div>
                </>
              )}
            </li>
          ))}
        </ul>
      )}
    </DataPage>
  );
}

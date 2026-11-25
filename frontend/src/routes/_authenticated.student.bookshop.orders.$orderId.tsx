import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { DetailGrid } from "@/components/layout/data-page";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import {
  formatAddress,
  formatPrice,
  payoutMethodLabel,
  requestRefund,
  type BookOrder,
} from "@/lib/api/bookshop";
import { getMyOrder, orderStatusLabel } from "@/lib/api/orders";

export const Route = createFileRoute("/_authenticated/student/bookshop/orders/$orderId")({
  component: Page,
});

function Page() {
  const { orderId } = Route.useParams();
  const { data, isError, error, refetch } = useQuery({
    queryKey: ["my-order", orderId],
    queryFn: () => getMyOrder(orderId),
  });

  return (
    <AppShell eyebrow="Order" title={data?.book_title || `Order ${orderId.slice(0, 8)}`}>
      {isError && (
        <QueryErrorPanel error={error} title="Couldn't load this order" onRetry={() => refetch()} />
      )}
      {data && (
        <>
          <DetailGrid
            items={[
              { label: "Status", value: orderStatusLabel[data.status] ?? data.status },
              { label: "Edition", value: data.format === "physical" ? "Printed copy" : "Digital" },
              { label: "Placed", value: new Date(data.created_at).toLocaleString() },
              {
                label: "Price",
                value:
                  data.amount === 0
                    ? "Free"
                    : formatPrice(Math.round(data.amount * 100), data.currency),
              },
              ...(data.format === "physical"
                ? [
                    { label: "Ship to", value: formatAddress(data.shipping_address) },
                    { label: "Tracking number", value: data.tracking_number || "Not shipped yet" },
                    ...(data.shipped_at
                      ? [{ label: "Shipped", value: new Date(data.shipped_at).toLocaleString() }]
                      : []),
                    ...(data.delivered_at
                      ? [
                          {
                            label: "Delivered",
                            value: new Date(data.delivered_at).toLocaleString(),
                          },
                        ]
                      : []),
                  ]
                : []),
            ]}
          />
          <div className="mt-8 flex flex-wrap gap-3">
            {data.format === "digital" &&
              (data.status === "placed" || data.status === "delivered") && (
                <Link
                  to="/student/bookshop/reader/$bookId"
                  params={{ bookId: data.book_id }}
                  className="bg-brand px-5 py-2.5 text-sm text-white"
                >
                  Read now
                </Link>
              )}
            <Link
              to="/student/bookshop/$bookId"
              params={{ bookId: data.book_id }}
              className="border border-brand/15 px-5 py-2.5 text-sm"
            >
              View book
            </Link>
          </div>
          <RefundSection order={data} />
        </>
      )}
    </AppShell>
  );
}

const REQUEST_STATUS = {
  pending: "Waiting for review",
  approved: "Approved",
  rejected: "Declined",
};

function RefundSection({ order }: { order: BookOrder }) {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const info = order.refund_info;
  const mut = useMutation({
    mutationFn: () => requestRefund(order.id, reason.trim()),
    onSuccess: () => {
      toast.success("Refund request sent");
      setOpen(false);
      setReason("");
      qc.invalidateQueries({ queryKey: ["my-order", order.id] });
    },
    onError: (e: Error) => toast.error(e.message),
  });
  if (!info) return null;
  const req = info.request;

  return (
    <section
      className="mt-10 max-w-2xl border-t border-brand/10 pt-6"
      aria-labelledby="refund-heading"
    >
      <h2 id="refund-heading" className="font-serif text-xl">
        Refund
      </h2>
      {info.refund ? (
        <p className="mt-2 text-sm">
          Refunded {formatPrice(Math.round(info.refund.amount * 100), info.refund.currency)}
          {info.refund.payout_method !== "none" && (
            <>
              {" "}
              via {payoutMethodLabel[info.refund.payout_method]}
              {info.refund.payout_reference ? ` (transaction ${info.refund.payout_reference})` : ""}
            </>
          )}{" "}
          on {new Date(info.refund.refunded_at).toLocaleDateString()}.
          {info.refund.note ? ` ${info.refund.note}` : ""}
        </p>
      ) : req && req.status !== "approved" ? (
        <div className="mt-2 text-sm">
          <p>
            Your request ({new Date(req.created_at).toLocaleDateString()}):{" "}
            <span className="font-medium">{REQUEST_STATUS[req.status]}</span>
          </p>
          {req.admin_note && <p className="mt-1 text-brand/70">{req.admin_note}</p>}
        </div>
      ) : null}

      {info.eligible ? (
        open ? (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              mut.mutate();
            }}
            className="mt-4 space-y-3"
          >
            <label className="block">
              <span className="eyebrow text-brand/70">Why do you want a refund?</span>
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                required
                maxLength={2000}
                rows={3}
                className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              />
            </label>
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={mut.isPending || !reason.trim()}
                className="bg-brand px-4 py-2 text-sm text-white disabled:opacity-50"
              >
                {mut.isPending ? "Sending…" : "Send refund request"}
              </button>
              <button
                type="button"
                onClick={() => setOpen(false)}
                className="border border-brand/15 px-4 py-2 text-sm"
              >
                Cancel
              </button>
            </div>
          </form>
        ) : (
          <div className="mt-3">
            <button
              onClick={() => setOpen(true)}
              className="border border-brand/15 px-4 py-2 text-sm"
            >
              Request a refund
            </button>
            {info.deadline && (
              <p className="mt-1 text-xs text-brand/70">
                Available until {new Date(info.deadline).toLocaleDateString()}.
              </p>
            )}
          </div>
        )
      ) : (
        !info.refund &&
        info.blocked_reason &&
        !(req && req.status === "pending") && (
          <p className="mt-2 text-sm text-brand/70">{info.blocked_reason}</p>
        )
      )}
    </section>
  );
}

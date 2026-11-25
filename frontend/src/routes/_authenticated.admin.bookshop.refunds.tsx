import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

import { RefundPayoutForm } from "@/components/bookshop/refund-payout-form";
import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import {
  formatPrice,
  newIdempotencyKey,
  payoutMethodLabel,
  type RefundRequest,
} from "@/lib/api/bookshop";
import {
  approveRefundRequest,
  issueRefund,
  listRefundRequests,
  listRefunds,
  rejectRefundRequest,
  type PayoutInput,
  type RefundableOrder,
} from "@/lib/api/refunds";

export const Route = createFileRoute("/_authenticated/admin/bookshop/refunds")({
  component: Page,
});

type Tab = "requests" | "orders";

const money = (amount: number, currency: string) =>
  amount === 0 ? "Free" : formatPrice(Math.round(amount * 100), currency);

/**
 * Student refund requests (approve with payout details, or decline with a
 * note) and every refundable order (refund directly). Each refund records
 * how the money was returned and its transaction ID; the LMS does not move
 * money itself.
 */
function Page() {
  const [tab, setTab] = useState<Tab>("requests");
  const pending = useQuery({
    queryKey: ["refund-requests", "pending"],
    queryFn: () => listRefundRequests("pending"),
  });
  const count = pending.data?.meta.total ?? 0;

  return (
    <AppShell eyebrow="Refunds" title="Book refunds">
      <Link
        to="/admin/bookshop"
        className="mb-4 inline-block text-xs text-brand/70 hover:text-brand"
      >
        ← Bookshop admin
      </Link>
      <div role="tablist" className="mb-6 flex w-fit border border-brand/15 bg-white/40">
        {(
          [
            ["requests", `Student requests${count ? ` (${count})` : ""}`],
            ["orders", "All orders"],
          ] as const
        ).map(([key, label]) => (
          <button
            key={key}
            role="tab"
            aria-selected={tab === key}
            onClick={() => setTab(key)}
            className={`px-4 py-2 text-xs ${tab === key ? "bg-brand text-white" : "text-brand/70"}`}
          >
            {label}
          </button>
        ))}
      </div>
      {tab === "requests" ? <RequestsTab /> : <OrdersTab />}
    </AppShell>
  );
}

function RequestsTab() {
  const [status, setStatus] = useState<RefundRequest["status"]>("pending");
  const q = useQuery({
    queryKey: ["refund-requests", status],
    queryFn: () => listRefundRequests(status),
  });
  return (
    <>
      <div className="mb-4 flex gap-2 text-xs">
        {(["pending", "approved", "rejected"] as const).map((s) => (
          <button
            key={s}
            onClick={() => setStatus(s)}
            aria-pressed={status === s}
            className={`border px-3 py-1.5 capitalize ${
              status === s ? "border-brand bg-brand text-white" : "border-brand/15"
            }`}
          >
            {s}
          </button>
        ))}
      </div>
      {q.isError ? (
        <QueryErrorPanel
          error={q.error}
          title="Couldn't load refund requests"
          onRetry={() => q.refetch()}
        />
      ) : q.isLoading ? (
        <div className="h-32 animate-pulse bg-brand/5" />
      ) : !q.data?.data.length ? (
        <EmptyState title={`No ${status} refund requests`} />
      ) : (
        <ul className="space-y-3">
          {q.data.data.map((r) => (
            <RequestRow key={r.id} request={r} />
          ))}
        </ul>
      )}
    </>
  );
}

function RequestRow({ request: r }: { request: RefundRequest }) {
  const qc = useQueryClient();
  const [mode, setMode] = useState<"idle" | "approve" | "reject">("idle");
  const [note, setNote] = useState("");
  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["refund-requests"] });
    qc.invalidateQueries({ queryKey: ["refunds"] });
  };
  const approve = useMutation({
    mutationFn: (p: PayoutInput) => approveRefundRequest(r.id, p),
    onSuccess: () => {
      toast.success("Refund recorded and the student notified");
      refresh();
    },
    onError: (e: Error) => toast.error(e.message),
  });
  const reject = useMutation({
    mutationFn: () => rejectRefundRequest(r.id, note.trim()),
    onSuccess: () => {
      toast.success("Request declined and the student notified");
      refresh();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <li className="border border-brand/10 bg-white/40 p-5" data-testid="refund-request">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="eyebrow text-brand/70">
            {r.format === "physical" ? "Printed copy" : "Digital"} · order {r.order_status} · #
            {r.order_id.slice(0, 8)}
          </p>
          <p className="mt-1 font-serif text-lg">{r.book_title}</p>
          <p className="text-sm text-brand/70">
            {r.student_name}
            {r.student_email ? ` · ${r.student_email}` : ""}
          </p>
          <p className="mt-2 text-sm">
            <span className="text-brand/70">Reason: </span>
            {r.reason}
          </p>
          {r.admin_note && (
            <p className="mt-1 text-sm">
              <span className="text-brand/70">Admin note: </span>
              {r.admin_note}
            </p>
          )}
          <p className="text-xs text-brand/70">
            Requested {new Date(r.created_at).toLocaleString()}
          </p>
        </div>
        <div className="text-right">
          <p className="font-serif text-lg">{money(r.amount, r.currency)}</p>
          {r.status === "pending" && mode === "idle" && (
            <div className="mt-2 flex gap-2">
              <button
                onClick={() => setMode("approve")}
                className="bg-brand px-3 py-2 text-xs text-white"
              >
                Approve refund
              </button>
              <button
                onClick={() => setMode("reject")}
                className="border border-brand/15 px-3 py-2 text-xs"
              >
                Decline
              </button>
            </div>
          )}
        </div>
      </div>
      {mode === "approve" && (
        <RefundPayoutForm
          amount={r.amount}
          currency={r.currency}
          submitLabel="Record refund"
          busy={approve.isPending}
          onSubmit={(p) => approve.mutate(p)}
          onCancel={() => setMode("idle")}
        />
      )}
      {mode === "reject" && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            reject.mutate();
          }}
          className="mt-4 space-y-3 border-t border-brand/10 pt-4"
        >
          <label className="block">
            <span className="eyebrow text-brand/70">Why is it declined? (sent to the student)</span>
            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              required
              maxLength={2000}
              rows={2}
              className="mt-1 w-full border border-brand/15 bg-white px-2 py-2 text-sm"
            />
          </label>
          <div className="flex gap-2">
            <button
              type="submit"
              disabled={reject.isPending || !note.trim()}
              className="bg-brand px-4 py-2 text-xs text-white disabled:opacity-50"
            >
              Decline request
            </button>
            <button
              type="button"
              onClick={() => setMode("idle")}
              className="border border-brand/15 px-4 py-2 text-xs"
            >
              Cancel
            </button>
          </div>
        </form>
      )}
    </li>
  );
}

function OrdersTab() {
  const q = useQuery({ queryKey: ["refunds"], queryFn: listRefunds });
  return q.isError ? (
    <QueryErrorPanel error={q.error} title="Couldn't load orders" onRetry={() => q.refetch()} />
  ) : q.isLoading ? (
    <div className="h-32 animate-pulse bg-brand/5" />
  ) : !q.data?.items.length ? (
    <EmptyState title="No orders to refund" />
  ) : (
    <ul className="space-y-3">
      {q.data.items.map((r) => (
        <OrderRow key={r.id} order={r} />
      ))}
    </ul>
  );
}

const STATUS: Record<RefundableOrder["order_status"], string> = {
  placed: "Placed",
  shipped: "Shipped",
  delivered: "Delivered",
  refunded: "Refunded",
};

function OrderRow({ order: r }: { order: RefundableOrder }) {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [key] = useState(newIdempotencyKey);
  const mut = useMutation({
    mutationFn: (p: PayoutInput) => issueRefund(r.order_id, key, p),
    onSuccess: (res) => {
      toast.success(
        res.access_revoked
          ? "Refunded — the book was removed from the student's library"
          : res.stock_restored
            ? "Refunded — the copy is back in stock"
            : "Refunded",
      );
      qc.invalidateQueries({ queryKey: ["refunds"] });
      qc.invalidateQueries({ queryKey: ["refund-requests"] });
      qc.invalidateQueries({ queryKey: ["admin-orders"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <li className="border border-brand/10 bg-white/40 p-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="eyebrow text-brand/70">
            {STATUS[r.order_status]} · {r.format === "physical" ? "Printed copy" : "Digital"} · #
            {r.order_id.slice(0, 8)}
            {r.pending_request_id ? " · refund requested" : ""}
          </p>
          <p className="mt-1 font-serif text-lg">{r.book_title}</p>
          <p className="text-sm text-brand/70">
            {r.requester.full_name}
            {r.requester.email ? ` · ${r.requester.email}` : ""}
          </p>
          <p className="text-xs text-brand/70">
            Ordered {new Date(r.created_at).toLocaleDateString()}
          </p>
          {r.refund && (
            <p className="mt-1 text-xs text-brand/70">
              Refunded {money(r.refund.amount, r.refund.currency)} via{" "}
              {payoutMethodLabel[r.refund.payout_method]}
              {r.refund.payout_reference ? ` (${r.refund.payout_reference})` : ""} on{" "}
              {new Date(r.refund.refunded_at).toLocaleDateString()}
            </p>
          )}
        </div>
        <div className="text-right">
          <p className="font-serif text-lg">{money(r.amount, r.currency)}</p>
          {r.refundable && !open && (
            <button
              onClick={() => setOpen(true)}
              className="mt-2 bg-brand px-4 py-2 text-xs text-white"
            >
              Refund…
            </button>
          )}
        </div>
      </div>
      {open && r.refundable && (
        <RefundPayoutForm
          amount={r.amount}
          currency={r.currency}
          submitLabel="Record refund"
          busy={mut.isPending}
          onSubmit={(p) => mut.mutate(p)}
          onCancel={() => setOpen(false)}
        />
      )}
    </li>
  );
}

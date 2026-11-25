import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { formatAddress, formatPrice } from "@/lib/api/bookshop";
import { listAllOrders, orderStatusLabel, updateOrderStatus, type Order } from "@/lib/api/orders";

export const Route = createFileRoute("/_authenticated/admin/bookshop/orders")({
  component: Page,
});

const STATUSES: Array<Order["status"] | "all"> = [
  "all",
  "placed",
  "shipped",
  "delivered",
  "refunded",
  "cancelled",
];

function Page() {
  const [status, setStatus] = useState<Order["status"] | "all">("placed");
  const [page, setPage] = useState(1);
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["admin-orders", status, page],
    queryFn: () =>
      listAllOrders({ status: status === "all" ? undefined : status, page, limit: 50 }),
  });

  return (
    <AppShell eyebrow="Orders" title="Book orders">
      <Link
        to="/admin/bookshop"
        className="inline-block text-xs text-brand/70 hover:text-brand mb-4"
      >
        ← Bookshop admin
      </Link>

      <div className="mb-6 flex flex-wrap gap-1 border border-brand/15 bg-white/40 w-fit">
        {STATUSES.map((s) => (
          <button
            key={s}
            onClick={() => {
              setStatus(s);
              setPage(1);
            }}
            aria-pressed={status === s}
            className={`px-3 py-2 text-xs capitalize ${
              status === s ? "bg-brand text-white" : "text-brand/65 hover:bg-brand/[0.03]"
            }`}
          >
            {s === "all" ? "All" : orderStatusLabel[s]}
          </button>
        ))}
      </div>

      {isError ? (
        <QueryErrorPanel error={error} title="Couldn't load orders" onRetry={() => refetch()} />
      ) : isLoading ? (
        <div className="h-40 border border-brand/10 bg-white/30 animate-pulse" />
      ) : !data || data.items.length === 0 ? (
        <EmptyState title="No orders" />
      ) : (
        <>
          <ul className="space-y-3">
            {data.items.map((o) => (
              <OrderRow key={o.id} order={o} />
            ))}
          </ul>
          {data.meta && data.meta.total_pages > 1 && (
            <div className="mt-6 flex items-center gap-3 text-sm">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
                className="border border-brand/15 px-3 py-1.5 disabled:opacity-40"
              >
                Previous
              </button>
              <span className="text-brand/60">
                Page {data.meta.page} of {data.meta.total_pages}
              </span>
              <button
                disabled={page >= data.meta.total_pages}
                onClick={() => setPage((p) => p + 1)}
                className="border border-brand/15 px-3 py-1.5 disabled:opacity-40"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </AppShell>
  );
}

function OrderRow({ order: o }: { order: Order }) {
  const qc = useQueryClient();
  const [tracking, setTracking] = useState("");
  const mut = useMutation({
    mutationFn: (args: { status: "shipped" | "delivered" | "cancelled"; tracking?: string }) =>
      updateOrderStatus(o.id, args.status, args.tracking),
    onSuccess: (updated) => {
      toast.success(`Order ${orderStatusLabel[updated.status].toLowerCase()}`);
      qc.invalidateQueries({ queryKey: ["admin-orders"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });
  const physical = o.format === "physical";

  return (
    <li className="border border-brand/10 bg-white/40 p-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="eyebrow text-brand/70">
            {orderStatusLabel[o.status]} · {physical ? "Printed copy" : "Digital"} · #
            {o.id.slice(0, 8)}
          </p>
          <p className="mt-1 font-serif text-lg">{o.book_title}</p>
          <p className="text-sm text-brand/65">
            {o.student_name}
            {o.student_email ? ` · ${o.student_email}` : ""}
          </p>
          {physical && (
            <p className="mt-1 text-sm text-brand/65">
              Ship to: {formatAddress(o.shipping_address)}
            </p>
          )}
          {o.tracking_number && (
            <p className="text-xs text-brand/70">Tracking: {o.tracking_number}</p>
          )}
        </div>
        <div className="text-right">
          <p className="font-serif text-lg">
            {o.amount === 0 ? "Free" : formatPrice(Math.round(o.amount * 100), o.currency)}
          </p>
          <p className="text-xs text-brand/65">{new Date(o.created_at).toLocaleString()}</p>
        </div>
      </div>

      {physical && o.status === "placed" && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            mut.mutate({ status: "shipped", tracking: tracking.trim() });
          }}
          className="mt-4 flex flex-wrap items-end gap-2"
        >
          <label className="block">
            <span className="eyebrow text-brand/65">Tracking number</span>
            <input
              value={tracking}
              onChange={(e) => setTracking(e.target.value)}
              required
              maxLength={100}
              className="mt-1 block border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
          <button
            type="submit"
            disabled={mut.isPending || !tracking.trim()}
            className="bg-brand px-4 py-2 text-xs text-white disabled:opacity-50"
          >
            Mark shipped
          </button>
          <button
            type="button"
            disabled={mut.isPending}
            onClick={() => {
              if (window.confirm("Cancel this order? The copy goes back into stock.")) {
                mut.mutate({ status: "cancelled" });
              }
            }}
            className="border border-brand/15 px-4 py-2 text-xs disabled:opacity-50"
          >
            Cancel order
          </button>
        </form>
      )}
      {physical && o.status === "shipped" && (
        <button
          disabled={mut.isPending}
          onClick={() => mut.mutate({ status: "delivered" })}
          className="mt-4 bg-brand px-4 py-2 text-xs text-white disabled:opacity-50"
        >
          Mark delivered
        </button>
      )}
    </li>
  );
}

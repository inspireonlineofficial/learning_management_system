import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { ListTable } from "@/components/layout/data-page";
import { listAllOrders, type Order } from "@/lib/api/orders";

export const Route = createFileRoute("/_authenticated/admin/bookshop/orders")({
  component: Page,
});

const STATUSES: Array<Order["status"] | "all"> = [
  "all",
  "pending",
  "paid",
  "shipped",
  "delivered",
  "refunded",
  "cancelled",
];

function Page() {
  const [status, setStatus] = useState<Order["status"] | "all">("all");
  const [q, setQ] = useState("");
  const { data, isLoading } = useQuery({
    queryKey: ["admin-orders", status, q],
    queryFn: () =>
      listAllOrders({
        status: status === "all" ? undefined : status,
        q: q.trim() || undefined,
      }),
  });

  return (
    <AppShell eyebrow="Orders" title="All orders">
      <Link
        to="/admin/bookshop"
        className="inline-block text-xs text-brand/55 hover:text-brand mb-4"
      >
        ← Bookshop admin
      </Link>

      <div className="flex flex-wrap gap-3 mb-6">
        <div className="flex gap-1 border border-brand/15 bg-white/40">
          {STATUSES.map((s) => (
            <button
              key={s}
              onClick={() => setStatus(s)}
              className={`px-3 py-2 text-xs capitalize ${
                status === s ? "bg-brand text-white" : "text-brand/65 hover:bg-brand/[0.03]"
              }`}
            >
              {s}
            </button>
          ))}
        </div>
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search by order id or customer…"
          className="flex-1 min-w-[220px] px-3 py-2 text-sm border border-brand/15 bg-white"
        />
      </div>

      {isLoading && <div className="h-40 border border-brand/10 bg-white/30 animate-pulse" />}
      {!isLoading && (data?.items?.length ?? 0) === 0 && <EmptyState title="No orders" />}
      {!isLoading && data && data.items.length > 0 && (
        <ListTable<Order>
          rows={data.items}
          columns={[
            { key: "id", label: "Order", render: (r) => r.id.slice(0, 8) },
            {
              key: "status",
              label: "Status",
              render: (r) => <span className="eyebrow text-brand/55">{r.status}</span>,
            },
            { key: "total", label: "Total", render: (r) => `${r.currency} ${r.total.toFixed(2)}` },
            {
              key: "when",
              label: "Placed",
              render: (r) => new Date(r.created_at).toLocaleDateString(),
            },
          ]}
        />
      )}
    </AppShell>
  );
}

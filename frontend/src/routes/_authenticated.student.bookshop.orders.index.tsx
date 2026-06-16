import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listMyOrders, type Order } from "@/lib/api/orders";

export const Route = createFileRoute("/_authenticated/student/bookshop/orders/")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Orders"
      title="Your orders"
      queryKey={["my-orders"]}
      queryFn={listMyOrders}
      empty={{ title: "No orders yet" }}
    >
      {(data) => (
        <ListTable<Order>
          rows={data.items}
          columns={[
            {
              key: "id",
              label: "Order",
              render: (r) => (
                <Link
                  to="/student/bookshop/orders/$orderId"
                  params={{ orderId: r.id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {r.id.slice(0, 8)}
                </Link>
              ),
            },
            {
              key: "status",
              label: "Status",
              render: (r) => <span className="eyebrow text-brand/55">{r.status}</span>,
            },
            { key: "items", label: "Items", render: (r) => r.items.length },
            { key: "total", label: "Total", render: (r) => `${r.currency} ${r.total.toFixed(2)}` },
            {
              key: "when",
              label: "Placed",
              render: (r) => new Date(r.created_at).toLocaleDateString(),
            },
          ]}
        />
      )}
    </DataPage>
  );
}

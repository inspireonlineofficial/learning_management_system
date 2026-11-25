import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { formatPrice } from "@/lib/api/bookshop";
import { listMyOrders, orderStatusLabel, type Order } from "@/lib/api/orders";

export const Route = createFileRoute("/_authenticated/student/bookshop/orders/")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Orders"
      title="Your orders"
      queryKey={["my-orders"]}
      queryFn={() => listMyOrders({ limit: 100 })}
      empty={{ title: "No orders yet" }}
    >
      {(data) => (
        <ListTable<Order>
          rows={data.items}
          columns={[
            {
              key: "book",
              label: "Book",
              render: (r) => (
                <Link
                  to="/student/bookshop/orders/$orderId"
                  params={{ orderId: r.id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {r.book_title || `Order ${r.id.slice(0, 8)}`}
                </Link>
              ),
            },
            {
              key: "edition",
              label: "Edition",
              render: (r) => (r.format === "physical" ? "Printed copy" : "Digital"),
            },
            {
              key: "status",
              label: "Status",
              render: (r) => (
                <span className="eyebrow text-brand/55">
                  {orderStatusLabel[r.status] ?? r.status}
                </span>
              ),
            },
            {
              key: "total",
              label: "Price",
              render: (r) =>
                r.amount === 0 ? "Free" : formatPrice(Math.round(r.amount * 100), r.currency),
            },
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

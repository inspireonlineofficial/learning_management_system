import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listMyBookRequests, type BookPurchaseRequest } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/student/bookshop/requests")({
  component: Page,
});

const STATUS: Record<BookPurchaseRequest["status"], string> = {
  pending: "Waiting for approval",
  approved: "Approved",
  rejected: "Declined",
};

function Page() {
  return (
    <DataPage
      eyebrow="Requests"
      title="Book purchase requests"
      queryKey={["my-book-requests"]}
      queryFn={() => listMyBookRequests({ limit: 100 })}
      empty={{ title: "No purchase requests yet" }}
    >
      {(data) => (
        <ListTable<BookPurchaseRequest>
          rows={data.data}
          columns={[
            {
              key: "book",
              label: "Book",
              render: (r) => (
                <Link
                  to="/student/bookshop/$bookId"
                  params={{ bookId: r.item_id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {r.item_title}
                </Link>
              ),
            },
            {
              key: "edition",
              label: "Edition",
              render: (r) => (r.book_format === "physical" ? "Printed copy" : "Digital"),
            },
            {
              key: "status",
              label: "Status",
              render: (r) => (
                <span>
                  <span className="eyebrow text-brand/55">{STATUS[r.status] ?? r.status}</span>
                  {r.status === "rejected" && r.rejection_reason && (
                    <span className="block text-xs text-brand/55">{r.rejection_reason}</span>
                  )}
                </span>
              ),
            },
            {
              key: "next",
              label: "",
              render: (r) =>
                r.status === "approved" ? (
                  r.book_format === "physical" && r.result_order_id ? (
                    <Link
                      to="/student/bookshop/orders/$orderId"
                      params={{ orderId: r.result_order_id }}
                      className="text-xs underline"
                    >
                      Track order
                    </Link>
                  ) : (
                    <Link
                      to="/student/bookshop/reader/$bookId"
                      params={{ bookId: r.item_id }}
                      className="text-xs underline"
                    >
                      Read now
                    </Link>
                  )
                ) : null,
            },
            {
              key: "when",
              label: "Sent",
              render: (r) => new Date(r.created_at).toLocaleDateString(),
            },
          ]}
        />
      )}
    </DataPage>
  );
}

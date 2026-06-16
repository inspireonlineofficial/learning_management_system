import { createFileRoute } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listMyRequests, type ApprovalItem } from "@/lib/api/approvals";

export const Route = createFileRoute("/_authenticated/student/bookshop/requests")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Requests"
      title="Purchase requests"
      queryKey={["my-requests"]}
      queryFn={listMyRequests}
      empty={{ title: "No requests yet" }}
    >
      {(data) => (
        <ListTable<ApprovalItem>
          rows={data.items}
          columns={[
            {
              key: "kind",
              label: "Type",
              render: (r) => <span className="eyebrow text-brand/55">{r.kind}</span>,
            },
            { key: "payload", label: "Details", render: (r) => r.payload_summary },
            {
              key: "when",
              label: "Submitted",
              render: (r) => new Date(r.created_at).toLocaleDateString(),
            },
          ]}
        />
      )}
    </DataPage>
  );
}

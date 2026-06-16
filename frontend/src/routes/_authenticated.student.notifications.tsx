import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Bell } from "lucide-react";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listNotifications, markAllRead, type Notification } from "@/lib/api/notifications";

export const Route = createFileRoute("/_authenticated/student/notifications")({
  component: Page,
});

function Page() {
  const qc = useQueryClient();
  const mut = useMutation({
    mutationFn: () => markAllRead(),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications"] }),
  });

  return (
    <DataPage
      eyebrow="Notifications"
      title="Notifications"
      queryKey={["notifications"]}
      queryFn={listNotifications}
      empty={{ title: "All caught up", description: "You have no notifications." }}
      toolbar={
        <button
          onClick={() => mut.mutate()}
          className="text-xs border border-brand/15 px-3 py-2 hover:bg-brand/[0.03]"
        >
          Mark all read
        </button>
      }
    >
      {(data) => (
        <ListTable<Notification>
          rows={data.items}
          columns={[
            {
              key: "title",
              label: "Title",
              render: (r) => (
                <div className={r.read_at ? "text-brand/60" : "font-medium"}>
                  <Bell className="inline h-3.5 w-3.5 mr-2 text-accent" />
                  {r.title}
                  {r.body && <p className="mt-1 text-xs text-brand/55">{r.body}</p>}
                </div>
              ),
            },
            {
              key: "type",
              label: "Type",
              render: (r) => <span className="eyebrow text-brand/45">{r.type}</span>,
            },
            {
              key: "when",
              label: "When",
              render: (r) => new Date(r.created_at).toLocaleString(),
              width: "180px",
            },
          ]}
        />
      )}
    </DataPage>
  );
}

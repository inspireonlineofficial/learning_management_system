import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bell } from "lucide-react";
import { toast } from "sonner";

import { EmptyState } from "@/components/layout/app-shell";
import { ListTable } from "@/components/layout/data-page";
import { listNotifications, markAllRead, type Notification } from "@/lib/api/notifications";

export function NotificationInbox({ className = "" }: { className?: string }) {
  const qc = useQueryClient();
  const query = useQuery({ queryKey: ["notifications"], queryFn: listNotifications });
  const markAll = useMutation({
    mutationFn: () => markAllRead(),
    onSuccess: () => {
      toast.success("Notifications marked as read");
      qc.invalidateQueries({ queryKey: ["notifications"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <div className={className}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="eyebrow text-brand/45">Inbox</p>
          <h2 className="font-serif text-2xl">Your notifications</h2>
        </div>
        <button
          onClick={() => markAll.mutate()}
          disabled={markAll.isPending || query.isLoading || (query.data?.unread_count ?? 0) === 0}
          className="self-start text-xs border border-brand/15 px-3 py-2 hover:bg-brand/[0.03] disabled:opacity-50"
        >
          {markAll.isPending ? "Marking..." : "Mark all read"}
        </button>
      </div>

      {query.isLoading ? (
        <div className="mt-6 grid gap-3">
          {Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="h-20 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : query.isError ? (
        <div className="mt-6 border border-destructive/20 bg-destructive/5 p-6 text-sm">
          <p className="font-medium text-destructive">Couldn't load notifications</p>
          <p className="mt-1 text-brand/60">{(query.error as Error).message}</p>
          <button
            onClick={() => query.refetch()}
            className="mt-3 px-4 py-2 bg-brand text-white text-xs"
          >
            Try again
          </button>
        </div>
      ) : !query.data?.items.length ? (
        <div className="mt-6">
          <EmptyState title="All caught up" description="You have no notifications." />
        </div>
      ) : (
        <div className="mt-6">
          <ListTable<Notification>
            rows={query.data.items}
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
        </div>
      )}
    </div>
  );
}

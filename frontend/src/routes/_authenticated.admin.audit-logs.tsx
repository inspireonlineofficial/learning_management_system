import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listAuditLogs, type AuditLog } from "@/lib/api/audit";

export const Route = createFileRoute("/_authenticated/admin/audit-logs")({
  component: Page,
});

function Page() {
  const [actor, setActor] = useState("");
  const [action, setAction] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  // Server query (broad). Client filtering refines further so changes feel instant.
  const { data, isLoading } = useQuery({
    queryKey: ["audit-logs"],
    queryFn: () => listAuditLogs(),
  });

  const actions = useMemo(() => {
    const set = new Set<string>();
    data?.items.forEach((it) => set.add(it.action));
    return Array.from(set).sort();
  }, [data]);

  const filtered = useMemo(() => {
    if (!data) return [] as AuditLog[];
    const a = actor.trim().toLowerCase();
    const fromTs = from ? new Date(from).getTime() : null;
    const toTs = to ? new Date(to).getTime() + 86_400_000 : null;
    return data.items.filter((it) => {
      if (a && !it.actor.full_name.toLowerCase().includes(a)) return false;
      if (action && it.action !== action) return false;
      const ts = new Date(it.created_at).getTime();
      if (fromTs != null && ts < fromTs) return false;
      if (toTs != null && ts >= toTs) return false;
      return true;
    });
  }, [data, actor, action, from, to]);

  const clear = () => {
    setActor("");
    setAction("");
    setFrom("");
    setTo("");
  };

  return (
    <DataPage
      eyebrow="Audit"
      title="Audit logs"
      queryKey={["audit-logs-shell"]}
      queryFn={async () => ({ items: [{}] })}
      toolbar={
        <div className="grid sm:grid-cols-2 lg:grid-cols-5 gap-2 max-w-4xl">
          <input
            value={actor}
            onChange={(e) => setActor(e.target.value)}
            placeholder="Actor name"
            className="px-3 py-2 text-sm bg-white border border-brand/15 focus:border-brand/40 focus:outline-none"
          />
          <select
            value={action}
            onChange={(e) => setAction(e.target.value)}
            className="px-3 py-2 text-sm bg-white border border-brand/15 focus:border-brand/40 focus:outline-none"
          >
            <option value="">All actions</option>
            {actions.map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </select>
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="px-3 py-2 text-sm bg-white border border-brand/15 focus:border-brand/40 focus:outline-none"
          />
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="px-3 py-2 text-sm bg-white border border-brand/15 focus:border-brand/40 focus:outline-none"
          />
          <button
            onClick={clear}
            className="px-3 py-2 text-xs border border-brand/15 text-brand/65 hover:text-brand hover:bg-brand/[0.03]"
          >
            Reset filters
          </button>
        </div>
      }
    >
      {() => (
        <>
          <p className="text-xs text-brand/55 mb-3">
            {isLoading ? "Loading…" : `${filtered.length} of ${data?.items.length ?? 0} events`}
          </p>
          {filtered.length === 0 ? (
            <p className="text-sm text-brand/55">No events match these filters.</p>
          ) : (
            <ListTable<AuditLog>
              rows={filtered}
              columns={[
                { key: "actor", label: "Actor", render: (r) => r.actor.full_name },
                {
                  key: "action",
                  label: "Action",
                  render: (r) => <span className="eyebrow text-brand/60">{r.action}</span>,
                },
                { key: "target", label: "Target", render: (r) => r.target ?? "—" },
                { key: "ip", label: "IP", render: (r) => r.ip ?? "—" },
                {
                  key: "when",
                  label: "When",
                  render: (r) => new Date(r.created_at).toLocaleString(),
                },
              ]}
            />
          )}
        </>
      )}
    </DataPage>
  );
}

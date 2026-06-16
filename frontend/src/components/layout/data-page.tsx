import { useQuery, type QueryKey } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { AppShell, EmptyState, SectionHeading } from "@/components/layout/app-shell";

export function DataPage<T>({
  eyebrow,
  title,
  description,
  queryKey,
  queryFn,
  empty,
  children,
  toolbar,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  queryKey: QueryKey;
  queryFn: () => Promise<T>;
  empty?: { title: string; description?: string; action?: ReactNode };
  children: (data: T) => ReactNode;
  toolbar?: ReactNode;
}) {
  const { data, isLoading, isError, error, refetch } = useQuery({ queryKey, queryFn });

  const maybeItems = (data as unknown as { items?: unknown[] } | null)?.items;
  const isEmpty = Array.isArray(maybeItems) && maybeItems.length === 0;

  return (
    <AppShell eyebrow={eyebrow} title={title}>
      {description && <p className="max-w-2xl text-brand/65 leading-relaxed">{description}</p>}
      {toolbar && <div className="mt-6">{toolbar}</div>}

      {isLoading && (
        <div className="mt-10 grid sm:grid-cols-2 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-24 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      )}

      {isError && (
        <div className="mt-10 border border-destructive/20 bg-destructive/5 p-6 text-sm">
          <p className="font-medium text-destructive">Couldn't load data</p>
          <p className="mt-1 text-brand/60">{(error as Error)?.message}</p>
          <button onClick={() => refetch()} className="mt-3 px-4 py-2 bg-brand text-white text-xs">
            Try again
          </button>
        </div>
      )}

      {data != null && !isLoading && !isError && (
        <div className="mt-10">
          {isEmpty && empty ? (
            <EmptyState title={empty.title} description={empty.description} action={empty.action} />
          ) : (
            children(data)
          )}
        </div>
      )}
    </AppShell>
  );
}

export function ListTable<T extends { id: string }>({
  rows,
  columns,
  onRowHref,
}: {
  rows: T[];
  columns: Array<{ key: string; label: string; render: (row: T) => ReactNode; width?: string }>;
  onRowHref?: (row: T) => string;
}) {
  return (
    <div className="border border-brand/10 bg-white/40">
      <table className="w-full text-sm">
        <thead className="border-b border-brand/10 bg-brand/[0.02]">
          <tr>
            {columns.map((c) => (
              <th
                key={c.key}
                className="text-left px-4 py-3 font-medium text-brand/60 eyebrow"
                style={c.width ? { width: c.width } : undefined}
              >
                {c.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.id}
              className="border-b border-brand/5 last:border-b-0 hover:bg-brand/[0.02]"
            >
              {columns.map((c) => (
                <td key={c.key} className="px-4 py-3 align-top">
                  {onRowHref && c.key === columns[0].key ? (
                    <a href={onRowHref(row)} className="font-medium text-brand hover:text-accent">
                      {c.render(row)}
                    </a>
                  ) : (
                    c.render(row)
                  )}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function DetailGrid({ items }: { items: Array<{ label: string; value: ReactNode }> }) {
  return (
    <dl className="grid sm:grid-cols-2 gap-x-10 gap-y-4 border border-brand/10 bg-white/40 p-6">
      {items.map((it) => (
        <div key={it.label}>
          <dt className="eyebrow text-brand/45">{it.label}</dt>
          <dd className="mt-1 text-brand">{it.value}</dd>
        </div>
      ))}
    </dl>
  );
}

export function Section({
  title,
  children,
  action,
}: {
  title: string;
  children: ReactNode;
  action?: ReactNode;
}) {
  return (
    <section className="mt-14">
      <SectionHeading title={title} action={action} />
      {children}
    </section>
  );
}

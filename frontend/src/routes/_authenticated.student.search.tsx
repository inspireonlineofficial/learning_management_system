import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search as SearchIcon } from "lucide-react";

import { AppShell } from "@/components/layout/app-shell";
import { search } from "@/lib/api/search";

export const Route = createFileRoute("/_authenticated/student/search")({
  component: Page,
});

function Page() {
  const [q, setQ] = useState("");
  const { data, isFetching } = useQuery({
    queryKey: ["search", q],
    queryFn: () => search(q),
    enabled: q.length > 1,
  });

  return (
    <AppShell eyebrow="Search" title="Search the library">
      <div className="max-w-2xl flex items-center border border-brand/15 bg-white">
        <SearchIcon className="h-4 w-4 mx-3 text-brand/45" />
        <input
          autoFocus
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Courses, lessons, books, discussions…"
          className="flex-1 py-3 pr-3 text-sm bg-transparent outline-none"
        />
      </div>
      <div className="mt-8 max-w-2xl">
        {isFetching && <p className="text-sm text-brand/55">Searching…</p>}
        {data && data.items.length === 0 && q.length > 1 && (
          <p className="text-sm text-brand/55">No matches.</p>
        )}
        <ul className="space-y-2">
          {data?.items.map((r) => (
            <li key={`${r.type}-${r.id}`}>
              <a
                href={r.url}
                className="block border border-brand/10 bg-white/50 p-4 hover:bg-white"
              >
                <p className="eyebrow text-accent">{r.type}</p>
                <p className="mt-1 font-serif text-base">{r.title}</p>
                {r.snippet && <p className="mt-1 text-xs text-brand/55">{r.snippet}</p>}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </AppShell>
  );
}

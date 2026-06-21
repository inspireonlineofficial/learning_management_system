import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { BookOpen } from "lucide-react";

import { AppShell, EmptyState, SectionHeading } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { listMyLibrary } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/student/bookshop/library/")({
  component: LibraryPage,
});

function LibraryPage() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["library"],
    queryFn: () => listMyLibrary({ limit: 60 }),
  });

  return (
    <AppShell eyebrow="Bookshop" title="My library.">
      {isError ? (
        <QueryErrorPanel error={error} title="Couldn't load library" onRetry={() => refetch()} />
      ) : isLoading ? (
        <div className="grid sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="aspect-[2/3] bg-brand/5 animate-pulse" />
          ))}
        </div>
      ) : !data || data.data.length === 0 ? (
        <EmptyState
          icon={BookOpen}
          title="Your library is empty"
          description="Purchased and free books appear here."
          action={
            <Link
              to="/student/bookshop"
              className="px-5 py-2.5 bg-brand text-white text-xs font-medium"
            >
              Browse the bookshop
            </Link>
          }
        />
      ) : (
        <>
          <SectionHeading title={`${data.meta.total} book${data.meta.total === 1 ? "" : "s"}`} />
          <div className="grid sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-x-6 gap-y-10">
            {data.data.map((b) => (
              <Link
                key={b.id}
                to="/student/bookshop/reader/$bookId"
                params={{ bookId: b.id }}
                className="group block"
              >
                <div className="aspect-[2/3] bg-brand/10 mb-4 overflow-hidden">
                  {b.cover_url ? (
                    <img
                      src={b.cover_url}
                      alt={b.title}
                      className="w-full h-full object-cover group-hover:scale-[1.02] transition-transform"
                    />
                  ) : (
                    <div className="w-full h-full grid place-items-center text-brand/30">
                      <BookOpen className="h-10 w-10" />
                    </div>
                  )}
                </div>
                <p className="text-[11px] uppercase tracking-wider text-brand/45 mb-1">
                  {b.author}
                </p>
                <p className="font-serif text-base leading-snug group-hover:text-accent">
                  {b.title}
                </p>
              </Link>
            ))}
          </div>
        </>
      )}
    </AppShell>
  );
}

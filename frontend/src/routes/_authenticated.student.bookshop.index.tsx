import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { BookOpen, Search } from "lucide-react";
import { useState } from "react";

import { BookshopNav } from "@/components/bookshop/bookshop-nav";
import { EmptyState } from "@/components/layout/app-shell";
import { FilterChip, HeroPanel, PageHero, fieldClass } from "@/components/layout/page-hero";
import { PublicFooter } from "@/components/layout/public-footer";
import { PublicHeader } from "@/components/layout/public-header";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { formatPrice, listBooks, type BookSummary } from "@/lib/api/bookshop";

type Sort = "newest" | "price_asc" | "price_desc" | "title";

export const Route = createFileRoute("/_authenticated/student/bookshop/")({
  component: BookshopIndex,
});

function BookshopIndex() {
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<Sort>("newest");

  const books = useQuery({
    queryKey: ["bookshop-books", { search, sort }],
    queryFn: () => listBooks({ search, sort, limit: 48 }),
  });

  return (
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy-light font-sans antialiased flex flex-col">
      <PublicHeader active="bookshop" />

      <PageHero eyebrow="Student bookshop" title="Curated science study materials">
        <div className="mb-6">
          <BookshopNav />
        </div>
        <HeroPanel className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
          <div className="flex-1 min-w-0 relative">
            <Search
              className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-slatebody pointer-events-none"
              aria-hidden="true"
            />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by title or author…"
              aria-label="Search books"
              className={`${fieldClass} pl-10`}
            />
          </div>
          <SortSelect value={sort} onChange={setSort} />
        </HeroPanel>
      </PageHero>

      <main className="flex-1 page-container py-10 sm:py-12">
        {books.isError ? (
          <QueryErrorPanel
            error={books.error}
            title="Couldn't load books"
            onRetry={() => books.refetch()}
          />
        ) : books.isLoading ? (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 sm:gap-6">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="rounded-lg border border-borderui bg-white p-3 animate-pulse">
                <div className="aspect-[2/3] rounded-md bg-stoneband" />
                <div className="mt-3 h-3 w-3/4 rounded bg-stoneband" />
                <div className="mt-2 h-3 w-1/2 rounded bg-stoneband" />
              </div>
            ))}
          </div>
        ) : !books.data || books.data.data.length === 0 ? (
          <EmptyState
            icon={BookOpen}
            title={search ? "No books match your search" : "No books available yet"}
            description={
              search
                ? "Try a different title, author or subject."
                : "New study materials are added regularly. Check back soon."
            }
          />
        ) : (
          <>
            <p className="mb-5 text-sm text-slatebody">
              {books.data.meta.total} book{books.data.meta.total === 1 ? "" : "s"}
            </p>
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 sm:gap-6">
              {books.data.data.map((b) => (
                <BookCard key={b.id} book={b} />
              ))}
            </div>
          </>
        )}
      </main>

      <PublicFooter />
    </div>
  );
}

function BookCard({ book: b }: { book: BookSummary }) {
  return (
    <Link
      to="/student/bookshop/$bookId"
      params={{ bookId: b.id }}
      className="group flex flex-col rounded-lg overflow-hidden border border-borderui bg-white p-3 sm:p-4 hover:border-navy transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
    >
      <div className="aspect-[2/3] rounded-md bg-stoneband mb-3.5 overflow-hidden">
        {b.cover_url ? (
          <img
            src={b.cover_url}
            alt={b.title}
            loading="lazy"
            className="w-full h-full object-cover"
          />
        ) : (
          <div className="w-full h-full grid place-items-center text-navy/25">
            <BookOpen className="h-10 w-10" aria-hidden="true" />
          </div>
        )}
      </div>
      <p className="text-xs font-semibold uppercase tracking-wider text-bronze/85 mb-1 truncate">
        {b.author}
      </p>
      <p className="font-serif text-base sm:text-lg font-bold text-navy leading-snug line-clamp-2 group-hover:text-bronze transition-colors">
        {b.title}
      </p>
      <p className="mt-auto pt-3 text-sm font-bold text-navy">
        {b.in_library ? (
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-md bg-emerald-50 text-emerald-800 text-xs font-semibold">
            In your library
          </span>
        ) : b.is_free ? (
          <span className="text-emerald-700">Free</span>
        ) : (
          formatPrice(b.price_cents, b.currency)
        )}
      </p>
    </Link>
  );
}

function SortSelect({ value, onChange }: { value: Sort; onChange: (v: Sort) => void }) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value as Sort)}
      aria-label="Sort books"
      className={`${fieldClass} sm:w-auto font-medium`}
    >
      <option value="newest">Newest</option>
      <option value="title">Title A–Z</option>
      <option value="price_asc">Price: low to high</option>
      <option value="price_desc">Price: high to low</option>
    </select>
  );
}

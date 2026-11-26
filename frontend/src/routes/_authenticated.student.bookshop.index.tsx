import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { BookOpen, Search } from "lucide-react";
import { useState } from "react";

import { BookshopNav } from "@/components/bookshop/bookshop-nav";
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

      <section className="hero-atmosphere text-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16 lg:py-20">
          <p className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-white/10 border border-amber-300/35 text-amber-200 text-xs font-semibold tracking-[0.14em] uppercase mb-5">
            <span className="w-2 h-2 rounded-full bg-gold" aria-hidden="true" />
            Student Bookshop
          </p>
          <h1 className="font-serif text-3xl sm:text-4xl lg:text-5xl font-bold leading-[1.14] text-balance max-w-3xl mb-6">
            Curated science study materials.
          </h1>
          <div className="mb-7">
            <BookshopNav />
          </div>

          <div className="bg-white rounded-xl p-3 sm:p-4 shadow-xl border border-borderui max-w-3xl flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
            <div className="flex-1 min-w-0 relative">
              <Search
                className="absolute left-3.5 top-1/2 -translate-y-1/2 h-5 w-5 text-slatebody pointer-events-none"
                aria-hidden="true"
              />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by title or author…"
                className="w-full min-h-[46px] pl-11 pr-4 py-2.5 rounded-lg text-sm sm:text-base border border-borderui bg-canvas text-navy placeholder:text-slatebody focus:bg-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
              />
            </div>
            <select
              value={sort}
              onChange={(e) => setSort(e.target.value as Sort)}
              aria-label="Sort books"
              className="w-full sm:w-auto min-h-[46px] px-3.5 py-2.5 rounded-lg text-sm font-medium border border-borderui bg-canvas text-navy focus:bg-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
            >
              <option value="newest">Newest</option>
              <option value="title">Title A–Z</option>
              <option value="price_asc">Price: low to high</option>
              <option value="price_desc">Price: high to low</option>
            </select>
          </div>
        </div>
      </section>

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-10 sm:py-12">
        {books.isError ? (
          <QueryErrorPanel
            error={books.error}
            title="Couldn't load books"
            onRetry={() => books.refetch()}
          />
        ) : books.isLoading ? (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 sm:gap-6">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="rounded-xl border border-borderui bg-white p-3 animate-pulse">
                <div className="aspect-[2/3] rounded-lg bg-stoneband" />
                <div className="mt-3 h-3 w-3/4 rounded bg-stoneband" />
                <div className="mt-2 h-3 w-1/2 rounded bg-stoneband" />
              </div>
            ))}
          </div>
        ) : !books.data || books.data.data.length === 0 ? (
          <div className="rounded-xl border border-dashed border-borderui bg-white px-6 py-16 text-center">
            <span className="inline-flex h-12 w-12 items-center justify-center rounded-xl bg-stoneband text-navy">
              <BookOpen className="h-6 w-6" aria-hidden="true" />
            </span>
            <h3 className="mt-4 font-serif text-2xl font-bold text-navy">No books found</h3>
            <p className="mt-2 text-sm text-slatebody">Try a different search term.</p>
          </div>
        ) : (
          <>
            <p className="text-xs font-bold tracking-[0.14em] uppercase text-bronze mb-6">
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
      className="group flex flex-col rounded-xl overflow-hidden border border-borderui bg-white p-3 sm:p-4 shadow-sm hover:border-navy hover:-translate-y-0.5 hover:shadow-md transition-all focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
    >
      <div className="aspect-[2/3] rounded-lg bg-stoneband mb-3.5 overflow-hidden">
        {b.cover_url ? (
          <img
            src={b.cover_url}
            alt={b.title}
            loading="lazy"
            className="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-300"
          />
        ) : (
          <div className="w-full h-full grid place-items-center text-navy/25">
            <BookOpen className="h-10 w-10" aria-hidden="true" />
          </div>
        )}
      </div>
      <p className="text-[11px] font-semibold uppercase tracking-wider text-bronze/85 mb-1 truncate">
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

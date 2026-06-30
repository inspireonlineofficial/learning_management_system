import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { BookOpen, Search } from "lucide-react";
import { useState } from "react";

import { PublicHeader } from "@/components/layout/public-header";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { formatPrice, listBooks, type BookSummary } from "@/lib/api/bookshop";

type Sort = "popular" | "newest" | "price_asc" | "price_desc";

export const Route = createFileRoute("/_authenticated/student/bookshop/")({
  component: BookshopIndex,
});

function BookshopIndex() {
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<Sort>("popular");

  const books = useQuery({
    queryKey: ["bookshop-books", { search, sort }],
    queryFn: () => listBooks({ search, sort, limit: 48 }),
  });

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      <PublicHeader active="bookshop" />

      <section className="px-6 lg:px-10 py-12 lg:py-20 border-b border-brand/10">
        <div className="max-w-7xl mx-auto">
          <p className="eyebrow text-accent mb-4">The bookshop</p>
          <h1 className="font-serif text-4xl lg:text-6xl text-balance max-w-3xl mb-8">
            Curated science study materials.
          </h1>

          <div className="flex flex-wrap items-center gap-3 max-w-3xl">
            <div className="flex-1 min-w-[240px] relative">
              <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-brand/40" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by title or author…"
                className="w-full pl-11 pr-4 py-3 text-sm border border-brand/15 bg-white/50 focus:outline-none focus:border-brand/40"
              />
            </div>
            <select
              value={sort}
              onChange={(e) => setSort(e.target.value as Sort)}
              className="px-4 py-3 text-sm border border-brand/15 bg-white/50"
            >
              <option value="popular">Most popular</option>
              <option value="newest">Newest</option>
              <option value="price_asc">Price: low to high</option>
              <option value="price_desc">Price: high to low</option>
            </select>
          </div>
        </div>
      </section>

      <main className="max-w-7xl mx-auto px-6 lg:px-10 py-12">
        {books.isError ? (
          <QueryErrorPanel
            error={books.error}
            title="Couldn't load books"
            onRetry={() => books.refetch()}
          />
        ) : books.isLoading ? (
          <div className="grid sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="aspect-[2/3] bg-brand/5 animate-pulse" />
            ))}
          </div>
        ) : !books.data || books.data.data.length === 0 ? (
          <div className="border border-dashed border-brand/15 px-8 py-16 text-center">
            <BookOpen className="h-8 w-8 mx-auto text-brand/30" />
            <h3 className="mt-4 font-serif text-2xl">No books found</h3>
            <p className="mt-2 text-sm text-brand/55">Try a different search term.</p>
          </div>
        ) : (
          <>
            <p className="eyebrow text-brand/45 mb-6">
              {books.data.meta.total} book{books.data.meta.total === 1 ? "" : "s"}
            </p>
            <div className="grid sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-x-6 gap-y-10">
              {books.data.data.map((b) => (
                <BookCard key={b.id} book={b} />
              ))}
            </div>
          </>
        )}
      </main>
    </div>
  );
}

function BookCard({ book: b }: { book: BookSummary }) {
  return (
    <Link to="/student/bookshop/$bookId" params={{ bookId: b.id }} className="group block">
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
      <p className="text-[11px] uppercase tracking-wider text-brand/45 mb-1">{b.author}</p>
      <p className="font-serif text-base leading-snug group-hover:text-accent transition-colors">
        {b.title}
      </p>
      <p className="mt-2 text-sm text-brand">
        {b.in_library ? (
          <span className="text-emerald-700 text-xs">In your library</span>
        ) : b.is_free ? (
          "Free"
        ) : (
          formatPrice(b.price_cents, b.currency)
        )}
      </p>
    </Link>
  );
}

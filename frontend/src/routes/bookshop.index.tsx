import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { BookOpen, Search, Star } from "lucide-react";
import { useState } from "react";
import { z } from "zod";

import { PublicFooter } from "@/components/layout/public-footer";
import { PublicHeader } from "@/components/layout/public-header";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { formatPrice, listBooks, type BookSummary } from "@/lib/api/bookshop";
import { DEFAULT_HOMEPAGE_CONFIG, getPublicHomepage } from "@/lib/api/homepage";

type Sort = "newest" | "price_asc" | "price_desc" | "title";

const searchSchema = z.object({
  q: z.string().optional(),
  category: z.string().optional(),
});

export const Route = createFileRoute("/bookshop/")({
  validateSearch: searchSchema,
  component: PublicBookshopIndex,
});

function PublicBookshopIndex() {
  const routeSearch = Route.useSearch();
  const [search, setSearch] = useState(routeSearch.q ?? "");
  const [category, setCategory] = useState(routeSearch.category ?? "");
  const [sort, setSort] = useState<Sort>("newest");

  const { data: homepageData } = useQuery({
    queryKey: ["homepage", "public"],
    queryFn: getPublicHomepage,
    retry: false,
    staleTime: 5 * 60_000,
  });
  const subjects =
    homepageData?.homepage_config?.bookshop_subjects ?? DEFAULT_HOMEPAGE_CONFIG.bookshop_subjects;

  const books = useQuery({
    queryKey: ["public-bookshop-books", { search, category, sort }],
    queryFn: () => listBooks({ search, category, sort, limit: 48 }),
  });

  return (
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy-light font-sans antialiased flex flex-col">
      <PublicHeader active="bookshop" />

      <section className="hero-atmosphere text-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16 lg:py-20">
          <p className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-white/10 border border-amber-300/35 text-amber-200 text-xs font-semibold tracking-[0.14em] uppercase mb-5">
            <span className="w-2 h-2 rounded-full bg-gold" aria-hidden="true" />
            Inspire Bookshop
          </p>
          <h1 className="font-serif text-3xl sm:text-4xl lg:text-5xl font-bold leading-[1.14] text-balance max-w-3xl">
            Expand your learning with curated reading materials.
          </h1>
          <p className="mt-4 max-w-xl text-slate-200 text-sm sm:text-base leading-relaxed">
            Access textbooks, workbooks, solution guides, and lab manuals aligned with your academic
            syllabus.
          </p>

          <div className="mt-8 bg-white rounded-xl p-3 sm:p-4 shadow-xl border border-borderui max-w-3xl space-y-3">
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
              <div className="flex-1 min-w-0 relative">
                <Search
                  className="absolute left-3.5 top-1/2 -translate-y-1/2 h-5 w-5 text-slatebody pointer-events-none"
                  aria-hidden="true"
                />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search by title, author, category..."
                  className="w-full min-h-[46px] pl-11 pr-4 py-2.5 rounded-lg text-sm sm:text-base border border-borderui bg-canvas text-navy placeholder:text-slatebody focus:bg-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy transition-colors"
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
            <div className="flex flex-wrap gap-2 pt-1">
              <FilterChip active={!category} onClick={() => setCategory("")}>
                All subjects
              </FilterChip>
              {subjects.map((subject) => (
                <FilterChip
                  key={subject}
                  active={category === subject}
                  onClick={() => setCategory(subject)}
                >
                  {subject}
                </FilterChip>
              ))}
            </div>
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
            <p className="mt-2 text-sm text-slatebody">Try a different search term or subject.</p>
          </div>
        ) : (
          <>
            <p className="text-xs font-bold tracking-[0.14em] uppercase text-bronze mb-6">
              Available Books ({books.data.meta.total})
            </p>
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 sm:gap-6">
              {books.data.data.map((b) => (
                <PublicBookCard key={b.id} book={b} />
              ))}
            </div>
          </>
        )}
      </main>

      <PublicFooter />
    </div>
  );
}

function PublicBookCard({ book: b }: { book: BookSummary }) {
  const stockLabel =
    b.format === "digital"
      ? "Digital access"
      : b.physical_stock && b.physical_stock > 0
        ? `${b.physical_stock} in stock`
        : "Request availability";

  return (
    <Link
      to="/bookshop/$bookId"
      params={{ bookId: b.id }}
      className="group flex flex-col rounded-xl overflow-hidden border border-borderui bg-white p-3 sm:p-4 shadow-sm hover:border-navy hover:-translate-y-0.5 hover:shadow-md transition-all focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
    >
      <div className="aspect-[2/3] rounded-lg bg-stoneband mb-3.5 overflow-hidden relative">
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
      <div className="mt-2.5 flex items-center justify-between gap-2 text-xs text-slatebody">
        <span className="truncate">{b.category ?? "General science"}</span>
        {typeof b.rating === "number" && (
          <span className="inline-flex items-center gap-1 shrink-0 font-medium text-navy">
            <Star className="h-3.5 w-3.5 fill-gold text-gold" aria-hidden="true" />
            {b.rating.toFixed(1)}
          </span>
        )}
      </div>
      <div className="mt-auto pt-3 border-t border-borderui/60 mt-3 flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm font-bold text-navy">
          {b.is_free ? (
            <span className="text-emerald-700">Free</span>
          ) : (
            formatPrice(b.price_cents, b.currency)
          )}
        </p>
        <span className="text-[11px] font-medium text-slatebody">{stockLabel}</span>
      </div>
      <span className="mt-3 inline-flex text-xs font-semibold text-navy underline underline-offset-4 decoration-gold decoration-2 group-hover:text-bronze">
        {b.in_library ? "Read now" : "View details"} &rarr;
      </span>
    </Link>
  );
}

function FilterChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`inline-flex items-center justify-center min-h-[38px] px-3.5 py-1.5 rounded-lg border text-xs sm:text-sm font-semibold transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy ${
        active
          ? "border-navy bg-navy text-white"
          : "border-borderui bg-canvas text-navy-light hover:border-navy hover:bg-white"
      }`}
    >
      {children}
    </button>
  );
}

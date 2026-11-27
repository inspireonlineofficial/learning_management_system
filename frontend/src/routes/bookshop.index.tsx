import { usePublicSiteConfig } from "@/hooks/use-site-config";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { BookOpen, Search, Star } from "lucide-react";
import { useState } from "react";
import { z } from "zod";

import { EmptyState } from "@/components/layout/app-shell";
import { FilterChip, HeroPanel, PageHero, fieldClass } from "@/components/layout/page-hero";
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
  const page = usePublicSiteConfig().pages.bookshop;
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

      <PageHero eyebrow={page.eyebrow} title={page.title} body={page.body}>
        <HeroPanel className="space-y-3">
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
            <div className="flex-1 min-w-0 relative">
              <Search
                className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-slatebody pointer-events-none"
                aria-hidden="true"
              />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by title, author or subject…"
                aria-label="Search books"
                className={`${fieldClass} pl-10`}
              />
            </div>
            <SortSelect value={sort} onChange={setSort} />
          </div>
          <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by subject">
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
            title={search || category ? "No books match your search" : "No books available yet"}
            description={
              search || category
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
      className="group flex flex-col rounded-lg overflow-hidden border border-borderui bg-white p-3 sm:p-4 hover:border-navy transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
    >
      <div className="aspect-[2/3] rounded-md bg-stoneband mb-3.5 overflow-hidden relative">
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
        <span className="text-xs font-medium text-slatebody">{stockLabel}</span>
      </div>
      <span className="mt-3 inline-flex text-xs font-semibold text-navy underline underline-offset-4 decoration-gold decoration-2 group-hover:text-bronze">
        {b.in_library ? "Read now" : "View details"} &rarr;
      </span>
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

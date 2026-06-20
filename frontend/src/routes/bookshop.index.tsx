import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { BookOpen, Search, Star } from "lucide-react";
import { useState } from "react";
import { z } from "zod";

import { BrandLogo } from "@/components/layout/brand-logo";
import { useAuth } from "@/context/auth-context";
import { formatPrice, listBooks, type BookSummary } from "@/lib/api/bookshop";

type Sort = "popular" | "newest" | "price_asc" | "price_desc";

const searchSchema = z.object({
  q: z.string().optional(),
  category: z.string().optional(),
});

const subjects = ["Physics", "Chemistry", "Biology", "Mathematics", "ICT", "Exam Prep"];

export const Route = createFileRoute("/bookshop/")({
  validateSearch: searchSchema,
  component: PublicBookshopIndex,
});

function PublicBookshopIndex() {
  const { isAuthenticated, user } = useAuth();
  const routeSearch = Route.useSearch();
  const [search, setSearch] = useState(routeSearch.q ?? "");
  const [category, setCategory] = useState(routeSearch.category ?? "");
  const [sort, setSort] = useState<Sort>("popular");

  const books = useQuery({
    queryKey: ["public-bookshop-books", { search, category, sort }],
    queryFn: () => listBooks({ search, category, sort, limit: 48 }),
  });

  return (
    <div className="min-h-screen overflow-x-hidden bg-surface text-brand font-sans">
      <header className="border-b border-brand/10">
        <div className="max-w-7xl mx-auto px-6 lg:px-10 py-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <BrandLogo imageClassName="max-h-14 max-w-[220px]" />
          <nav className="flex w-full flex-wrap items-center gap-x-4 gap-y-2 text-sm sm:w-auto">
            <Link to="/courses" className="text-brand/70 hover:text-brand transition-colors">
              Courses
            </Link>
            <Link to="/bookshop" className="font-medium text-brand">
              Bookshop
            </Link>
            {isAuthenticated ? (
              <Link
                to={
                  user?.role === "admin"
                    ? "/admin"
                    : user?.role === "teacher"
                      ? "/teacher"
                      : "/student"
                }
                className="px-4 py-2 bg-brand text-white text-xs font-medium hover:bg-brand/90 transition-colors"
              >
                Go to Dashboard
              </Link>
            ) : (
              <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
                <Link to="/login" className="text-brand/70 hover:text-brand transition-colors">
                  Sign in
                </Link>
                <Link
                  to="/register"
                  className="px-4 py-2 bg-brand text-white text-xs font-medium hover:bg-brand/90 transition-colors"
                >
                  Register
                </Link>
              </div>
            )}
          </nav>
        </div>
      </header>

      <section className="px-6 lg:px-10 py-12 lg:py-20 border-b border-brand/10 bg-gradient-to-br from-brand/[0.02] to-accent/[0.02]">
        <div className="max-w-7xl mx-auto">
          <p className="eyebrow text-accent mb-4">Inspire Bookstore</p>
          <h1 className="font-serif text-3xl leading-[1.1] text-balance max-w-3xl mb-8 break-words sm:text-4xl lg:text-6xl">
            Expand your learning with curated reading materials.
          </h1>
          <p className="max-w-xl text-brand/65 text-sm lg:text-base mb-8">
            Access text-books, supplementary reading guides, and reference material linked with your
            academic syllabus.
          </p>

          <div className="flex flex-col sm:flex-row sm:flex-wrap items-stretch sm:items-center gap-3 max-w-3xl">
            <div className="flex-1 min-w-0 relative">
              <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-brand/40" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by title, author, category..."
                className="w-full pl-11 pr-4 py-3 text-sm border border-brand/15 bg-white/50 focus:outline-none focus:border-brand/40 focus:bg-white transition-all"
              />
            </div>
            <select
              value={sort}
              onChange={(e) => setSort(e.target.value as Sort)}
              className="w-full sm:w-auto px-4 py-3 text-sm border border-brand/15 bg-white/50 focus:outline-none focus:border-brand/40 focus:bg-white transition-all"
            >
              <option value="popular">Most popular</option>
              <option value="newest">Newest</option>
              <option value="price_asc">Price: low to high</option>
              <option value="price_desc">Price: high to low</option>
            </select>
          </div>
          <div className="mt-6 flex flex-wrap gap-2">
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
      </section>

      <main className="max-w-7xl mx-auto px-6 lg:px-10 py-12">
        {books.isError ? (
          <div className="border border-destructive/20 bg-destructive/5 p-6 text-sm">
            <p className="font-medium text-destructive">Couldn't load books</p>
            <p className="mt-1 text-brand/60">{(books.error as Error)?.message}</p>
          </div>
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
            <p className="eyebrow text-brand/45 mb-6">Available Books ({books.data.meta.total})</p>
            <div className="grid sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-x-6 gap-y-10">
              {books.data.data.map((b) => (
                <PublicBookCard key={b.id} book={b} />
              ))}
            </div>
          </>
        )}
      </main>
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
    <Link to="/bookshop/$bookId" params={{ bookId: b.id }} className="group block">
      <div className="aspect-[2/3] bg-brand/10 mb-4 overflow-hidden shadow-sm hover:shadow-md transition-all duration-300 relative">
        {b.cover_url ? (
          <img
            src={b.cover_url}
            alt={b.title}
            className="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-300"
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
      <div className="mt-3 flex items-center justify-between gap-3 text-xs text-brand/55">
        <span>{b.category ?? "General science"}</span>
        {typeof b.rating === "number" && (
          <span className="inline-flex items-center gap-1">
            <Star className="h-3 w-3 fill-accent text-accent" />
            {b.rating.toFixed(1)}
          </span>
        )}
      </div>
      <div className="mt-3 flex items-center justify-between gap-3">
        <p className="text-sm text-brand font-medium">
          {b.is_free ? (
            <span className="text-emerald-700">Free</span>
          ) : (
            formatPrice(b.price_cents, b.currency)
          )}
        </p>
        <span className="text-[11px] uppercase tracking-wider text-brand/45">{stockLabel}</span>
      </div>
      <span className="mt-4 inline-flex text-xs font-medium text-accent underline underline-offset-4">
        {b.in_library ? "Read now" : "View details"}
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
      className={`border px-3 py-1.5 text-xs transition-colors ${
        active
          ? "border-brand bg-brand text-white"
          : "border-brand/15 bg-white/50 text-brand/70 hover:bg-white"
      }`}
    >
      {children}
    </button>
  );
}

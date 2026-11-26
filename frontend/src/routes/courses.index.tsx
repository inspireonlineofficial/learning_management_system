import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Search } from "lucide-react";
import { useState } from "react";
import { z } from "zod";

import { CourseCard, CourseCardSkeleton } from "@/components/course/course-card";
import { PublicFooter } from "@/components/layout/public-footer";
import { PublicHeader } from "@/components/layout/public-header";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { listCategories, listCourses, type CourseLevel } from "@/lib/api/courses";

const searchSchema = z.object({
  q: z.string().optional(),
  category: z.string().optional(),
  level: z.enum(["beginner", "intermediate", "advanced"]).optional(),
  price: z.enum(["free", "paid"]).optional(),
  page: z.coerce.number().int().min(1).optional(),
});

export const Route = createFileRoute("/courses/")({
  validateSearch: searchSchema,
  head: () => ({
    meta: [
      { title: "Course catalog — Inspire Online" },
      {
        name: "description",
        content:
          "Browse the Inspire science course catalog. Find courses by subject, level, and instructor.",
      },
      { property: "og:title", content: "Course catalog — Inspire Online" },
      {
        property: "og:description",
        content: "Find courses by category, level, and instructor at Inspire.",
      },
    ],
  }),
  component: CatalogPage,
});

const levels: { value: CourseLevel; label: string }[] = [
  { value: "beginner", label: "Beginner" },
  { value: "intermediate", label: "Intermediate" },
  { value: "advanced", label: "Advanced" },
];

function CatalogPage() {
  const search = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });
  const [draft, setDraft] = useState(search.q ?? "");

  const { data, isLoading, isError, error, refetch, isFetching } = useQuery({
    queryKey: ["courses", search],
    queryFn: () =>
      listCourses({
        search: search.q,
        category: search.category,
        level: search.level,
        price_type: search.price,
        page: search.page ?? 1,
        limit: 12,
      }),
    placeholderData: (prev: any) => prev,
  });

  const { data: categories } = useQuery({
    queryKey: ["categories"],
    queryFn: listCategories,
  });

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    navigate({ search: (prev) => ({ ...prev, q: draft || undefined, page: 1 }) });
  };

  return (
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy-light font-sans antialiased flex flex-col">
      <PublicHeader active="courses" />

      <section className="hero-atmosphere text-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16 lg:py-20">
          <p className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-white/10 border border-amber-300/35 text-amber-200 text-xs font-semibold tracking-[0.14em] uppercase mb-5">
            <span className="w-2 h-2 rounded-full bg-gold" aria-hidden="true" />
            Course Catalog
          </p>
          <h1 className="font-serif text-3xl sm:text-4xl lg:text-5xl font-bold leading-[1.14] max-w-3xl text-balance">
            A curated science catalog.
          </h1>
          <p className="mt-4 max-w-xl text-slate-200 text-sm sm:text-base leading-relaxed">
            Browse structured courses by subject, level, and instructor. Sign in to enroll and track
            your weekly study progress.
          </p>

          <form
            onSubmit={submit}
            className="mt-8 bg-white rounded-xl p-3 sm:p-4 shadow-xl border border-borderui max-w-2xl flex flex-col sm:flex-row gap-3"
          >
            <div className="relative flex-1 min-w-0">
              <Search
                className="absolute left-3.5 top-1/2 -translate-y-1/2 h-5 w-5 text-slatebody pointer-events-none"
                aria-hidden="true"
              />
              <input
                type="search"
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="Search subjects or courses…"
                className="w-full min-h-[46px] pl-11 pr-4 py-2.5 rounded-lg bg-canvas border border-borderui text-navy text-sm sm:text-base placeholder:text-slatebody focus:bg-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
              />
            </div>
            <button
              type="submit"
              className="inline-flex items-center justify-center min-h-[46px] rounded-lg bg-navy text-white px-6 py-2.5 text-sm font-semibold hover:bg-navy-light shadow-sm transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
            >
              Search
            </button>
          </form>
        </div>
      </section>

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-10 sm:py-12">
        {/* Filters */}
        <div className="bg-white rounded-xl border border-borderui p-4 sm:p-5 shadow-sm mb-8 space-y-3">
          <div className="flex flex-wrap gap-2">
            <FilterChip
              active={!search.category}
              onClick={() => navigate({ search: (p) => ({ ...p, category: undefined, page: 1 }) })}
            >
              All subjects
            </FilterChip>
            {categories?.data.map((c) => (
              <FilterChip
                key={c.id}
                active={search.category === c.slug}
                onClick={() => navigate({ search: (p) => ({ ...p, category: c.slug, page: 1 }) })}
              >
                {c.name}
              </FilterChip>
            ))}
          </div>
          <div className="h-px bg-borderui/60" />
          <div className="flex flex-wrap items-center gap-2">
            <FilterChip
              active={!search.level}
              onClick={() => navigate({ search: (p) => ({ ...p, level: undefined, page: 1 }) })}
            >
              Any level
            </FilterChip>
            {levels.map((l) => (
              <FilterChip
                key={l.value}
                active={search.level === l.value}
                onClick={() => navigate({ search: (p) => ({ ...p, level: l.value, page: 1 }) })}
              >
                {l.label}
              </FilterChip>
            ))}
            <span className="hidden sm:inline-block h-5 w-px bg-borderui mx-1" aria-hidden="true" />
            <FilterChip
              active={!search.price}
              onClick={() => navigate({ search: (p) => ({ ...p, price: undefined, page: 1 }) })}
            >
              Any price
            </FilterChip>
            <FilterChip
              active={search.price === "free"}
              onClick={() => navigate({ search: (p) => ({ ...p, price: "free", page: 1 }) })}
            >
              Free
            </FilterChip>
            <FilterChip
              active={search.price === "paid"}
              onClick={() => navigate({ search: (p) => ({ ...p, price: "paid", page: 1 }) })}
            >
              Paid
            </FilterChip>
          </div>
        </div>

        {/* Results */}
        {isError && (
          <QueryErrorPanel
            error={error}
            title="Couldn't load the catalog"
            onRetry={() => refetch()}
          />
        )}

        {isLoading && (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <CourseCardSkeleton key={i} />
            ))}
          </div>
        )}

        {!isLoading && data && data.data.length === 0 && (
          <div className="rounded-xl border border-dashed border-borderui bg-white px-6 py-16 text-center">
            <p className="font-serif text-2xl font-bold text-navy">No courses match your search</p>
            <p className="mt-2 text-sm text-slatebody">
              Try a different keyword or clear the active filters.
            </p>
          </div>
        )}

        {!isLoading && data && data.data.length > 0 && (
          <>
            <p className="text-xs font-bold tracking-[0.14em] uppercase text-bronze mb-5">
              {data.meta.total.toLocaleString()} course{data.meta.total === 1 ? "" : "s"}
              {isFetching && " · refreshing"}
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
              {data.data.map((c) => (
                <CourseCard key={c.id} course={c} />
              ))}
            </div>

            {data.meta.total_pages > 1 && (
              <div className="mt-10 flex items-center justify-center gap-3">
                <button
                  type="button"
                  disabled={(search.page ?? 1) <= 1}
                  onClick={() =>
                    navigate({
                      search: (p) => ({ ...p, page: Math.max(1, (p.page ?? 1) - 1) }),
                    })
                  }
                  className="inline-flex items-center justify-center min-h-[44px] px-4 rounded-lg border border-borderui bg-white text-sm font-semibold text-navy disabled:opacity-40 hover:bg-stoneband transition-colors"
                >
                  Previous
                </button>
                <span className="text-sm font-medium text-slatebody">
                  Page {data.meta.page} of {data.meta.total_pages}
                </span>
                <button
                  type="button"
                  disabled={(search.page ?? 1) >= data.meta.total_pages}
                  onClick={() =>
                    navigate({
                      search: (p) => ({ ...p, page: (p.page ?? 1) + 1 }),
                    })
                  }
                  className="inline-flex items-center justify-center min-h-[44px] px-4 rounded-lg border border-borderui bg-white text-sm font-semibold text-navy disabled:opacity-40 hover:bg-stoneband transition-colors"
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </main>

      <PublicFooter />
    </div>
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
      className={`inline-flex items-center justify-center min-h-[40px] px-3.5 py-1.5 rounded-lg text-xs sm:text-sm font-semibold transition-colors border focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy ${
        active
          ? "bg-navy text-white border-navy"
          : "bg-canvas text-navy-light border-borderui hover:border-navy hover:bg-white"
      }`}
    >
      {children}
    </button>
  );
}

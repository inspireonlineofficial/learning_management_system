import { usePublicSiteConfig } from "@/hooks/use-site-config";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { GraduationCap, Search } from "lucide-react";
import { useState } from "react";
import { z } from "zod";

import { CourseCard, CourseCardSkeleton } from "@/components/course/course-card";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/layout/app-shell";
import { FilterChip, HeroPanel, PageHero, fieldClass } from "@/components/layout/page-hero";
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
  const page = usePublicSiteConfig().pages.courses;
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

      <PageHero eyebrow={page.eyebrow} title={page.title} body={page.body}>
        <HeroPanel
          as="form"
          onSubmit={submit}
          className="max-w-2xl flex flex-col sm:flex-row gap-3"
        >
          <div className="relative flex-1 min-w-0">
            <Search
              className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-slatebody pointer-events-none"
              aria-hidden="true"
            />
            <input
              type="search"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Search subjects or courses…"
              aria-label="Search courses"
              className={`${fieldClass} pl-10`}
            />
          </div>
          <Button type="submit" size="lg" className="min-h-11">
            Search
          </Button>
        </HeroPanel>
      </PageHero>

      <main className="flex-1 page-container py-10 sm:py-12">
        {/* Filters */}
        <div className="mb-8 space-y-3 border-b border-borderui pb-6">
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
          <EmptyState
            icon={GraduationCap}
            title="No courses match your search"
            description="Try a different keyword or clear the active filters."
          />
        )}

        {!isLoading && data && data.data.length > 0 && (
          <>
            <p className="mb-5 text-sm text-slatebody">
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
                  className="inline-flex items-center justify-center min-h-10 px-4 rounded-md border border-borderui bg-white text-sm font-medium text-navy disabled:opacity-40 hover:border-navy transition-colors"
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
                  className="inline-flex items-center justify-center min-h-10 px-4 rounded-md border border-borderui bg-white text-sm font-medium text-navy disabled:opacity-40 hover:border-navy transition-colors"
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

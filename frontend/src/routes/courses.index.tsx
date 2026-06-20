import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Search } from "lucide-react";
import { useState } from "react";
import { z } from "zod";

import { CourseCard, CourseCardSkeleton } from "@/components/course/course-card";
import { BrandLogo } from "@/components/layout/brand-logo";
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
      { title: "Course catalog — Inspire LMS" },
      {
        name: "description",
        content:
          "Browse the Inspire LMS science course catalog. Find courses by subject, level, and instructor.",
      },
      { property: "og:title", content: "Course catalog — Inspire LMS" },
      {
        property: "og:description",
        content: "Find courses by category, level, and instructor at Inspire LMS.",
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
  const navigate = useNavigate({ from: "/courses" });
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
    navigate({ search: (prev: any) => ({ ...prev, q: draft || undefined, page: 1 }) });
  };

  return (
    <div className="min-h-screen overflow-x-hidden bg-surface text-brand font-sans">
      <header className="px-6 md:px-12 lg:px-20 py-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-brand/10">
        <BrandLogo imageClassName="max-h-14 max-w-[220px]" />
        <nav className="flex w-full flex-wrap items-center gap-x-4 gap-y-2 text-sm sm:w-auto">
          <Link to="/courses" className="text-brand font-medium">
            Catalog
          </Link>
          <Link to="/bookshop" className="text-brand/60 hover:text-brand transition-colors">
            Bookshop
          </Link>
          <Link to="/login" className="text-brand/60 hover:text-brand transition-colors">
            Sign in
          </Link>
          <Link
            to="/register"
            className="bg-brand text-white px-5 py-2.5 hover:bg-brand/90 transition-colors"
          >
            Enroll
          </Link>
        </nav>
      </header>

      <section className="px-6 md:px-12 lg:px-20 py-12 lg:py-16 border-b border-brand/10">
        <p className="eyebrow text-accent mb-4">The library</p>
        <h1 className="font-serif text-3xl leading-[1.1] max-w-3xl break-words sm:text-4xl lg:text-6xl">
          A curated science catalog.
        </h1>
        <p className="mt-6 max-w-xl text-brand/60 leading-relaxed">
          Browse courses by subject and level. Sign in to enroll and track your progress.
        </p>

        <form onSubmit={submit} className="mt-10 flex flex-col sm:flex-row gap-3 max-w-2xl">
          <div className="relative flex-1 min-w-0">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-brand/40" />
            <input
              type="search"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Search the catalog…"
              className="w-full pl-11 pr-4 py-4 bg-white border border-brand/15 focus:border-brand/40 focus:outline-none text-sm"
            />
          </div>
          <button
            type="submit"
            className="bg-brand text-white px-7 py-4 text-sm font-medium hover:bg-brand/90 transition-colors"
          >
            Search
          </button>
        </form>
      </section>

      <section className="px-6 md:px-12 lg:px-20 py-10">
        {/* Filters */}
        <div className="flex flex-wrap gap-2 mb-8">
          <FilterChip
            active={!search.category}
            onClick={() =>
              navigate({ search: (p: any) => ({ ...p, category: undefined, page: 1 }) })
            }
          >
            All subjects
          </FilterChip>
          {categories?.data.map((c) => (
            <FilterChip
              key={c.id}
              active={search.category === c.slug}
              onClick={() =>
                navigate({ search: (p: any) => ({ ...p, category: c.slug, page: 1 }) })
              }
            >
              {c.name}
            </FilterChip>
          ))}
          <div className="w-full h-px bg-brand/5 my-2" />
          <FilterChip
            active={!search.level}
            onClick={() => navigate({ search: (p: any) => ({ ...p, level: undefined, page: 1 }) })}
          >
            Any level
          </FilterChip>
          {levels.map((l) => (
            <FilterChip
              key={l.value}
              active={search.level === l.value}
              onClick={() => navigate({ search: (p: any) => ({ ...p, level: l.value, page: 1 }) })}
            >
              {l.label}
            </FilterChip>
          ))}
          <div className="w-full h-px bg-brand/5 my-2" />
          <FilterChip
            active={!search.price}
            onClick={() => navigate({ search: (p: any) => ({ ...p, price: undefined, page: 1 }) })}
          >
            Any price
          </FilterChip>
          <FilterChip
            active={search.price === "free"}
            onClick={() => navigate({ search: (p: any) => ({ ...p, price: "free", page: 1 }) })}
          >
            Free
          </FilterChip>
          <FilterChip
            active={search.price === "paid"}
            onClick={() => navigate({ search: (p: any) => ({ ...p, price: "paid", page: 1 }) })}
          >
            Paid
          </FilterChip>
        </div>

        {/* Results */}
        {isError && (
          <div className="border border-destructive/20 bg-destructive/5 p-6 text-sm">
            <p className="font-medium text-destructive">Couldn't load the catalog</p>
            <p className="mt-1 text-brand/60">
              {(error as Error)?.message ?? "The library is temporarily unavailable."}
            </p>
            <button
              onClick={() => refetch()}
              className="mt-4 inline-flex items-center px-4 py-2 bg-brand text-white text-xs"
            >
              Try again
            </button>
          </div>
        )}

        {isLoading && (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <CourseCardSkeleton key={i} />
            ))}
          </div>
        )}

        {!isLoading && data && data.data.length === 0 && (
          <div className="border border-dashed border-brand/15 px-8 py-20 text-center">
            <p className="font-serif text-2xl">No courses match your search</p>
            <p className="mt-2 text-sm text-brand/55">
              Try a different keyword or clear the filters.
            </p>
          </div>
        )}

        {!isLoading && data && data.data.length > 0 && (
          <>
            <p className="text-xs text-brand/45 mb-4">
              {data.meta.total.toLocaleString()} course{data.meta.total === 1 ? "" : "s"}
              {isFetching && " · refreshing"}
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
              {data.data.map((c) => (
                <CourseCard key={c.id} course={c} />
              ))}
            </div>

            {data.meta.total_pages > 1 && (
              <div className="mt-12 flex items-center justify-center gap-3">
                <button
                  disabled={(search.page ?? 1) <= 1}
                  onClick={() =>
                    navigate({
                      search: (p: any) => ({ ...p, page: Math.max(1, (p.page ?? 1) - 1) }),
                    })
                  }
                  className="px-4 py-2 border border-brand/15 text-sm disabled:opacity-40 hover:bg-brand/[0.03]"
                >
                  Previous
                </button>
                <span className="text-sm text-brand/55">
                  Page {data.meta.page} of {data.meta.total_pages}
                </span>
                <button
                  disabled={(search.page ?? 1) >= data.meta.total_pages}
                  onClick={() =>
                    navigate({
                      search: (p: any) => ({ ...p, page: (p.page ?? 1) + 1 }),
                    })
                  }
                  className="px-4 py-2 border border-brand/15 text-sm disabled:opacity-40 hover:bg-brand/[0.03]"
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </section>
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
      onClick={onClick}
      className={`px-4 py-2 text-xs font-medium transition-colors ${
        active
          ? "bg-brand text-white"
          : "border border-brand/15 text-brand/70 hover:text-brand hover:bg-brand/[0.03]"
      }`}
    >
      {children}
    </button>
  );
}

import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Star } from "lucide-react";

import { AppShell } from "@/components/layout/app-shell";
import { listCourseReviews } from "@/lib/api/courses";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/reviews")({
  component: Page,
});

function Page() {
  const { courseId } = Route.useParams();
  const reviews = useQuery({
    queryKey: ["course-reviews", courseId],
    queryFn: () => listCourseReviews(courseId, { limit: 100 }),
  });

  const rows = reviews.data?.reviews ?? [];
  const average =
    rows.length > 0 ? rows.reduce((sum, review) => sum + review.rating, 0) / rows.length : 0;

  return (
    <AppShell>
      <Link
        to="/teacher/courses/$courseId/builder"
        params={{ courseId }}
        className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-6"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Back to builder
      </Link>
      <p className="eyebrow text-accent">Reviews</p>
      <h1 className="mt-3 font-serif text-4xl lg:text-5xl">Course reviews</h1>
      <p className="mt-2 text-sm text-brand/55">
        {rows.length} reviews · {average.toFixed(1)} average rating
      </p>

      {reviews.isLoading ? (
        <div className="mt-8 space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-24 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : rows.length === 0 ? (
        <p className="mt-8 text-sm text-brand/55 border border-dashed border-brand/15 p-8 text-center">
          No reviews yet.
        </p>
      ) : (
        <ul className="mt-8 space-y-3">
          {rows.map((review) => (
            <li key={review.id} className="border border-brand/15 bg-white/50 p-4">
              <div className="flex items-center gap-1 text-accent">
                {Array.from({ length: review.rating }).map((_, i) => (
                  <Star key={i} className="h-3.5 w-3.5 fill-current" />
                ))}
              </div>
              <p className="mt-3 text-sm text-brand/75">{review.comment || "No written review."}</p>
              <p className="mt-3 text-[11px] text-brand/40">
                {new Date(review.updated_at || review.created_at).toLocaleDateString()}
              </p>
            </li>
          ))}
        </ul>
      )}
    </AppShell>
  );
}

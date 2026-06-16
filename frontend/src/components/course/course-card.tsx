import { Link } from "@tanstack/react-router";
import { Clock, Star, Users } from "lucide-react";

import type { CourseSummary } from "@/lib/api/courses";

function formatPrice(price?: number, currency = "BDT") {
  if (price === undefined || price === null) return "—";
  if (price === 0) return "Free";
  return new Intl.NumberFormat("en-BD", { style: "currency", currency, maximumFractionDigits: 0 })
    .format(price)
    .replace(/^\D+/, (s) => `${s} `);
}

export function CourseCard({ course }: { course: CourseSummary }) {
  return (
    <Link
      to="/courses/$courseId"
      params={{ courseId: course.id }}
      className="group flex flex-col border border-brand/10 bg-white/50 hover:bg-white transition-colors"
    >
      <div className="aspect-[16/10] bg-brand/5 overflow-hidden">
        {course.cover_url ? (
          <img
            src={course.cover_url}
            alt={course.title}
            loading="lazy"
            className="h-full w-full object-cover group-hover:scale-[1.02] transition-transform duration-700"
          />
        ) : (
          <div className="h-full w-full grid place-items-center font-serif italic text-3xl text-brand/20">
            {course.title.slice(0, 1)}
          </div>
        )}
      </div>
      <div className="p-5 flex flex-col flex-1">
        {course.category?.name && (
          <p className="eyebrow text-accent mb-2">{course.category.name}</p>
        )}
        <h3 className="font-serif text-xl leading-snug text-balance group-hover:text-accent transition-colors">
          {course.title}
        </h3>
        {course.teacher?.full_name && (
          <p className="mt-2 text-sm text-brand/55">by {course.teacher.full_name}</p>
        )}
        <div className="mt-auto flex flex-col gap-3 pt-5 text-xs text-brand/55 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap items-center gap-3">
            {typeof course.rating === "number" && (
              <span className="inline-flex items-center gap-1">
                <Star className="h-3 w-3 fill-accent text-accent" />
                {course.rating.toFixed(1)}
              </span>
            )}
            {typeof course.enrollment_count === "number" && (
              <span className="inline-flex items-center gap-1">
                <Users className="h-3 w-3" />
                {course.enrollment_count.toLocaleString()}
              </span>
            )}
            {typeof course.duration_minutes === "number" && (
              <span className="inline-flex items-center gap-1">
                <Clock className="h-3 w-3" />
                {Math.round(course.duration_minutes / 60)}h
              </span>
            )}
          </div>
          <span className="font-serif text-base text-brand sm:text-right">
            {formatPrice(course.price, course.currency)}
          </span>
        </div>
      </div>
    </Link>
  );
}

export function CourseCardSkeleton() {
  return (
    <div className="border border-brand/10 bg-white/30 animate-pulse">
      <div className="aspect-[16/10] bg-brand/5" />
      <div className="p-5 space-y-3">
        <div className="h-3 w-20 bg-brand/10" />
        <div className="h-5 w-3/4 bg-brand/10" />
        <div className="h-3 w-1/2 bg-brand/10" />
      </div>
    </div>
  );
}

import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

/**
 * Shared hero band for public listing pages (Courses, Bookshop, Forum).
 * One layout, one type scale, no decorative markers — keep page-specific
 * controls in `children` (usually a <HeroPanel>).
 */
export function PageHero({
  eyebrow,
  title,
  body,
  children,
  className,
}: {
  eyebrow?: string;
  title: string;
  body?: string | null;
  children?: ReactNode;
  className?: string;
}) {
  return (
    <section className={cn("hero-atmosphere text-white", className)}>
      <div className="page-container py-10 sm:py-14 lg:py-16">
        {eyebrow && <p className="eyebrow text-amber-200/90 mb-3">{eyebrow}</p>}
        <h1 className="font-serif text-3xl sm:text-4xl font-bold leading-tight text-pretty max-w-2xl">
          {title}
        </h1>
        {body && (
          <p className="mt-3 max-w-xl text-slate-300 text-sm sm:text-base leading-relaxed">
            {body}
          </p>
        )}
        {children && <div className="mt-8">{children}</div>}
      </div>
    </section>
  );
}

/** White control surface that sits inside a PageHero (search, sort, filters). */
export function HeroPanel({
  children,
  className,
  as: Tag = "div",
  ...rest
}: {
  children: ReactNode;
  className?: string;
  as?: "div" | "form";
} & React.HTMLAttributes<HTMLElement> & { onSubmit?: React.FormEventHandler }) {
  return (
    <Tag
      className={cn("max-w-3xl rounded-lg border border-white/10 bg-white p-3 sm:p-4", className)}
      {...(rest as Record<string, unknown>)}
    >
      {children}
    </Tag>
  );
}

/** Shared field styling for inputs/selects placed in a HeroPanel or filter bar. */
export const fieldClass =
  "w-full min-h-11 rounded-md border border-borderui bg-white px-3.5 py-2 text-sm text-navy placeholder:text-slatebody transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-navy";

export function FilterChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        "inline-flex items-center justify-center min-h-9 px-3.5 rounded-md border text-sm font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy",
        active
          ? "border-navy bg-navy text-white"
          : "border-borderui bg-white text-navy-light hover:border-navy",
      )}
    >
      {children}
    </button>
  );
}

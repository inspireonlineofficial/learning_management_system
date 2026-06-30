import { Link } from "@tanstack/react-router";

import { BrandLogo } from "@/components/layout/brand-logo";
import { PublicHeaderActions } from "@/components/layout/public-header-actions";

type PublicHeaderSection = "courses" | "bookshop" | "forum";

const navItems: { to: string; label: string; section: PublicHeaderSection }[] = [
  { to: "/courses", label: "Courses", section: "courses" },
  { to: "/bookshop", label: "Bookshop", section: "bookshop" },
  { to: "/forum", label: "Forum", section: "forum" },
];

export function PublicHeader({ active }: { active?: PublicHeaderSection }) {
  return (
    <header className="border-b border-brand/10 bg-surface text-brand">
      <div className="mx-auto flex min-h-[5.5rem] w-full max-w-7xl flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between md:px-12 lg:px-10">
        <BrandLogo imageClassName="max-h-14 max-w-[220px]" />
        <nav
          className="flex w-full flex-wrap items-center gap-x-4 gap-y-2 text-sm sm:w-auto sm:justify-end"
          aria-label="Primary navigation"
        >
          {navItems.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              className={
                active === item.section
                  ? "font-medium text-brand"
                  : "text-brand/60 transition-colors hover:text-brand"
              }
            >
              {item.label}
            </Link>
          ))}
          <PublicHeaderActions />
        </nav>
      </div>
    </header>
  );
}

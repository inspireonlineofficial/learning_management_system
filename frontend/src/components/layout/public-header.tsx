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
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-3 px-4 py-3 sm:min-h-[5.5rem] sm:flex-row sm:items-center sm:justify-between sm:px-6 sm:py-5 md:px-12 lg:px-10">
        <div className="flex min-h-10 items-center justify-between gap-3 sm:min-h-0">
          <BrandLogo
            className="shrink-0"
            imageClassName="max-h-10 max-w-[160px] sm:max-h-14 sm:max-w-[220px]"
          />
          <div className="shrink-0 sm:hidden">
            <PublicHeaderActions compact />
          </div>
        </div>
        <div className="flex min-w-0 items-center justify-between gap-4 sm:justify-end">
          <nav
            className="-mx-4 flex min-w-0 flex-1 items-center gap-4 overflow-x-auto whitespace-nowrap px-4 pb-1 text-sm sm:mx-0 sm:flex-none sm:overflow-visible sm:px-0 sm:pb-0"
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
          </nav>
          <div className="hidden shrink-0 sm:block">
            <PublicHeaderActions />
          </div>
        </div>
      </div>
    </header>
  );
}

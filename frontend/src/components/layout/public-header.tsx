import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import { Menu, X } from "lucide-react";

import { BrandLogo } from "@/components/layout/brand-logo";
import { PublicHeaderActions } from "@/components/layout/public-header-actions";

type PublicHeaderSection = "courses" | "bookshop" | "forum";

const navItems: Array<
  | {
      kind: "route";
      to: "/courses" | "/bookshop" | "/forum";
      label: string;
      section: PublicHeaderSection;
    }
  | { kind: "anchor"; href: string; label: string }
> = [
  { kind: "route", to: "/courses", label: "Courses", section: "courses" },
  { kind: "route", to: "/bookshop", label: "Bookshop", section: "bookshop" },
  { kind: "anchor", href: "/#outcomes", label: "Outcomes" },
  { kind: "anchor", href: "/#faq", label: "FAQ" },
  { kind: "route", to: "/forum", label: "Forum", section: "forum" },
];

export function PublicHeader({ active }: { active?: PublicHeaderSection }) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    if (!mobileMenuOpen) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setMobileMenuOpen(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [mobileMenuOpen]);

  const linkClass = (isActive?: boolean) =>
    `inline-flex items-center min-h-[44px] px-3.5 text-sm font-medium rounded-lg transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy ${
      isActive
        ? "text-navy font-semibold underline underline-offset-4 decoration-2 decoration-gold"
        : "text-navy-light hover:text-navy hover:underline underline-offset-4 decoration-2 decoration-gold"
    }`;

  return (
    <header className="sticky top-0 z-40 bg-white/95 backdrop-blur-md border-b border-borderui text-navy">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-20 gap-4">
          {/* Existing Inspire Logo preserved as-is */}
          <BrandLogo
            className="shrink-0 min-h-[44px] pr-2 rounded-lg focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
            imageClassName="max-h-10 max-w-[160px] sm:max-h-12 sm:max-w-[200px]"
          />

          {/* Desktop Navigation */}
          <nav
            aria-label="Primary navigation"
            className="hidden md:flex items-center gap-1 lg:gap-2"
          >
            {navItems.map((item) =>
              item.kind === "route" ? (
                <Link key={item.to} to={item.to} className={linkClass(active === item.section)}>
                  {item.label}
                </Link>
              ) : (
                <a key={item.href} href={item.href} className={linkClass(false)}>
                  {item.label}
                </a>
              ),
            )}
          </nav>

          {/* Desktop Auth Actions */}
          <div className="hidden md:flex items-center gap-3 shrink-0">
            <PublicHeaderActions />
          </div>

          {/* Mobile Menu Toggle */}
          <button
            type="button"
            onClick={() => setMobileMenuOpen((prev) => !prev)}
            className="md:hidden inline-flex items-center justify-center min-w-[44px] min-h-[44px] rounded-lg border border-borderui text-navy hover:bg-stoneband transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
            aria-expanded={mobileMenuOpen}
            aria-controls="mobile-menu"
            aria-label={mobileMenuOpen ? "Close main navigation menu" : "Open main navigation menu"}
          >
            {mobileMenuOpen ? (
              <X className="w-6 h-6" aria-hidden="true" />
            ) : (
              <Menu className="w-6 h-6" aria-hidden="true" />
            )}
          </button>
        </div>
      </div>

      {/* Mobile Navigation Panel */}
      {mobileMenuOpen && (
        <div
          id="mobile-menu"
          className="md:hidden bg-white border-t border-borderui px-4 pt-3 pb-5 shadow-lg"
        >
          <nav aria-label="Mobile navigation" className="flex flex-col space-y-1">
            {navItems.map((item) =>
              item.kind === "route" ? (
                <Link
                  key={item.to}
                  to={item.to}
                  onClick={() => setMobileMenuOpen(false)}
                  className={`flex items-center min-h-[44px] px-3 rounded-lg text-base font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy ${
                    active === item.section
                      ? "bg-stoneband text-navy font-semibold"
                      : "text-navy-light hover:bg-stoneband"
                  }`}
                >
                  {item.label}
                </Link>
              ) : (
                <a
                  key={item.href}
                  href={item.href}
                  onClick={() => setMobileMenuOpen(false)}
                  className="flex items-center min-h-[44px] px-3 rounded-lg text-base font-medium text-navy-light hover:bg-stoneband transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
                >
                  {item.label}
                </a>
              ),
            )}
          </nav>
          <div className="mt-4 pt-4 border-t border-borderui">
            <PublicHeaderActions mobile onNavigate={() => setMobileMenuOpen(false)} />
          </div>
        </div>
      )}
    </header>
  );
}

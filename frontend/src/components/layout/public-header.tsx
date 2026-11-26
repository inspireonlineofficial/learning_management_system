import { useEffect, useState } from "react";
import { Menu, X } from "lucide-react";

import { BrandLogo } from "@/components/layout/brand-logo";
import { PublicHeaderActions } from "@/components/layout/public-header-actions";
import { SiteNavLink } from "@/components/layout/site-nav-link";
import { safeHref } from "@/lib/api/homepage";
import { usePublicSiteConfig } from "@/hooks/use-site-config";

type PublicHeaderSection = "courses" | "bookshop" | "forum";

function isActiveLink(href: string, active?: PublicHeaderSection) {
  if (!active) return false;
  const path = href.split(/[?#]/)[0].replace(/\/+$/, "");
  return path === `/${active}`;
}

export function PublicHeader({ active }: { active?: PublicHeaderSection }) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const cfg = usePublicSiteConfig();
  const navItems = cfg.site.nav_links
    .map((l) => ({ label: l.label.trim(), href: safeHref(l.href) }))
    .filter((l): l is { label: string; href: string } => !!l.label && !!l.href);

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
            {navItems.map((item) => (
              <SiteNavLink
                key={`${item.label}-${item.href}`}
                href={item.href}
                className={linkClass(isActiveLink(item.href, active))}
              >
                {item.label}
              </SiteNavLink>
            ))}
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
            {navItems.map((item) => (
              <SiteNavLink
                key={`${item.label}-${item.href}`}
                href={item.href}
                onClick={() => setMobileMenuOpen(false)}
                className={`flex items-center min-h-[44px] px-3 rounded-lg text-base font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy ${
                  isActiveLink(item.href, active)
                    ? "bg-stoneband text-navy font-semibold"
                    : "text-navy-light hover:bg-stoneband"
                }`}
              >
                {item.label}
              </SiteNavLink>
            ))}
          </nav>
          <div className="mt-4 pt-4 border-t border-borderui">
            <PublicHeaderActions mobile onNavigate={() => setMobileMenuOpen(false)} />
          </div>
        </div>
      )}
    </header>
  );
}

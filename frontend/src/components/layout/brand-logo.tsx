import { Link } from "@tanstack/react-router";

// 720px-wide optimised copy of inspire-logo.png (the 2508px original weighed
// ~780 KB and was downloaded on every page for a ~40px-tall header logo).
import inspireLogo from "@/assets/inspire-logo-720.png";
import { safeHref } from "@/lib/api/homepage";
import { usePublicSiteConfig } from "@/hooks/use-site-config";

type BrandLogoProps = {
  className?: string;
  imageClassName?: string;
  to?: string;
};

export function BrandLogo({ className = "", imageClassName = "", to = "/" }: BrandLogoProps) {
  const cfg = usePublicSiteConfig();
  const brand = cfg.footer.brand_name || "Inspire Online";
  const custom = cfg.site.logo_url ? safeHref(cfg.site.logo_url) : null;

  return (
    <Link to={to} className={`inline-flex items-center ${className}`} aria-label={brand}>
      <img
        src={custom ?? inspireLogo}
        alt={brand}
        {...(custom ? {} : { width: 720, height: 180 })}
        decoding="async"
        className={`block h-auto w-auto object-contain ${imageClassName}`}
      />
    </Link>
  );
}

import type { ReactNode } from "react";
import { Link } from "@tanstack/react-router";

import { isExternalHref } from "@/lib/api/homepage";

/**
 * Renders an admin-configured link. Plain in-app paths use the router (no full
 * page reload); anchors, mailto/tel and external URLs use a normal <a>, with
 * external ones opening in a new tab. `href` must already be passed through
 * safeHref().
 */
export function SiteNavLink({
  href,
  className,
  onClick,
  children,
}: {
  href: string;
  className?: string;
  onClick?: () => void;
  children: ReactNode;
}) {
  if (isExternalHref(href)) {
    return (
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className={className}
        onClick={onClick}
      >
        {children}
      </a>
    );
  }
  if (href.startsWith("/") && !href.includes("#") && !href.includes("?")) {
    return (
      <Link to={href as never} className={className} onClick={onClick}>
        {children}
      </Link>
    );
  }
  return (
    <a href={href} className={className} onClick={onClick}>
      {children}
    </a>
  );
}

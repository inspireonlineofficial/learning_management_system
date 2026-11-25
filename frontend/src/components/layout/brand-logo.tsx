import { Link } from "@tanstack/react-router";

// 720px-wide optimised copy of inspire-logo.png (the 2508px original weighed
// ~780 KB and was downloaded on every page for a ~40px-tall header logo).
import inspireLogo from "@/assets/inspire-logo-720.png";

type BrandLogoProps = {
  className?: string;
  imageClassName?: string;
  to?: string;
};

export function BrandLogo({ className = "", imageClassName = "", to = "/" }: BrandLogoProps) {
  return (
    <Link to={to} className={`inline-flex items-center ${className}`} aria-label="Inspire Online">
      <img
        src={inspireLogo}
        alt="Inspire Online"
        width={720}
        height={180}
        decoding="async"
        className={`block h-auto w-auto object-contain ${imageClassName}`}
      />
    </Link>
  );
}

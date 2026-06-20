import { Link } from "@tanstack/react-router";

import inspireLogo from "@/assets/inspire-logo.png";

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
        className={`block h-auto w-auto object-contain ${imageClassName}`}
      />
    </Link>
  );
}

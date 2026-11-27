import { Link } from "@tanstack/react-router";

import { BrandLogo } from "@/components/layout/brand-logo";
import { SiteNavLink } from "@/components/layout/site-nav-link";
import { safeHref, type CourseTrackItem, type SiteLink } from "@/lib/api/homepage";
import { usePublicSiteConfig } from "@/hooks/use-site-config";

const linkClass =
  "inline-flex items-center min-h-[36px] text-slate-200 hover:text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded";
const headingClass = "font-sans text-xs font-bold tracking-[0.12em] uppercase text-amber-300 mb-4";

function usableLinks(list: SiteLink[]) {
  return list
    .map((l) => ({ label: l.label.trim(), href: safeHref(l.href) }))
    .filter((l): l is { label: string; href: string } => !!l.label && !!l.href);
}

export function PublicFooter({
  brandName,
  tagline,
  tracks,
  onTrackSelect,
}: {
  brandName?: string;
  tagline?: string;
  tracks?: CourseTrackItem[];
  onTrackSelect?: (trackTitle: string) => void;
}) {
  const cfg = usePublicSiteConfig();
  const site = cfg.site;
  const brand = brandName ?? cfg.footer.brand_name;
  const tag = tagline ?? cfg.footer.tagline;
  const trackList = tracks ?? cfg.course_catalog.tracks;

  const columns = site.footer_columns
    .map((c) => ({ title: c.title.trim(), links: usableLinks(c.links) }))
    .filter((c) => c.title || c.links.length);
  const social = usableLinks(site.social_links);
  const legal = usableLinks(site.legal_links);
  const email = site.support_email.trim();
  const phone = site.phone.trim();
  const address = site.address.trim();
  const hasContact = Boolean(email || phone || address || social.length);

  return (
    <footer className="bg-navy text-slate-200 border-t border-slate-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8 sm:gap-10">
          <div>
            <div className="inline-flex rounded-lg bg-white/95 px-3 py-2 shadow-xs">
              <BrandLogo imageClassName="max-h-9 max-w-[160px]" />
            </div>
            {site.about_text && (
              <p className="mt-4 text-sm text-slate-300 leading-relaxed">{site.about_text}</p>
            )}
          </div>

          {site.show_tracks_column && trackList.length > 0 && (
            <div>
              <h3 className={headingClass}>{site.tracks_column_title}</h3>
              <ul className="space-y-1.5 text-sm">
                {trackList.map((track) => (
                  <li key={track.title}>
                    {onTrackSelect ? (
                      <a
                        href="#courses"
                        onClick={() => onTrackSelect(track.title)}
                        className={linkClass}
                      >
                        {track.title}
                      </a>
                    ) : (
                      <Link
                        to="/courses"
                        search={{ q: track.query?.trim() || track.title } as never}
                        className={linkClass}
                      >
                        {track.title}
                      </Link>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {columns.map((col, i) => (
            <div key={`${col.title}-${i}`}>
              {col.title && <h3 className={headingClass}>{col.title}</h3>}
              <ul className="space-y-1.5 text-sm">
                {col.links.map((l) => (
                  <li key={`${l.label}-${l.href}`}>
                    <SiteNavLink href={l.href} className={linkClass}>
                      {l.label}
                    </SiteNavLink>
                  </li>
                ))}
              </ul>
            </div>
          ))}

          {hasContact && (
            <div>
              <h3 className={headingClass}>{site.contact_title}</h3>
              <ul className="space-y-2.5 text-sm text-slate-300">
                {email && (
                  <li>
                    <span className="block text-xs uppercase tracking-wider text-slate-400">
                      Email
                    </span>
                    <a
                      href={`mailto:${email}`}
                      className="text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded break-all"
                    >
                      {email}
                    </a>
                  </li>
                )}
                {phone && (
                  <li>
                    {site.phone_label && (
                      <span className="block text-xs uppercase tracking-wider text-slate-400">
                        {site.phone_label}
                      </span>
                    )}
                    <a
                      href={`tel:${phone.replace(/[^\d+]/g, "")}`}
                      className="text-white hover:underline underline-offset-4"
                    >
                      {phone}
                    </a>
                  </li>
                )}
                {address && <li className="whitespace-pre-line">{address}</li>}
                {social.length > 0 && (
                  <li className="pt-1 flex flex-wrap gap-4 text-xs">
                    {social.map((l) => (
                      <SiteNavLink
                        key={`${l.label}-${l.href}`}
                        href={l.href}
                        className="text-slate-300 hover:text-white underline underline-offset-4"
                      >
                        {l.label}
                      </SiteNavLink>
                    ))}
                  </li>
                )}
              </ul>
            </div>
          )}
        </div>

        <div className="mt-10 sm:mt-12 pt-6 sm:pt-8 border-t border-slate-800 flex flex-col sm:flex-row items-center justify-between gap-3 text-sm text-slate-300 text-center sm:text-left">
          <p>
            &copy; {new Date().getFullYear()} {brand}. {tag}
          </p>
          <div className="flex flex-wrap items-center justify-center gap-4 text-xs text-slate-400">
            {legal.map((l) => (
              <SiteNavLink
                key={`${l.label}-${l.href}`}
                href={l.href}
                className="hover:text-white underline underline-offset-4"
              >
                {l.label}
              </SiteNavLink>
            ))}
            {site.bottom_note && <span>{site.bottom_note}</span>}
          </div>
        </div>
      </div>
    </footer>
  );
}

import { Link } from "@tanstack/react-router";

import { BrandLogo } from "@/components/layout/brand-logo";
import { DEFAULT_HOMEPAGE_CONFIG, type CourseTrackItem } from "@/lib/api/homepage";

export function PublicFooter({
  brandName = DEFAULT_HOMEPAGE_CONFIG.footer.brand_name,
  tagline = DEFAULT_HOMEPAGE_CONFIG.footer.tagline,
  tracks = DEFAULT_HOMEPAGE_CONFIG.course_catalog.tracks,
  onTrackSelect,
}: {
  brandName?: string;
  tagline?: string;
  tracks?: CourseTrackItem[];
  onTrackSelect?: (trackTitle: string) => void;
}) {
  return (
    <footer className="bg-navy text-slate-200 border-t border-slate-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8 sm:gap-10">
          {/* Column 1: Brand Summary (existing Inspire logo preserved on crisp light pill) */}
          <div>
            <div className="inline-flex rounded-lg bg-white/95 px-3 py-2 shadow-xs">
              <BrandLogo imageClassName="max-h-9 max-w-[160px]" />
            </div>
            <p className="mt-4 text-sm text-slate-300 leading-relaxed">
              Practical, teacher-led science tutoring and curated study materials designed for high
              school, college, and admission learners.
            </p>
          </div>

          {/* Column 2: Science Tracks */}
          <div>
            <h3 className="font-sans text-xs font-bold tracking-[0.16em] uppercase text-amber-300 mb-4">
              Science Tracks
            </h3>
            <ul className="space-y-1.5 text-sm">
              {tracks.map((track) => (
                <li key={track.title}>
                  {onTrackSelect ? (
                    <a
                      href="#courses"
                      onClick={() => onTrackSelect(track.title)}
                      className="inline-flex items-center min-h-[36px] text-slate-200 hover:text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                    >
                      {track.title}
                    </a>
                  ) : (
                    <Link
                      to="/courses"
                      search={{ q: track.query?.trim() || track.title } as never}
                      className="inline-flex items-center min-h-[36px] text-slate-200 hover:text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                    >
                      {track.title}
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          </div>

          {/* Column 3: Bookshop & Community */}
          <div>
            <h3 className="font-sans text-xs font-bold tracking-[0.16em] uppercase text-amber-300 mb-4">
              Bookshop &amp; Community
            </h3>
            <ul className="space-y-1.5 text-sm">
              <li>
                <Link
                  to="/bookshop"
                  className="inline-flex items-center min-h-[36px] text-slate-200 hover:text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                >
                  Textbooks &amp; Workbooks
                </Link>
              </li>
              <li>
                <Link
                  to="/bookshop"
                  className="inline-flex items-center min-h-[36px] text-slate-200 hover:text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                >
                  Lab Manuals &amp; Solution Guides
                </Link>
              </li>
              <li>
                <Link
                  to="/forum"
                  className="inline-flex items-center min-h-[36px] text-slate-200 hover:text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                >
                  Student Science Forum
                </Link>
              </li>
              <li>
                <a
                  href="/#outcomes"
                  className="inline-flex items-center min-h-[36px] text-slate-200 hover:text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                >
                  Learning Outcomes
                </a>
              </li>
              <li>
                <a
                  href="/#faq"
                  className="inline-flex items-center min-h-[36px] text-slate-200 hover:text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                >
                  FAQ
                </a>
              </li>
            </ul>
          </div>

          {/* Column 4: Support & Contact */}
          <div>
            <h3 className="font-sans text-xs font-bold tracking-[0.16em] uppercase text-amber-300 mb-4">
              Support &amp; Contact
            </h3>
            <ul className="space-y-2.5 text-sm text-slate-300">
              <li>
                <span className="block text-xs uppercase tracking-wider text-slate-400">
                  Student Support Email
                </span>
                <a
                  href="mailto:support@inspireonlinebd.com"
                  className="text-white hover:underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                >
                  support@inspireonlinebd.com
                </a>
              </li>
              <li>
                <span className="block text-xs uppercase tracking-wider text-slate-400">
                  Academic Helpdesk (Sat–Thu, 10 AM – 8 PM BST)
                </span>
                <span className="text-white">+880 1700-000000 (WhatsApp &amp; Phone)</span>
              </li>
              <li className="pt-1 flex flex-wrap gap-4 text-xs">
                <a
                  href="/#faq"
                  className="text-slate-300 hover:text-white underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                >
                  Terms of Use
                </a>
                <a
                  href="/#faq"
                  className="text-slate-300 hover:text-white underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gold rounded"
                >
                  Privacy Policy
                </a>
              </li>
            </ul>
          </div>
        </div>

        {/* Bottom Copyright Bar */}
        <div className="mt-10 sm:mt-12 pt-6 sm:pt-8 border-t border-slate-800 flex flex-col sm:flex-row items-center justify-between gap-3 text-sm text-slate-300 text-center sm:text-left">
          <p>
            &copy; {new Date().getFullYear()} {brandName}. {tagline}
          </p>
          <p className="text-slate-400 text-xs">Built with WCAG 2.1 AA accessibility standards.</p>
        </div>
      </div>
    </footer>
  );
}

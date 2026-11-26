import { useEffect, useRef, useState } from "react";
import { ArrowRight, ChevronLeft, ChevronRight } from "lucide-react";

import type { Slide } from "@/lib/api/slides";

const DEFAULT_INTERVAL_MS = 5000;

function prefersReducedMotion() {
  if (typeof window === "undefined") return false;
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

export function SlideCarousel({
  slides,
  intervalMs = DEFAULT_INTERVAL_MS,
  autoplay = true,
}: {
  slides: Slide[];
  intervalMs?: number;
  autoplay?: boolean;
}) {
  const [index, setIndex] = useState(0);
  const [paused, setPaused] = useState(false);
  const touchStartX = useRef<number | null>(null);
  const count = slides.length;

  useEffect(() => {
    if (!autoplay || count <= 1 || paused || prefersReducedMotion()) return;
    const ms = Math.max(1500, slides[index]?.duration_ms || intervalMs);
    const id = window.setInterval(() => {
      setIndex((i) => (i + 1) % count);
    }, ms);
    return () => window.clearInterval(id);
  }, [count, paused, autoplay, intervalMs, index, slides]);

  if (count === 0) return null;

  const go = (n: number) => setIndex(((n % count) + count) % count);
  const slide = slides[index];

  return (
    <section
      aria-roledescription="carousel"
      aria-label="Featured slides"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocusCapture={() => setPaused(true)}
      onBlurCapture={() => setPaused(false)}
      onTouchStart={(e) => {
        touchStartX.current = e.touches[0].clientX;
      }}
      onTouchEnd={(e) => {
        if (touchStartX.current == null) return;
        const dx = e.changedTouches[0].clientX - touchStartX.current;
        if (Math.abs(dx) > 40) go(index + (dx < 0 ? 1 : -1));
        touchStartX.current = null;
      }}
      className="relative overflow-hidden bg-navy text-white"
    >
      <div
        className="flex transition-transform duration-700 ease-out"
        style={{ transform: `translateX(-${index * 100}%)` }}
      >
        {slides.map((item, i) => {
          const isActive = item.id === slide.id;
          const HeadingTag = i === 0 ? "h1" : "h2";

          return (
            <article
              key={item.id}
              className="relative w-full shrink-0 min-h-[380px] sm:min-h-[440px] lg:min-h-[500px] flex items-center"
              aria-hidden={!isActive}
            >
              <img
                src={item.media_url}
                alt={item.title}
                className="absolute inset-0 w-full h-full object-cover"
                loading={i === 0 ? "eager" : "lazy"}
              />
              {/* High-contrast atmospheric academic navy gradient */}
              <div className="absolute inset-0 bg-gradient-to-r from-navy/92 via-navy/75 to-navy/35" />
              <div
                className="absolute inset-0 pointer-events-none"
                style={{
                  backgroundImage:
                    "radial-gradient(ellipse 65% 60% at 85% 20%, rgba(217, 119, 6, 0.18), transparent 65%)",
                }}
              />

              <div className="relative w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-14 sm:py-16 lg:py-20">
                <div className="max-w-3xl">
                  <p className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-white/10 border border-amber-300/35 text-amber-200 text-xs font-semibold tracking-[0.14em] uppercase mb-5 backdrop-blur-xs">
                    <span className="w-2 h-2 rounded-full bg-gold" aria-hidden="true" />
                    Science Tutoring LMS &bull; SSC, HSC &amp; Admission Prep
                  </p>

                  <HeadingTag className="font-serif text-3xl sm:text-4xl lg:text-[3.25rem] font-bold tracking-tight text-white leading-[1.14] text-balance">
                    {item.title}
                  </HeadingTag>

                  {item.subtitle && (
                    <p className="mt-5 text-base sm:text-lg text-slate-200 leading-relaxed max-w-[60ch]">
                      {item.subtitle}
                    </p>
                  )}

                  <div className="mt-7 flex flex-wrap items-center gap-3.5">
                    {item.link_url ? (
                      <a
                        href={item.link_url}
                        tabIndex={isActive ? 0 : -1}
                        className="inline-flex items-center justify-center gap-2 min-h-[48px] px-6 py-3 rounded-lg bg-white text-navy font-semibold text-sm sm:text-base hover:bg-amber-50 shadow-sm transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gold"
                      >
                        Explore Science Tracks
                        <ArrowRight className="w-4 h-4" aria-hidden="true" />
                      </a>
                    ) : null}
                    <a
                      href="#bookshop"
                      tabIndex={isActive ? 0 : -1}
                      className="inline-flex items-center justify-center min-h-[48px] px-6 py-3 rounded-lg border border-slate-300/70 bg-navy/35 text-white font-semibold text-sm sm:text-base hover:bg-white/15 hover:border-white transition-colors backdrop-blur-xs focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gold"
                    >
                      Visit Bookshop
                    </a>
                  </div>
                </div>
              </div>
            </article>
          );
        })}
      </div>

      {count > 1 && (
        <>
          <button
            type="button"
            onClick={() => go(index - 1)}
            aria-label="Previous slide"
            className="hidden sm:grid place-items-center absolute left-4 lg:left-6 top-1/2 -translate-y-1/2 h-11 w-11 rounded-lg bg-white/85 hover:bg-white text-navy shadow-md transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gold"
          >
            <ChevronLeft className="h-5 w-5" aria-hidden="true" />
          </button>
          <button
            type="button"
            onClick={() => go(index + 1)}
            aria-label="Next slide"
            className="hidden sm:grid place-items-center absolute right-4 lg:right-6 top-1/2 -translate-y-1/2 h-11 w-11 rounded-lg bg-white/85 hover:bg-white text-navy shadow-md transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gold"
          >
            <ChevronRight className="h-5 w-5" aria-hidden="true" />
          </button>
          <div className="absolute bottom-5 left-0 right-0 flex justify-center gap-2">
            {slides.map((item, i) => (
              <button
                key={item.id}
                type="button"
                onClick={() => go(i)}
                aria-label={`Go to slide ${i + 1}`}
                aria-current={i === index}
                className={`h-2 rounded-full transition-all focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gold ${
                  i === index ? "w-9 bg-gold" : "w-4 bg-white/50 hover:bg-white/80"
                }`}
              />
            ))}
          </div>
        </>
      )}
    </section>
  );
}

export const AdCarousel = SlideCarousel;

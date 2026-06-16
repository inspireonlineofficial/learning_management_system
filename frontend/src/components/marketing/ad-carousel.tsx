import { useEffect, useRef, useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";

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
      className="relative overflow-hidden bg-brand/[0.04]"
    >
      <div
        className="flex transition-transform duration-700 ease-out"
        style={{ transform: `translateX(-${index * 100}%)` }}
      >
        {slides.map((item) => (
          <article
            key={item.id}
            className="relative w-full shrink-0 aspect-[21/9] md:aspect-[24/7]"
            aria-hidden={item.id !== slide.id}
          >
            <img
              src={item.media_url}
              alt={item.title}
              className="absolute inset-0 w-full h-full object-cover"
              loading="lazy"
            />
            <div className="absolute inset-0 bg-gradient-to-r from-brand/85 via-brand/55 to-transparent" />
            <div className="relative h-full px-6 md:px-12 lg:px-20 flex flex-col justify-center max-w-3xl text-white">
              <h2 className="font-serif text-3xl md:text-5xl leading-tight text-balance">
                {item.title}
              </h2>
              {item.subtitle && (
                <p className="mt-4 text-base md:text-lg max-w-xl text-white/75">{item.subtitle}</p>
              )}
              {item.link_url && (
                <a
                  href={item.link_url}
                  className="mt-6 inline-flex w-fit items-center gap-2 px-6 py-3 text-sm font-medium transition-colors bg-white text-brand hover:bg-white/90"
                >
                  Learn more
                  <span aria-hidden>→</span>
                </a>
              )}
            </div>
          </article>
        ))}
      </div>

      {count > 1 && (
        <>
          <button
            type="button"
            onClick={() => go(index - 1)}
            aria-label="Previous slide"
            className="absolute left-3 md:left-6 top-1/2 -translate-y-1/2 grid place-items-center h-10 w-10 bg-white/80 hover:bg-white text-brand transition-colors"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
          <button
            type="button"
            onClick={() => go(index + 1)}
            aria-label="Next slide"
            className="absolute right-3 md:right-6 top-1/2 -translate-y-1/2 grid place-items-center h-10 w-10 bg-white/80 hover:bg-white text-brand transition-colors"
          >
            <ChevronRight className="h-5 w-5" />
          </button>
          <div className="absolute bottom-4 left-0 right-0 flex justify-center gap-2">
            {slides.map((item, i) => (
              <button
                key={item.id}
                type="button"
                onClick={() => go(i)}
                aria-label={`Go to slide ${i + 1}`}
                aria-current={i === index}
                className={`h-1.5 transition-all ${
                  i === index ? "w-8 bg-brand" : "w-4 bg-brand/30 hover:bg-brand/50"
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

import {
  adminDeactivateSlide,
  adminListSlides,
  adminReorderSlides,
  adminUpdateSlide,
  listSlides,
} from "./slides";

export type Ad = {
  id: string;
  image_url: string;
  headline: string;
  subhead?: string;
  cta_label?: string;
  cta_href?: string;
  theme?: "light" | "dark";
  is_active?: boolean;
  position?: number;
  placement?: AdPlacement;
};

export type AdPlacement = "home_top" | "courses_top" | "bookshop_top";

export type AdSettings = {
  slide_interval_ms: number;
  autoplay: boolean;
};

export type AdInput = Omit<Ad, "id">;

// ---------- Public ----------
export function listAds(placement: AdPlacement = "home_top") {
  void placement;
  return listSlides().then((slides) => ({
    data: slides.slides.map((slide) => ({
      id: slide.id,
      image_url: slide.media_url,
      headline: slide.title,
      subhead: slide.subtitle,
      cta_href: slide.link_url,
      is_active: slide.is_active,
      position: slide.position,
    })),
    settings: { slide_interval_ms: 5000, autoplay: true },
  }));
}

// ---------- Admin ----------
export function adminListAds(placement?: AdPlacement) {
  void placement;
  return adminListSlides().then((slides) => ({
    data: slides.slides.map((slide) => ({
      id: slide.id,
      image_url: slide.media_url,
      headline: slide.title,
      subhead: slide.subtitle,
      cta_href: slide.link_url,
      is_active: slide.is_active,
      position: slide.position,
    })),
  }));
}

export function adminCreateAd(input: AdInput) {
  void input;
  return Promise.reject(
    new Error("Create slides from the Slides admin page with an uploaded media file."),
  );
}

export function adminUpdateAd(id: string, input: Partial<AdInput>) {
  return adminUpdateSlide(id, {
    title: input.headline,
    subtitle: input.subhead,
    link_url: input.cta_href,
    is_active: input.is_active,
  }).then((slide) => ({
    id: slide.id,
    image_url: slide.media_url,
    headline: slide.title,
    subhead: slide.subtitle,
    cta_href: slide.link_url,
    is_active: slide.is_active,
    position: slide.position,
  }));
}

export function adminDeleteAd(id: string) {
  return adminDeactivateSlide(id);
}

export function adminReorderAds(ids: string[], placement: AdPlacement = "home_top") {
  void placement;
  return adminReorderSlides(
    ids.reduce<Record<string, number>>((positions, id, index) => {
      positions[id] = index + 1;
      return positions;
    }, {}),
  );
}

export function adminGetAdSettings(placement: AdPlacement = "home_top") {
  void placement;
  return Promise.resolve({ slide_interval_ms: 5000, autoplay: true });
}

export function adminUpdateAdSettings(placement: AdPlacement, input: Partial<AdSettings>) {
  void placement;
  return Promise.resolve({
    slide_interval_ms: input.slide_interval_ms ?? 5000,
    autoplay: input.autoplay ?? true,
  });
}

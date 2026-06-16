import { apiRequest } from "./client";

export type Slide = {
  id: string;
  title: string;
  subtitle?: string | null;
  link_url?: string | null;
  media_url: string;
  media_type?: string | null;
  duration_ms: number;
  position: number;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
};

export type SlideList = {
  slides: Slide[];
};

function toFormData(input: {
  title?: string;
  subtitle?: string;
  link_url?: string;
  duration_ms?: number;
  position?: number;
  is_active?: boolean;
  media?: File;
}) {
  const formData = new FormData();
  if (input.title != null) formData.append("title", input.title);
  if (input.subtitle != null) formData.append("subtitle", input.subtitle);
  if (input.link_url != null) formData.append("link_url", input.link_url);
  if (input.duration_ms != null) formData.append("duration_ms", String(input.duration_ms));
  if (input.position != null) formData.append("position", String(input.position));
  if (input.is_active != null) formData.append("is_active", String(input.is_active));
  if (input.media) formData.append("media", input.media);
  return formData;
}

export function listSlides() {
  return apiRequest<SlideList>("/v1/public/slides");
}

export function adminListSlides() {
  return apiRequest<SlideList>("/v1/admin/slides", { auth: true });
}

export function adminCreateSlide(input: {
  title: string;
  subtitle?: string;
  link_url?: string;
  duration_ms?: number;
  position?: number;
  media: File;
}) {
  return apiRequest<Slide>("/v1/admin/slides", {
    method: "POST",
    auth: true,
    body: toFormData(input),
  });
}

export function adminUpdateSlide(
  slideId: string,
  input: {
    title?: string;
    subtitle?: string;
    link_url?: string;
    duration_ms?: number;
    position?: number;
    is_active?: boolean;
    media?: File;
  },
) {
  return apiRequest<Slide>(`/v1/admin/slides/${encodeURIComponent(slideId)}`, {
    method: "PATCH",
    auth: true,
    body: toFormData(input),
  });
}

export function adminDeactivateSlide(slideId: string) {
  return apiRequest<{ ok: true }>(`/v1/admin/slides/${encodeURIComponent(slideId)}/deactivate`, {
    method: "POST",
    auth: true,
  });
}

export function adminReorderSlides(positions: Record<string, number>) {
  return apiRequest<{ ok: true }>("/v1/admin/slides/reorder", {
    method: "POST",
    auth: true,
    body: { positions },
  });
}

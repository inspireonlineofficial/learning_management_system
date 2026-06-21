import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { ArrowDown, ArrowUp, Plus, Save, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { SlideCarousel } from "@/components/marketing/ad-carousel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  adminCreateSlide,
  adminDeactivateSlide,
  adminListSlides,
  adminReorderSlides,
  adminUpdateSlide,
  type Slide,
} from "@/lib/api/slides";

export const Route = createFileRoute("/_authenticated/admin/slides")({
  component: SlidesAdminPage,
});

type Draft = {
  title: string;
  subtitle: string;
  link_url: string;
  duration_ms: number;
  position: number;
  is_active: boolean;
};

const blankDraft = (position = 0): Draft => ({
  title: "",
  subtitle: "",
  link_url: "",
  duration_ms: 5000,
  position,
  is_active: true,
});

function SlidesAdminPage() {
  const qc = useQueryClient();
  const slidesQuery = useQuery({
    queryKey: ["admin-slides"],
    queryFn: adminListSlides,
  });

  const slides = slidesQuery.data?.slides ?? [];
  const activeSlides = useMemo(() => slides.filter((slide) => slide.is_active), [slides]);

  const create = useMutation({
    mutationFn: async (payload: Draft & { media: File }) =>
      adminCreateSlide({
        title: payload.title,
        subtitle: payload.subtitle || undefined,
        link_url: payload.link_url || undefined,
        duration_ms: payload.duration_ms,
        position: payload.position,
        media: payload.media,
      }),
    onSuccess: () => {
      toast.success("Slide created");
      qc.invalidateQueries({ queryKey: ["admin-slides"] });
      qc.invalidateQueries({ queryKey: ["slides", "public"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const reorder = useMutation({
    mutationFn: (positions: Record<string, number>) => adminReorderSlides(positions),
    onSuccess: () => {
      toast.success("Slide order updated");
      qc.invalidateQueries({ queryKey: ["admin-slides"] });
      qc.invalidateQueries({ queryKey: ["slides", "public"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <AppShell eyebrow="Homepage" title="Slide management">
      <p className="text-sm text-brand/60 -mt-2 mb-8 max-w-2xl">
        Upload homepage slides, control their order, and keep inactive items hidden from the public
        carousel.
      </p>

      <div className="space-y-10">
        {activeSlides.length > 0 && (
          <section>
            <h2 className="font-serif text-xl mb-3">Live preview</h2>
            <div className="border border-brand/10 bg-white/60 overflow-hidden">
              <SlideCarousel slides={activeSlides} />
            </div>
          </section>
        )}

        <CreateSlideForm
          nextPosition={(slides.at(-1)?.position ?? 0) + 1}
          onCreate={(payload) => create.mutate(payload)}
          isSaving={create.isPending}
        />

        <section>
          <div className="flex items-center justify-between mb-4">
            <h2 className="font-serif text-xl">Slides ({slides.length})</h2>
          </div>

          {slidesQuery.isLoading && <p className="text-sm text-brand/55">Loading slides…</p>}
          {slidesQuery.isError && (
            <QueryErrorPanel
              error={slidesQuery.error}
              variant="compact"
              message={(slidesQuery.error as Error)?.message ?? "Couldn't load slides"}
            />
          )}

          {!slidesQuery.isLoading && slides.length === 0 && (
            <div className="border border-dashed border-brand/15 px-8 py-16 text-center">
              <p className="font-serif text-xl">No slides yet</p>
              <p className="mt-2 text-sm text-brand/55">
                Add the first slide to populate the homepage carousel.
              </p>
            </div>
          )}

          <div className="space-y-4">
            {slides.map((slide, index) => (
              <SlideEditor
                key={slide.id}
                slide={slide}
                canMoveUp={index > 0}
                canMoveDown={index < slides.length - 1}
                onMove={(direction) => {
                  const next = [...slides].sort((a, b) => a.position - b.position);
                  const currentIndex = next.findIndex((s) => s.id === slide.id);
                  const swapIndex = direction === "up" ? currentIndex - 1 : currentIndex + 1;
                  if (swapIndex < 0 || swapIndex >= next.length) return;
                  const current = next[currentIndex];
                  const target = next[swapIndex];
                  next[currentIndex] = target;
                  next[swapIndex] = current;
                  const positions = Object.fromEntries(next.map((s, i) => [s.id, i + 1] as const));
                  reorder.mutate(positions);
                }}
              />
            ))}
          </div>
        </section>
      </div>
    </AppShell>
  );
}

function CreateSlideForm({
  nextPosition,
  onCreate,
  isSaving,
}: {
  nextPosition: number;
  onCreate: (payload: Draft & { media: File }) => void;
  isSaving: boolean;
}) {
  const [draft, setDraft] = useState<Draft>(() => blankDraft(nextPosition));
  const [media, setMedia] = useState<File | null>(null);

  useEffect(() => {
    setDraft((prev) => ({ ...prev, position: nextPosition || prev.position }));
  }, [nextPosition]);

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((current) => ({ ...current, [key]: value }));

  return (
    <section className="border border-brand/10 bg-white/60 p-6">
      <h2 className="font-serif text-xl mb-4">Create slide</h2>
      <div className="grid lg:grid-cols-[260px_1fr] gap-6">
        <div className="space-y-3">
          {media ? (
            <img
              src={URL.createObjectURL(media)}
              alt="Slide preview"
              className="w-full aspect-video object-cover bg-brand/5"
            />
          ) : (
            <div className="w-full aspect-video bg-brand/5 grid place-items-center text-xs text-brand/40">
              No image selected
            </div>
          )}
          <div>
            <Label htmlFor="create-media">Slide image</Label>
            <Input
              id="create-media"
              type="file"
              accept="image/jpeg,image/png,image/webp,image/gif"
              className="mt-2"
              onChange={(e) => setMedia(e.target.files?.[0] ?? null)}
            />
            <p className="mt-1 text-[11px] text-brand/45">JPEG, PNG, WebP, or GIF up to 5 MB.</p>
          </div>
          <Button
            type="button"
            onClick={() => {
              if (!media) {
                toast.error("Choose an image first");
                return;
              }
              if (!draft.title.trim()) {
                toast.error("Add a title first");
                return;
              }
              onCreate({
                ...draft,
                title: draft.title.trim(),
                subtitle: draft.subtitle.trim(),
                link_url: draft.link_url.trim(),
                media,
              });
              setDraft(blankDraft(nextPosition + 1));
              setMedia(null);
            }}
            disabled={isSaving}
            className="gap-2"
          >
            <Plus className="h-4 w-4" />
            {isSaving ? "Creating…" : "Create slide"}
          </Button>
        </div>

        <div className="space-y-4">
          <div className="grid sm:grid-cols-2 gap-4">
            <div>
              <Label>Title</Label>
              <Input
                value={draft.title}
                onChange={(e) => set("title", e.target.value)}
                className="mt-2"
              />
            </div>
            <div>
              <Label>Position</Label>
              <Input
                type="number"
                min={0}
                value={draft.position}
                onChange={(e) => set("position", Number(e.target.value) || 0)}
                className="mt-2"
              />
            </div>
          </div>
          <div>
            <Label>Subtitle</Label>
            <Textarea
              value={draft.subtitle}
              onChange={(e) => set("subtitle", e.target.value)}
              rows={3}
              className="mt-2"
            />
          </div>
          <div>
            <Label>Link URL</Label>
            <Input
              value={draft.link_url}
              onChange={(e) => set("link_url", e.target.value)}
              className="mt-2"
              placeholder="/courses"
            />
          </div>
          <div className="grid sm:grid-cols-2 gap-4">
            <div>
              <Label>Duration (ms)</Label>
              <Input
                type="number"
                min={2000}
                step={500}
                value={draft.duration_ms}
                onChange={(e) => set("duration_ms", Number(e.target.value) || 5000)}
                className="mt-2"
              />
            </div>
            <div className="flex items-center gap-3 sm:mt-7">
              <Switch
                id="create-active"
                checked={draft.is_active}
                onCheckedChange={(checked) => set("is_active", checked)}
              />
              <Label htmlFor="create-active">Active</Label>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function SlideEditor({
  slide,
  canMoveUp,
  canMoveDown,
  onMove,
}: {
  slide: Slide;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onMove: (direction: "up" | "down") => void;
}) {
  const qc = useQueryClient();
  const [draft, setDraft] = useState({
    title: slide.title ?? "",
    subtitle: slide.subtitle ?? "",
    link_url: slide.link_url ?? "",
    duration_ms: slide.duration_ms ?? 5000,
    is_active: slide.is_active ?? true,
  });
  const [media, setMedia] = useState<File | null>(null);

  useEffect(() => {
    setDraft({
      title: slide.title ?? "",
      subtitle: slide.subtitle ?? "",
      link_url: slide.link_url ?? "",
      duration_ms: slide.duration_ms ?? 5000,
      is_active: slide.is_active ?? true,
    });
    setMedia(null);
  }, [slide]);

  const update = useMutation({
    mutationFn: () =>
      adminUpdateSlide(slide.id, {
        title: draft.title.trim(),
        subtitle: draft.subtitle.trim(),
        link_url: draft.link_url.trim(),
        duration_ms: draft.duration_ms,
        is_active: draft.is_active,
        media: media ?? undefined,
      }),
    onSuccess: () => {
      toast.success("Slide saved");
      qc.invalidateQueries({ queryKey: ["admin-slides"] });
      qc.invalidateQueries({ queryKey: ["slides", "public"] });
      setMedia(null);
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const remove = useMutation({
    mutationFn: () => adminDeactivateSlide(slide.id),
    onSuccess: () => {
      toast.success("Slide deactivated");
      qc.invalidateQueries({ queryKey: ["admin-slides"] });
      qc.invalidateQueries({ queryKey: ["slides", "public"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <article className="border border-brand/10 bg-white/60 p-6 grid lg:grid-cols-[220px_1fr] gap-6">
      <div className="space-y-3">
        {slide.media_url ? (
          <img
            src={media ? URL.createObjectURL(media) : slide.media_url}
            alt={slide.title}
            className="w-full aspect-video object-cover bg-brand/5"
          />
        ) : (
          <div className="w-full aspect-video bg-brand/5 grid place-items-center text-xs text-brand/40">
            No preview
          </div>
        )}
        <div className="flex items-center gap-2">
          <Switch
            id={`active-${slide.id}`}
            checked={draft.is_active}
            onCheckedChange={(checked) =>
              setDraft((current) => ({ ...current, is_active: checked }))
            }
          />
          <Label htmlFor={`active-${slide.id}`} className="text-xs">
            {draft.is_active ? "Active" : "Hidden"}
          </Label>
        </div>
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            className="gap-2 flex-1"
            disabled={!canMoveUp || update.isPending}
            onClick={() => onMove("up")}
          >
            <ArrowUp className="h-4 w-4" />
            Up
          </Button>
          <Button
            type="button"
            variant="outline"
            className="gap-2 flex-1"
            disabled={!canMoveDown || update.isPending}
            onClick={() => onMove("down")}
          >
            <ArrowDown className="h-4 w-4" />
            Down
          </Button>
        </div>
      </div>

      <div className="space-y-4">
        <div className="grid sm:grid-cols-2 gap-4">
          <div>
            <Label>Title</Label>
            <Input
              value={draft.title}
              onChange={(e) => setDraft((current) => ({ ...current, title: e.target.value }))}
              className="mt-2"
            />
          </div>
          <div>
            <Label>Duration (ms)</Label>
            <Input
              type="number"
              min={2000}
              step={500}
              value={draft.duration_ms}
              onChange={(e) =>
                setDraft((current) => ({ ...current, duration_ms: Number(e.target.value) || 5000 }))
              }
              className="mt-2"
            />
          </div>
        </div>
        <div>
          <Label>Subtitle</Label>
          <Textarea
            value={draft.subtitle}
            onChange={(e) => setDraft((current) => ({ ...current, subtitle: e.target.value }))}
            rows={3}
            className="mt-2"
          />
        </div>
        <div className="grid sm:grid-cols-2 gap-4">
          <div>
            <Label>Link URL</Label>
            <Input
              value={draft.link_url}
              onChange={(e) => setDraft((current) => ({ ...current, link_url: e.target.value }))}
              className="mt-2"
            />
          </div>
          <div>
            <Label>Replace image</Label>
            <Input
              type="file"
              accept="image/jpeg,image/png,image/webp,image/gif"
              className="mt-2"
              onChange={(e) => setMedia(e.target.files?.[0] ?? null)}
            />
          </div>
        </div>

        <div className="flex justify-between pt-2">
          <Button
            type="button"
            variant="ghost"
            className="text-destructive gap-2"
            onClick={() => {
              if (confirm("Deactivate this slide?")) remove.mutate();
            }}
            disabled={remove.isPending}
          >
            <Trash2 className="h-4 w-4" />
            Deactivate
          </Button>
          <Button
            type="button"
            onClick={() => update.mutate()}
            disabled={update.isPending}
            className="gap-2"
          >
            <Save className="h-4 w-4" />
            {update.isPending ? "Saving…" : "Save changes"}
          </Button>
        </div>
      </div>
    </article>
  );
}

import { useCallback, useEffect, useImperativeHandle, useRef, useState, type Ref } from "react";

import type { PDFDocumentProxy, RenderTask } from "pdfjs-dist";

export type PdfViewerHandle = {
  goTo: (page: number) => void;
  zoom: (factor: number) => void;
};

type Props = {
  url: string;
  initialPage: number;
  onPageChange: (page: number) => void;
  onPageCount: (count: number) => void;
  onError: (error: Error) => void;
  ref?: Ref<PdfViewerHandle>;
};

/** How many pages around the visible one stay rendered. */
const KEEP_RENDERED = 3;

/**
 * Renders a PDF with PDF.js. The file is read with HTTP range requests
 * (disableAutoFetch), so opening page 200 of a large book downloads only the
 * pages shown. Pages are drawn as they scroll into view and dropped when far
 * away. The page with the most visible area is reported as the current page,
 * which the reader saves as the bookmark.
 */
export function PdfViewer({ url, initialPage, onPageChange, onPageCount, onError, ref }: Props) {
  const container = useRef<HTMLDivElement>(null);
  const [doc, setDoc] = useState<PDFDocumentProxy | null>(null);
  const [pageSize, setPageSize] = useState<{ width: number; height: number } | null>(null);
  const [scale, setScale] = useState(1);
  const [fitWidth, setFitWidth] = useState(800);
  const visible = useRef(new Map<number, number>());
  const rendered = useRef(new Map<number, { task?: RenderTask; scale: number }>());
  const current = useRef(Math.max(1, initialPage));
  const callbacks = useRef({ onPageChange, onPageCount, onError });
  callbacks.current = { onPageChange, onPageCount, onError };

  // Load the document (client only: pdfjs touches window/worker).
  useEffect(() => {
    const renderedPages = rendered.current;
    const visiblePages = visible.current;
    let cancelled = false;
    let loadingTask: { destroy: () => Promise<void> } | null = null;
    (async () => {
      try {
        const pdfjs = await import("pdfjs-dist");
        const worker = await import("pdfjs-dist/build/pdf.worker.min.mjs?url");
        pdfjs.GlobalWorkerOptions.workerSrc = worker.default;
        const task = pdfjs.getDocument({
          url,
          rangeChunkSize: 256 * 1024,
          disableAutoFetch: true,
          disableStream: false,
        });
        loadingTask = task;
        const loaded = await task.promise;
        if (cancelled) {
          void task.destroy();
          return;
        }
        const first = await loaded.getPage(1);
        const vp = first.getViewport({ scale: 1 });
        setPageSize({ width: vp.width, height: vp.height });
        setDoc(loaded);
        callbacks.current.onPageCount(loaded.numPages);
      } catch (error) {
        if (!cancelled) callbacks.current.onError(error as Error);
      }
    })();
    return () => {
      cancelled = true;
      renderedPages.forEach((r) => r.task?.cancel());
      renderedPages.clear();
      visiblePages.clear();
      void loadingTask?.destroy();
    };
  }, [url]);

  // Fit pages to the container width (capped for large screens).
  useEffect(() => {
    const el = container.current;
    if (!el) return;
    const measure = () => setFitWidth(Math.min(el.clientWidth - 16, 1000));
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const pageScale = pageSize ? (fitWidth / pageSize.width) * scale : 1;

  const renderPage = useCallback(
    async (n: number) => {
      if (!doc) return;
      const existing = rendered.current.get(n);
      if (existing && existing.scale === pageScale) return;
      existing?.task?.cancel();
      const canvas = container.current?.querySelector<HTMLCanvasElement>(
        `canvas[data-page="${n}"]`,
      );
      if (!canvas) return;
      const entry: { task?: RenderTask; scale: number } = { scale: pageScale };
      rendered.current.set(n, entry);
      try {
        const page = await doc.getPage(n);
        const ratio = Math.min(window.devicePixelRatio || 1, 2);
        const viewport = page.getViewport({ scale: pageScale * ratio });
        canvas.width = Math.floor(viewport.width);
        canvas.height = Math.floor(viewport.height);
        entry.task = page.render({ canvas, viewport });
        await entry.task.promise;
        canvas.dataset.rendered = "true";
      } catch (error) {
        if ((error as Error)?.name !== "RenderingCancelledException") {
          rendered.current.delete(n);
          callbacks.current.onError(error as Error);
        }
      }
    },
    [doc, pageScale],
  );

  const release = useCallback((n: number) => {
    const r = rendered.current.get(n);
    if (!r) return;
    r.task?.cancel();
    rendered.current.delete(n);
    const canvas = container.current?.querySelector<HTMLCanvasElement>(`canvas[data-page="${n}"]`);
    if (canvas) {
      canvas.width = 0;
      canvas.height = 0;
      delete canvas.dataset.rendered;
    }
  }, []);

  // Track visibility; render nearby pages, drop far ones, report the page.
  useEffect(() => {
    if (!doc || !container.current) return;
    const root = container.current;
    const io = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          const n = Number((e.target as HTMLElement).dataset.page);
          if (e.isIntersecting) visible.current.set(n, e.intersectionRatio);
          else visible.current.delete(n);
        }
        let best = current.current;
        let bestRatio = -1;
        visible.current.forEach((ratio, n) => {
          if (ratio > bestRatio) {
            best = n;
            bestRatio = ratio;
          }
        });
        if (bestRatio >= 0 && best !== current.current) {
          current.current = best;
          callbacks.current.onPageChange(best);
        }
        const keepFrom = current.current - KEEP_RENDERED;
        const keepTo = current.current + KEEP_RENDERED;
        for (let n = Math.max(1, keepFrom); n <= Math.min(doc.numPages, keepTo); n++)
          void renderPage(n);
        rendered.current.forEach((_v, n) => {
          if (n < keepFrom || n > keepTo) release(n);
        });
      },
      { root, threshold: [0, 0.25, 0.5, 0.75, 1] },
    );
    root.querySelectorAll<HTMLElement>("[data-page-slot]").forEach((el) => io.observe(el));
    return () => io.disconnect();
  }, [doc, renderPage, release]);

  // Re-render visible pages after a zoom or resize.
  useEffect(() => {
    if (!doc) return;
    const from = Math.max(1, current.current - KEEP_RENDERED);
    const to = Math.min(doc.numPages, current.current + KEEP_RENDERED);
    for (let n = from; n <= to; n++) void renderPage(n);
  }, [doc, pageScale, renderPage]);

  const goTo = useCallback(
    (n: number) => {
      if (!doc) return;
      const page = Math.min(doc.numPages, Math.max(1, Math.floor(n)));
      container.current
        ?.querySelector(`[data-page-slot="${page}"]`)
        ?.scrollIntoView({ block: "start" });
    },
    [doc],
  );

  // Open at the saved page once the slots exist.
  useEffect(() => {
    if (doc && pageSize) {
      const t = window.setTimeout(() => goTo(current.current), 0);
      return () => window.clearTimeout(t);
    }
  }, [doc, pageSize, goTo]);

  useImperativeHandle(
    ref,
    () => ({
      goTo,
      zoom: (factor: number) =>
        setScale((s) => Math.min(3, Math.max(0.5, +(s * factor).toFixed(2)))),
    }),
    [goTo],
  );

  const slotHeight = pageSize ? Math.round(pageSize.height * pageScale) : 0;
  const slotWidth = pageSize ? Math.round(pageSize.width * pageScale) : 0;

  return (
    <div
      ref={container}
      // Focusable so keyboard users can scroll the book with arrow keys.
      tabIndex={0}
      role="region"
      aria-label="Book pages"
      className="h-full w-full overflow-auto bg-brand/[0.04] py-4 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand/40"
      data-testid="pdf-viewer"
    >
      {doc &&
        pageSize &&
        Array.from({ length: doc.numPages }, (_, i) => i + 1).map((n) => (
          <div
            key={n}
            data-page-slot={n}
            data-page={n}
            className="mx-auto mb-4 bg-white shadow-sm"
            style={{ width: slotWidth, height: slotHeight, maxWidth: "100%" }}
          >
            <canvas
              data-page={n}
              width={0}
              height={0}
              role="img"
              aria-label={`Page ${n} of ${doc.numPages}`}
              style={{ width: slotWidth, height: slotHeight, maxWidth: "100%" }}
            />
          </div>
        ))}
    </div>
  );
}

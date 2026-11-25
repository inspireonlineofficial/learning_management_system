import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, ChevronLeft, ChevronRight, ExternalLink, ZoomIn, ZoomOut } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { PdfViewer, type PdfViewerHandle } from "@/components/bookshop/pdf-viewer";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { getBook, getReaderAccess, saveReadingPosition } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/student/bookshop/reader/$bookId")({
  component: ReaderPage,
});

/**
 * Reads a purchased book. The PDF is rendered in the page with PDF.js
 * (works on phones, and the page on screen is known, so the bookmark follows
 * the reader). The signed URL is renewed before it expires. If PDF.js can't
 * load the file (for example the storage bucket has no CORS rule for this
 * site), the browser's own PDF viewer is used instead.
 */
function ReaderPage() {
  const { bookId } = Route.useParams();
  const viewer = useRef<PdfViewerHandle>(null);

  const book = useQuery({
    queryKey: ["book", bookId],
    queryFn: () => getBook(bookId),
    retry: false,
  });
  const access = useQuery({
    queryKey: ["reader-access", bookId],
    queryFn: () => getReaderAccess(bookId),
    retry: false,
    refetchOnWindowFocus: false,
    refetchInterval: (q) => {
      const ttl = q.state.data?.expires_in ?? 0;
      return ttl > 600 ? (ttl - 300) * 1000 : false;
    },
  });

  const [page, setPage] = useState<number | null>(null);
  const [pageCount, setPageCount] = useState<number | null>(null);
  const [pageInput, setPageInput] = useState("");
  const [fallback, setFallback] = useState(false);
  const savedPage = useRef<number | null>(null);
  const startPage = useRef<number | null>(null);

  useEffect(() => {
    if (startPage.current === null && access.data) {
      const start = Math.max(1, access.data.last_page_read || 1);
      startPage.current = start;
      savedPage.current = access.data.last_page_read || 0;
      setPage(start);
      setPageInput(String(start));
    }
  }, [access.data]);

  const save = useMutation({
    mutationFn: (p: number) => saveReadingPosition(bookId, p),
    onSuccess: (_res, p) => {
      savedPage.current = p;
    },
  });

  // Save real changes only, debounced: scrolling through pages sends one request.
  useEffect(() => {
    if (page === null || page === savedPage.current) return;
    const t = window.setTimeout(() => save.mutate(page), 1000);
    return () => window.clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  const onViewerPage = (p: number) => {
    setPage(p);
    setPageInput(String(p));
  };

  const goTo = (p: number) => {
    const max = pageCount ?? 100000;
    const next = Math.min(max, Math.max(1, Math.floor(p)));
    setPage(next);
    setPageInput(String(next));
    viewer.current?.goTo(next);
  };

  const title = book.data?.title ?? "Reader";
  const url = access.data?.access_url;
  const fallbackSrc = url && page ? `${url}#page=${page}` : "";

  return (
    <div className="flex h-screen flex-col bg-surface text-brand font-sans">
      <header className="z-30 border-b border-brand/10 bg-surface/95 backdrop-blur">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-3 lg:px-6">
          <Link
            to="/student/bookshop/library"
            className="flex items-center gap-2 text-xs text-brand/70 hover:text-brand"
          >
            <ArrowLeft className="h-3.5 w-3.5" aria-hidden />
            Library
          </Link>
          <h1 className="max-w-[45%] truncate font-serif text-sm">{title}</h1>
          {url && page !== null && (
            <div className="flex items-center gap-1 sm:gap-2">
              <button
                type="button"
                onClick={() => goTo(page - 1)}
                disabled={page <= 1}
                aria-label="Previous page"
                className="p-1.5 text-brand/70 hover:text-brand disabled:opacity-30"
              >
                <ChevronLeft className="h-4 w-4" aria-hidden />
              </button>
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  const n = Number(pageInput);
                  if (Number.isFinite(n) && n >= 1) goTo(n);
                  else setPageInput(String(page));
                }}
                className="flex items-center gap-1 text-xs text-brand/70"
              >
                <label htmlFor="reader-page">Page</label>
                <input
                  id="reader-page"
                  inputMode="numeric"
                  value={pageInput}
                  onChange={(e) => setPageInput(e.target.value.replace(/[^0-9]/g, ""))}
                  onBlur={() => setPageInput(String(page))}
                  className="w-12 border border-brand/15 bg-white px-1.5 py-1 text-center text-sm"
                />
                {pageCount && <span aria-label={`of ${pageCount} pages`}>/ {pageCount}</span>}
              </form>
              <button
                type="button"
                onClick={() => goTo(page + 1)}
                disabled={pageCount !== null && page >= pageCount}
                aria-label="Next page"
                className="p-1.5 text-brand/70 hover:text-brand disabled:opacity-30"
              >
                <ChevronRight className="h-4 w-4" aria-hidden />
              </button>
              {!fallback && (
                <>
                  <button
                    type="button"
                    onClick={() => viewer.current?.zoom(1 / 1.2)}
                    aria-label="Zoom out"
                    className="hidden p-1.5 text-brand/70 hover:text-brand sm:block"
                  >
                    <ZoomOut className="h-4 w-4" aria-hidden />
                  </button>
                  <button
                    type="button"
                    onClick={() => viewer.current?.zoom(1.2)}
                    aria-label="Zoom in"
                    className="hidden p-1.5 text-brand/70 hover:text-brand sm:block"
                  >
                    <ZoomIn className="h-4 w-4" aria-hidden />
                  </button>
                </>
              )}
              <a
                href={fallbackSrc}
                target="_blank"
                rel="noreferrer"
                aria-label="Open the PDF in a new tab"
                className="ml-1 inline-flex items-center gap-1 text-xs text-brand/70 hover:text-brand"
              >
                <ExternalLink className="h-3.5 w-3.5" aria-hidden />
                <span className="hidden sm:inline">New tab</span>
              </a>
            </div>
          )}
        </div>
      </header>

      <main className="min-h-0 flex-1">
        {access.isLoading ? (
          <div className="m-6 h-[70vh] animate-pulse bg-brand/5" />
        ) : access.isError ? (
          <div className="mx-auto w-full max-w-xl px-6 py-16">
            <QueryErrorPanel
              error={access.error}
              title="This book can't be opened"
              onRetry={() => access.refetch()}
            />
            <p className="mt-4 text-sm text-brand/70">
              Books open after the purchase is approved.{" "}
              <Link to="/student/bookshop/$bookId" params={{ bookId }} className="underline">
                Back to the book
              </Link>
            </p>
          </div>
        ) : url && startPage.current !== null ? (
          fallback ? (
            <div className="flex h-full flex-col">
              <p className="px-4 py-2 text-center text-xs text-brand/70" role="status">
                Showing the browser&apos;s PDF viewer. Use the page box to move and save your place;
                if nothing appears, use “New tab”.
              </p>
              <iframe
                key={url}
                src={fallbackSrc}
                title={`${title} — PDF`}
                className="w-full flex-1 border-0 bg-white"
              />
            </div>
          ) : (
            <PdfViewer
              ref={viewer}
              url={url}
              initialPage={startPage.current}
              onPageChange={onViewerPage}
              onPageCount={setPageCount}
              onError={() => setFallback(true)}
            />
          )
        ) : null}
        {save.isError && (
          <p className="px-4 py-1 text-center text-xs text-destructive" role="alert">
            Your reading position couldn&apos;t be saved.
          </p>
        )}
      </main>
    </div>
  );
}

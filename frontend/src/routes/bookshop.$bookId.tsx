import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, BookOpen } from "lucide-react";

import { BookPurchasePanel } from "@/components/bookshop/book-purchase-panel";
import { PublicFooter } from "@/components/layout/public-footer";
import { PublicHeader } from "@/components/layout/public-header";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { getBook } from "@/lib/api/bookshop";

export const Route = createFileRoute("/bookshop/$bookId")({
  component: PublicBookDetailPage,
});

function PublicBookDetailPage() {
  const { bookId } = Route.useParams();
  const {
    data: book,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["book", bookId],
    queryFn: () => getBook(bookId),
  });

  return (
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy-light font-sans antialiased flex flex-col">
      <PublicHeader active="bookshop" />

      <main className="flex-1 max-w-6xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-12">
        <Link
          to="/bookshop"
          className="inline-flex items-center gap-2 min-h-11 text-sm font-semibold text-navy hover:text-bronze underline underline-offset-4 decoration-gold decoration-2 mb-6"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Back to bookshop
        </Link>

        {isLoading ? (
          <div className="h-96 rounded-xl border border-borderui bg-white animate-pulse" />
        ) : isError || !book ? (
          <QueryErrorPanel error={error} title="Book not found" onRetry={() => refetch()} />
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-[320px_1fr] gap-8 lg:gap-12 bg-white rounded-xl border border-borderui p-5 sm:p-8 lg:p-10 shadow-sm">
            <div className="aspect-[2/3] max-w-[280px] lg:max-w-none w-full mx-auto rounded-xl overflow-hidden bg-stoneband border border-borderui/60">
              {book.cover_url ? (
                <img src={book.cover_url} alt={book.title} className="h-full w-full object-cover" />
              ) : (
                <div className="h-full w-full grid place-items-center text-navy/25">
                  <BookOpen className="h-12 w-12" aria-hidden="true" />
                </div>
              )}
            </div>

            <article className="min-w-0">
              {book.category && (
                <p className="text-xs font-bold tracking-[0.12em] uppercase text-bronze mb-2.5">
                  {book.category}
                </p>
              )}
              <h1 className="font-serif text-3xl sm:text-4xl lg:text-5xl font-bold text-navy text-balance">
                {book.title}
              </h1>
              <p className="mt-3 text-slatebody text-base">by {book.author}</p>

              <div className="mt-8">
                <BookPurchasePanel book={book} returnTo={`/bookshop/${book.id}`} />
              </div>

              {book.description && (
                <section className="mt-10">
                  <h2 className="font-serif text-2xl font-bold text-navy mb-3">About this book</h2>
                  <p className="text-slatebody leading-relaxed whitespace-pre-wrap">
                    {book.description}
                  </p>
                </section>
              )}

              <dl className="mt-10 grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                {book.subject && <Meta label="Subject" value={book.subject} />}
                {book.class_grade && <Meta label="Class / grade" value={book.class_grade} />}
                {book.format && (
                  <Meta
                    label="Edition"
                    value={
                      book.format === "both"
                        ? "Digital and printed"
                        : book.format === "digital"
                          ? "Digital"
                          : "Printed"
                    }
                  />
                )}
                {book.isbn && <Meta label="ISBN" value={book.isbn} />}
              </dl>
            </article>
          </div>
        )}
      </main>

      <PublicFooter />
    </div>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-borderui bg-canvas p-4">
      <dt className="text-xs font-bold tracking-[0.12em] uppercase text-bronze">{label}</dt>
      <dd className="mt-1 text-navy font-semibold">{value}</dd>
    </div>
  );
}

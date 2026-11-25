import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, BookOpen } from "lucide-react";

import { BookPurchasePanel } from "@/components/bookshop/book-purchase-panel";
import { PublicHeader } from "@/components/layout/public-header";
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
  } = useQuery({
    queryKey: ["book", bookId],
    queryFn: () => getBook(bookId),
  });

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      <PublicHeader active="bookshop" />

      <main className="max-w-6xl mx-auto px-6 lg:px-10 py-12">
        <Link
          to="/bookshop"
          className="inline-flex items-center gap-2 text-xs text-brand/70 hover:text-brand mb-8"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Back to bookshop
        </Link>

        {isLoading ? (
          <div className="h-96 border border-brand/10 bg-white/30 animate-pulse" />
        ) : isError || !book ? (
          <p className="text-sm text-brand/60">{(error as Error)?.message ?? "Book not found"}</p>
        ) : (
          <div className="grid lg:grid-cols-[320px_1fr] gap-12">
            <div className="aspect-[2/3] bg-brand/10">
              {book.cover_url ? (
                <img src={book.cover_url} alt={book.title} className="h-full w-full object-cover" />
              ) : (
                <div className="h-full w-full grid place-items-center text-brand/30">
                  <BookOpen className="h-12 w-12" />
                </div>
              )}
            </div>

            <article>
              {book.category && <p className="eyebrow text-accent-ink mb-3">{book.category}</p>}
              <h1 className="font-serif text-4xl lg:text-5xl text-balance">{book.title}</h1>
              <p className="mt-3 text-brand/65">by {book.author}</p>

              <div className="mt-8">
                <BookPurchasePanel book={book} returnTo={`/bookshop/${book.id}`} />
              </div>

              {book.description && (
                <section className="mt-10">
                  <h2 className="font-serif text-2xl mb-3">About this book</h2>
                  <p className="text-brand/70 leading-relaxed whitespace-pre-wrap">
                    {book.description}
                  </p>
                </section>
              )}

              <dl className="mt-10 grid sm:grid-cols-2 gap-4 text-sm">
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
    </div>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-brand/10 bg-white/40 p-4">
      <dt className="eyebrow text-brand/65">{label}</dt>
      <dd className="mt-1">{value}</dd>
    </div>
  );
}

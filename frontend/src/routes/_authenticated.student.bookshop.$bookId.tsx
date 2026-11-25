import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { BookOpen } from "lucide-react";

import { BookPurchasePanel } from "@/components/bookshop/book-purchase-panel";
import { PublicHeader } from "@/components/layout/public-header";
import { getBook } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/student/bookshop/$bookId")({
  component: BookDetailPage,
});

function BookDetailPage() {
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

  if (isLoading) {
    return (
      <div className="max-w-5xl mx-auto px-6 lg:px-10 py-16">
        <div className="h-96 bg-brand/5 animate-pulse" />
      </div>
    );
  }

  if (isError || !book) {
    return (
      <div className="max-w-5xl mx-auto px-6 lg:px-10 py-16">
        <p className="text-sm text-brand/60">{(error as Error)?.message ?? "Not found"}</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      <PublicHeader active="bookshop" />

      <main className="max-w-5xl mx-auto px-6 lg:px-10 py-12 lg:py-16">
        <div className="grid lg:grid-cols-[280px_1fr] gap-12">
          <div className="aspect-[2/3] bg-brand/10">
            {book.cover_url ? (
              <img src={book.cover_url} alt={book.title} className="w-full h-full object-cover" />
            ) : (
              <div className="w-full h-full grid place-items-center text-brand/30">
                <BookOpen className="h-12 w-12" />
              </div>
            )}
          </div>

          <div>
            {book.category && <p className="eyebrow text-accent-ink mb-3">{book.category}</p>}
            <h1 className="font-serif text-4xl lg:text-5xl text-balance mb-3">{book.title}</h1>
            <p className="text-base text-brand/70 mb-6">by {book.author}</p>

            <div className="mb-10">
              <BookPurchasePanel book={book} returnTo={`/student/bookshop/${book.id}`} />
            </div>

            {book.description && (
              <section className="mb-10">
                <h2 className="eyebrow text-brand/65 mb-3">About this book</h2>
                <p className="text-brand/75 leading-relaxed whitespace-pre-wrap">
                  {book.description}
                </p>
              </section>
            )}

            <dl className="grid grid-cols-2 gap-4 text-sm border-t border-brand/10 pt-6">
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
              {book.class_grade && <Meta label="Class / grade" value={book.class_grade} />}
              {book.publisher && <Meta label="Publisher" value={book.publisher} />}
              {book.published_at && (
                <Meta label="Published" value={new Date(book.published_at).toLocaleDateString()} />
              )}
              {book.pages && <Meta label="Pages" value={String(book.pages)} />}
              {book.language && <Meta label="Language" value={book.language} />}
              {book.isbn && <Meta label="ISBN" value={book.isbn} />}
            </dl>

            {book.table_of_contents && book.table_of_contents.length > 0 && (
              <section className="mt-10">
                <h2 className="eyebrow text-brand/65 mb-3">Contents</h2>
                <ol className="divide-y divide-brand/10 border-y border-brand/10">
                  {book.table_of_contents.map((c, i) => (
                    <li key={c.id} className="flex items-center justify-between py-3 text-sm">
                      <span>
                        <span className="text-brand/65 mr-3">{String(i + 1).padStart(2, "0")}</span>
                        {c.title}
                      </span>
                      {c.page && <span className="text-xs text-brand/65">p. {c.page}</span>}
                    </li>
                  ))}
                </ol>
              </section>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="eyebrow text-brand/65">{label}</dt>
      <dd className="mt-1">{value}</dd>
    </div>
  );
}

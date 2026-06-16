import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, BookOpen, ShoppingBag } from "lucide-react";

import { useAuth } from "@/context/auth-context";
import { formatPrice, getBook } from "@/lib/api/bookshop";

export const Route = createFileRoute("/bookshop/$bookId")({
  component: PublicBookDetailPage,
});

function PublicBookDetailPage() {
  const { bookId } = Route.useParams();
  const { isAuthenticated } = useAuth();
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
      <header className="border-b border-brand/10">
        <div className="max-w-6xl mx-auto px-6 lg:px-10 py-6 flex items-center justify-between">
          <Link to="/" className="font-serif italic text-2xl text-accent">
            Inspire LMS
          </Link>
          <nav className="flex items-center gap-5 text-sm">
            <Link to="/courses" className="text-brand/65 hover:text-brand">
              Courses
            </Link>
            <Link to="/bookshop" className="text-brand/65 hover:text-brand">
              Bookshop
            </Link>
            <Link
              to={isAuthenticated ? "/student" : "/login"}
              className="bg-brand text-white px-4 py-2 text-xs"
            >
              {isAuthenticated ? "Dashboard" : "Sign in"}
            </Link>
          </nav>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 lg:px-10 py-12">
        <Link
          to="/bookshop"
          className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-8"
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
              {book.category && <p className="eyebrow text-accent mb-3">{book.category}</p>}
              <h1 className="font-serif text-4xl lg:text-5xl text-balance">{book.title}</h1>
              <p className="mt-3 text-brand/65">by {book.author}</p>

              <div className="mt-8 flex flex-wrap items-center gap-3">
                <span className="font-serif text-3xl">
                  {book.is_free ? "Free" : formatPrice(book.price_cents, book.currency)}
                </span>
                {book.format && (
                  <span className="border border-brand/15 px-3 py-1 text-xs capitalize text-brand/65">
                    {book.format}
                  </span>
                )}
                {typeof book.physical_stock === "number" && (
                  <span className="border border-brand/15 px-3 py-1 text-xs text-brand/65">
                    {book.physical_stock > 0 ? `${book.physical_stock} in stock` : "Out of stock"}
                  </span>
                )}
              </div>

              <div className="mt-8 flex flex-wrap gap-3">
                <Link
                  to={isAuthenticated ? "/student/bookshop/checkout/$itemId" : "/login"}
                  params={isAuthenticated ? { itemId: book.id } : undefined}
                  search={
                    isAuthenticated ? undefined : ({ return: `/bookshop/${book.id}` } as never)
                  }
                  className="inline-flex items-center gap-2 bg-brand text-white px-6 py-3 text-sm"
                >
                  <ShoppingBag className="h-4 w-4" />
                  {isAuthenticated ? "Request approval" : "Sign in to request"}
                </Link>
                <Link to="/register" className="border border-brand/15 px-6 py-3 text-sm">
                  Create account
                </Link>
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
                {book.language && <Meta label="Language" value={book.language} />}
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
      <dt className="eyebrow text-brand/45">{label}</dt>
      <dd className="mt-1">{value}</dd>
    </div>
  );
}

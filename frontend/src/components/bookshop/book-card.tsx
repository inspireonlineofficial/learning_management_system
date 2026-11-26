import { Link } from "@tanstack/react-router";

import { formatPrice, type BookSummary } from "@/lib/api/bookshop";

export function BookCard({ book }: { book: BookSummary }) {
  return (
    <Link
      to="/bookshop/$bookId"
      params={{ bookId: book.id }}
      className="group flex flex-col rounded-xl overflow-hidden border border-borderui bg-white p-3 shadow-sm hover:border-navy hover:-translate-y-0.5 hover:shadow-md transition-all focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
    >
      <div className="aspect-[2/3] rounded-lg bg-stoneband overflow-hidden">
        {book.cover_url ? (
          <img
            src={book.cover_url}
            alt={book.title}
            loading="lazy"
            className="w-full h-full object-cover group-hover:scale-[1.02] transition-transform duration-500"
          />
        ) : (
          <div className="w-full h-full grid place-items-center font-serif italic text-4xl text-navy/20">
            {book.title.slice(0, 1)}
          </div>
        )}
      </div>
      <div className="mt-3 flex flex-col flex-1">
        <p className="font-serif text-base font-bold text-navy leading-snug line-clamp-2 group-hover:text-bronze transition-colors">
          {book.title}
        </p>
        <p className="mt-1 text-xs text-slatebody">{book.author}</p>
        <p className="mt-1 text-[11px] font-semibold uppercase tracking-wider text-bronze/80">
          {book.category ?? "General science"}
          {book.format ? ` / ${book.format}` : ""}
        </p>
        <p className="mt-auto pt-2 text-sm font-semibold">
          {book.is_free || book.price_cents === 0 ? (
            <span className="text-bronze">Free</span>
          ) : (
            <span className="text-navy">{formatPrice(book.price_cents, book.currency)}</span>
          )}
        </p>
      </div>
    </Link>
  );
}

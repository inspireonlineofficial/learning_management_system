import { Link } from "@tanstack/react-router";

import { formatPrice, type BookSummary } from "@/lib/api/bookshop";

export function BookCard({ book }: { book: BookSummary }) {
  return (
    <Link to="/bookshop/$bookId" params={{ bookId: book.id }} className="group block">
      <div className="aspect-[2/3] bg-brand/5 overflow-hidden">
        {book.cover_url ? (
          <img
            src={book.cover_url}
            alt={book.title}
            loading="lazy"
            className="w-full h-full object-cover group-hover:scale-[1.02] transition-transform"
          />
        ) : (
          <div className="w-full h-full grid place-items-center font-serif italic text-4xl text-brand/20">
            {book.title.slice(0, 1)}
          </div>
        )}
      </div>
      <div className="mt-3">
        <p className="font-serif text-base leading-snug line-clamp-2 group-hover:text-accent transition-colors">
          {book.title}
        </p>
        <p className="mt-1 text-xs text-brand/55">{book.author}</p>
        <p className="mt-1 text-[11px] uppercase tracking-wider text-brand/40">
          {book.category ?? "General science"}
          {book.format ? ` / ${book.format}` : ""}
        </p>
        <p className="mt-2 text-sm">
          {book.is_free || book.price_cents === 0 ? (
            <span className="text-accent font-medium">Free</span>
          ) : (
            <span className="text-brand/80">{formatPrice(book.price_cents, book.currency)}</span>
          )}
        </p>
      </div>
    </Link>
  );
}

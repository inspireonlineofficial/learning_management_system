import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";

import { BookPurchasePanel } from "@/components/bookshop/book-purchase-panel";
import { AppShell } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { getBook } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/student/bookshop/checkout/$itemId")({
  component: Page,
});

// Kept for existing links (cart, emails): the same purchase panel as the
// book page, so the edition, delivery address and payment reference are
// collected instead of always requesting a digital copy.
function Page() {
  const { itemId } = Route.useParams();
  const book = useQuery({ queryKey: ["book", itemId], queryFn: () => getBook(itemId) });

  return (
    <AppShell eyebrow="Checkout" title={book.data ? book.data.title : "Get this book"}>
      {book.isError ? (
        <QueryErrorPanel
          error={book.error}
          title="Couldn't load this book"
          onRetry={() => book.refetch()}
        />
      ) : !book.data ? (
        <div className="h-48 max-w-xl bg-brand/5 animate-pulse" />
      ) : (
        <div className="max-w-xl">
          <p className="mb-6 text-sm text-brand/60">
            by {book.data.author} ·{" "}
            <Link
              to="/student/bookshop/$bookId"
              params={{ bookId: itemId }}
              className="underline hover:text-brand"
            >
              book details
            </Link>
          </p>
          <BookPurchasePanel book={book.data} returnTo={`/student/bookshop/checkout/${itemId}`} />
        </div>
      )}
    </AppShell>
  );
}

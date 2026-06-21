import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { ShoppingBag } from "lucide-react";
import { useEffect } from "react";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { formatPrice, getCart } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/student/bookshop/cart")({
  component: CartPage,
});

function CartPage() {
  const navigate = useNavigate();

  const {
    data: cart,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["cart"],
    queryFn: () => getCart(),
  });

  useEffect(() => {
    if (cart?.items.length === 1) {
      navigate({
        to: "/student/bookshop/checkout/$itemId",
        params: { itemId: cart.items[0].book.id },
        replace: true,
      });
    }
  }, [cart, navigate]);

  return (
    <AppShell eyebrow="Bookshop" title="Your cart.">
      {isError ? (
        <QueryErrorPanel error={error} title="Couldn't load cart" onRetry={() => refetch()} />
      ) : isLoading ? (
        <div className="h-40 bg-brand/5 animate-pulse" />
      ) : !cart || cart.items.length === 0 ? (
        <EmptyState
          icon={ShoppingBag}
          title="Approval requests are item by item"
          description="The current LMS workflow sends each course or book purchase to admin approval individually, so the cart is kept only for old links."
          action={
            <Link
              to="/student/bookshop"
              className="px-5 py-2.5 bg-brand text-white text-xs font-medium"
            >
              Browse books
            </Link>
          }
        />
      ) : (
        <div className="max-w-3xl">
          <div className="border border-accent/25 bg-accent/5 p-5 text-sm text-brand/70">
            Multi-item cart checkout is not part of the current approval workflow. Request approval
            for each book below, or review your existing requests.
          </div>
          <ul className="mt-6 divide-y divide-brand/10 border-y border-brand/10">
            {cart.items.map((item) => (
              <li key={item.id} className="flex flex-col gap-4 py-5 sm:flex-row">
                <Link
                  to="/student/bookshop/$bookId"
                  params={{ bookId: item.book.id }}
                  className="w-20 aspect-[2/3] bg-brand/10 shrink-0 overflow-hidden"
                >
                  {item.book.cover_url && (
                    <img
                      src={item.book.cover_url}
                      alt={item.book.title}
                      className="w-full h-full object-cover"
                    />
                  )}
                </Link>
                <div className="flex-1 min-w-0">
                  <Link
                    to="/student/bookshop/$bookId"
                    params={{ bookId: item.book.id }}
                    className="font-serif text-lg leading-snug hover:text-accent"
                  >
                    {item.book.title}
                  </Link>
                  <p className="text-xs text-brand/55 mt-1">{item.book.author}</p>
                  <p className="mt-3 text-xs text-brand/50">
                    Quantity {item.quantity} from an older cart session
                  </p>
                </div>
                <div className="shrink-0 sm:text-right">
                  <p className="font-serif text-lg">
                    {formatPrice(item.unit_price_cents * item.quantity, cart.currency)}
                  </p>
                  <Link
                    to="/student/bookshop/checkout/$itemId"
                    params={{ itemId: item.book.id }}
                    className="mt-3 inline-flex bg-brand px-4 py-2 text-xs font-medium text-white"
                  >
                    Request approval
                  </Link>
                </div>
              </li>
            ))}
          </ul>
          <div className="mt-6 flex flex-wrap gap-3">
            <Link to="/student/bookshop" className="border border-brand/15 px-5 py-2.5 text-sm">
              Continue browsing
            </Link>
            <Link
              to="/student/bookshop/requests"
              className="bg-brand px-5 py-2.5 text-sm text-white"
            >
              View approval requests
            </Link>
          </div>
        </div>
      )}
    </AppShell>
  );
}

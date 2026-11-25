import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { BookOpen, ShoppingBag, Trash2 } from "lucide-react";
import { useState, type FormEvent } from "react";
import { toast } from "sonner";

import { AddressFields } from "@/components/bookshop/address-fields";
import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import {
  checkoutCart,
  EMPTY_ADDRESS,
  formatPrice,
  getCart,
  removeCartItem,
  type CartCheckoutResult,
  type ShippingAddress,
} from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/student/bookshop/cart")({
  component: CartPage,
});

/**
 * The cart: one copy of each book and edition. Checkout places free items as
 * orders (all or none) and sends paid items to an admin as purchase requests
 * with one payment reference for the basket.
 */
function CartPage() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [address, setAddress] = useState<ShippingAddress>(EMPTY_ADDRESS);
  const [reference, setReference] = useState("");
  const [result, setResult] = useState<CartCheckoutResult | null>(null);

  const cart = useQuery({ queryKey: ["cart"], queryFn: getCart });

  const remove = useMutation({
    mutationFn: (id: string) => removeCartItem(id),
    onSuccess: (c) => qc.setQueryData(["cart"], c),
    onError: (e: Error) => toast.error(e.message),
  });

  const checkout = useMutation({
    mutationFn: () =>
      checkoutCart({
        shippingAddress: cart.data?.needs_address ? address : undefined,
        paymentReference: cart.data?.has_paid_items ? reference.trim() : undefined,
      }),
    onSuccess: (res) => {
      qc.setQueryData(["cart"], res.cart);
      for (const key of ["library", "my-orders", "my-book-requests"]) {
        qc.invalidateQueries({ queryKey: [key] });
      }
      setResult(res);
      if (res.failed.length === 0) toast.success("Checkout complete");
      else toast.warning("Some items couldn't be checked out");
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const data = cart.data;
  const unavailable = data?.items.some((i) => !i.available) ?? false;
  const submit = (e: FormEvent) => {
    e.preventDefault();
    checkout.mutate();
  };

  return (
    <AppShell eyebrow="Bookshop" title="Your cart">
      {result && (
        <section
          className="mb-8 border border-emerald-700/20 bg-emerald-50 p-5 text-sm"
          role="status"
        >
          <h2 className="font-serif text-lg">Checkout summary</h2>
          {result.orders.length > 0 && (
            <p className="mt-2">
              {result.orders.length} free {result.orders.length === 1 ? "order" : "orders"} placed.{" "}
              <Link to="/student/bookshop/orders" className="underline">
                View orders
              </Link>
            </p>
          )}
          {result.requests.length > 0 && (
            <p className="mt-1">
              {result.requests.length} purchase{" "}
              {result.requests.length === 1 ? "request" : "requests"} sent for admin approval.{" "}
              <Link to="/student/bookshop/requests" className="underline">
                View requests
              </Link>
            </p>
          )}
          {result.failed.map((f) => (
            <p key={f.item_id} className="mt-1 text-destructive">
              {f.title}: {f.error} (still in your cart)
            </p>
          ))}
        </section>
      )}

      {cart.isError ? (
        <QueryErrorPanel
          error={cart.error}
          title="Couldn't load your cart"
          onRetry={() => cart.refetch()}
        />
      ) : cart.isLoading ? (
        <div className="h-40 max-w-3xl animate-pulse bg-brand/5" />
      ) : !data || data.items.length === 0 ? (
        <EmptyState
          title="Your cart is empty"
          description="Add books from the bookshop to buy several at once."
          action={
            <button
              onClick={() => navigate({ to: "/student/bookshop" })}
              className="inline-flex items-center gap-2 bg-brand px-5 py-2.5 text-sm text-white"
            >
              <ShoppingBag className="h-4 w-4" aria-hidden />
              Browse books
            </button>
          }
        />
      ) : (
        <div className="grid max-w-5xl gap-8 lg:grid-cols-[1fr_360px]">
          <ul className="space-y-3">
            {data.items.map((item) => (
              <li
                key={item.id}
                className="flex gap-4 border border-brand/10 bg-white/50 p-4"
                data-testid="cart-item"
              >
                <div className="grid h-20 w-14 shrink-0 place-items-center overflow-hidden bg-brand/10">
                  {item.book.cover_url ? (
                    <img src={item.book.cover_url} alt="" className="h-full w-full object-cover" />
                  ) : (
                    <BookOpen className="h-5 w-5 text-brand/40" aria-hidden />
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <Link
                    to="/student/bookshop/$bookId"
                    params={{ bookId: item.book.id }}
                    className="font-serif text-base hover:underline"
                  >
                    {item.book.title}
                  </Link>
                  <p className="text-xs text-brand/70">
                    {item.format === "physical" ? "Printed copy" : "Digital edition"} ·{" "}
                    {item.book.author}
                  </p>
                  {!item.available && (
                    <p className="mt-1 text-xs text-destructive">{item.problem}</p>
                  )}
                </div>
                <div className="flex flex-col items-end justify-between">
                  <span className="text-sm">
                    {item.free ? "Free" : formatPrice(item.unit_price_cents, data.currency)}
                  </span>
                  <button
                    onClick={() => remove.mutate(item.id)}
                    disabled={remove.isPending}
                    aria-label={`Remove ${item.book.title} from the cart`}
                    className="p-1 text-brand/60 hover:text-destructive"
                  >
                    <Trash2 className="h-4 w-4" aria-hidden />
                  </button>
                </div>
              </li>
            ))}
          </ul>

          <form
            onSubmit={submit}
            className="h-fit space-y-5 border border-brand/10 bg-white/60 p-5"
          >
            <div className="flex items-baseline justify-between">
              <span className="eyebrow text-brand/70">Total</span>
              <span className="font-serif text-2xl">
                {data.total_cents === 0 ? "Free" : formatPrice(data.total_cents, data.currency)}
              </span>
            </div>

            {data.needs_address && <AddressFields value={address} onChange={setAddress} />}

            {data.has_paid_items && (
              <label className="block">
                <span className="eyebrow text-brand/70">Payment reference</span>
                <input
                  value={reference}
                  onChange={(e) => setReference(e.target.value)}
                  required
                  maxLength={120}
                  placeholder="bKash / Nagad transaction ID"
                  className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
                />
                <span className="mt-1 block text-xs text-brand/70">
                  Pay the total first. An admin checks the payment and approves each paid book.
                </span>
              </label>
            )}

            {unavailable && (
              <p className="text-xs text-destructive">
                Remove the items marked unavailable to continue.
              </p>
            )}
            <button
              type="submit"
              disabled={checkout.isPending || unavailable}
              className="w-full bg-brand px-5 py-3 text-sm text-white disabled:opacity-50"
            >
              {checkout.isPending
                ? "Checking out…"
                : data.has_paid_items
                  ? "Send purchase requests"
                  : "Place order"}
            </button>
          </form>
        </div>
      )}
    </AppShell>
  );
}

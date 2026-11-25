import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { BookOpen, Eye, Package, ShoppingBag } from "lucide-react";
import { useState, type FormEvent } from "react";
import { toast } from "sonner";

import { AddressFields } from "@/components/bookshop/address-fields";
import { useAuth } from "@/context/auth-context";
import {
  addToCart,
  formatPrice,
  getBookOwnership,
  getBookPreview,
  listMyBookRequests,
  orderFreeBook,
  requestBookPurchase,
  sellsDigital,
  sellsPrinted,
  EMPTY_ADDRESS,
  type BookDetail,
  type ShippingAddress,
} from "@/lib/api/bookshop";

type Format = "digital" | "physical";

/**
 * Everything a reader can do with one book: preview it, add a free book to
 * the library, order a free printed copy, or request a paid purchase (an
 * admin confirms the payment, then the order is created). Only actions the
 * backend can actually fulfil are offered.
 */
export function BookPurchasePanel({ book, returnTo }: { book: BookDetail; returnTo: string }) {
  const { isAuthenticated, user } = useAuth();
  const isStudent = isAuthenticated && user?.role === "student";
  const qc = useQueryClient();
  const navigate = useNavigate();

  const digital = sellsDigital(book);
  const printed = sellsPrinted(book);
  const [format, setFormat] = useState<Format>(digital ? "digital" : "physical");
  const [address, setAddress] = useState<ShippingAddress>(EMPTY_ADDRESS);
  const [note, setNote] = useState("");

  const ownership = useQuery({
    queryKey: ["book-ownership", book.id],
    queryFn: () => getBookOwnership(book.id),
    enabled: isStudent && digital,
  });
  const requests = useQuery({
    queryKey: ["my-book-requests"],
    queryFn: () => listMyBookRequests({ limit: 100 }),
    enabled: isStudent,
  });
  const latestRequest = requests.data?.data.find(
    (r) => r.item_id === book.id && (r.book_format ?? "digital") === format,
  );

  // The signed sample URL (valid 10 minutes) is fetched up front and shown as
  // a plain link: opening a tab after an await is blocked as a pop-up.
  const preview = useQuery({
    queryKey: ["book-preview", book.id],
    queryFn: () => getBookPreview(book.id),
    enabled: isAuthenticated && !!book.has_preview,
    staleTime: 8 * 60 * 1000,
    refetchInterval: 8 * 60 * 1000,
    retry: false,
  });

  const afterOrder = () => {
    qc.invalidateQueries({ queryKey: ["book-ownership", book.id] });
    qc.invalidateQueries({ queryKey: ["library"] });
    qc.invalidateQueries({ queryKey: ["my-orders"] });
    qc.invalidateQueries({ queryKey: ["book", book.id] });
  };
  const freeOrder = useMutation({
    mutationFn: () => orderFreeBook(book.id, format, format === "physical" ? address : undefined),
    onSuccess: (order) => {
      afterOrder();
      if (format === "digital") {
        toast.success("Added to your library");
        navigate({ to: "/student/bookshop/reader/$bookId", params: { bookId: book.id } });
      } else {
        toast.success("Order placed — we'll notify you when it ships");
        navigate({ to: "/student/bookshop/orders/$orderId", params: { orderId: order.id } });
      }
    },
    onError: (e: Error) => toast.error(e.message),
  });
  const purchaseRequest = useMutation({
    mutationFn: () =>
      requestBookPurchase({ bookId: book.id, format, shippingAddress: address, note: note.trim() }),
    onSuccess: (req) => {
      qc.invalidateQueries({ queryKey: ["my-book-requests"] });
      toast.success(
        req.status === "approved" ? "Already approved" : "Request sent — an admin will review it",
      );
      navigate({ to: "/student/bookshop/requests" });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const cartAdd = useMutation({
    mutationFn: () => addToCart(book.id, format),
    onSuccess: (cart) => {
      qc.setQueryData(["cart"], cart);
      toast.success("Added to your cart", {
        action: { label: "View cart", onClick: () => navigate({ to: "/student/bookshop/cart" }) },
      });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const ownsDigital = ownership.data?.in_library === true;
  const outOfStock = format === "physical" && (book.physical_stock ?? 0) <= 0;
  const digitalMissing = format === "digital" && book.has_digital_file === false;
  const pending = latestRequest?.status === "pending";
  const busy = freeOrder.isPending || purchaseRequest.isPending;

  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (book.is_free) freeOrder.mutate();
    else purchaseRequest.mutate();
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <span className="font-serif text-3xl">
          {book.is_free ? "Free" : formatPrice(book.price_cents, book.currency)}
        </span>
        {printed && (
          <span className="border border-brand/15 px-3 py-1 text-xs text-brand/65">
            {(book.physical_stock ?? 0) > 0
              ? `${book.physical_stock} printed ${book.physical_stock === 1 ? "copy" : "copies"} left`
              : "Printed copy out of stock"}
          </span>
        )}
      </div>

      {book.has_preview &&
        (isAuthenticated ? (
          preview.data ? (
            <a
              href={preview.data.preview_url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-2 border border-brand/15 px-5 py-2.5 text-sm hover:bg-brand/[0.03]"
            >
              <Eye className="h-4 w-4" aria-hidden />
              Read a free sample
            </a>
          ) : preview.isError ? (
            <p className="text-sm text-brand/65">The free sample couldn&apos;t be loaded.</p>
          ) : null
        ) : (
          <Link
            to="/login"
            search={{ return: returnTo } as never}
            className="inline-flex items-center gap-2 border border-brand/15 px-5 py-2.5 text-sm hover:bg-brand/[0.03]"
          >
            <Eye className="h-4 w-4" aria-hidden />
            Sign in to read a free sample
          </Link>
        ))}

      {!isAuthenticated ? (
        <div className="flex flex-wrap gap-3">
          <Link
            to="/login"
            search={{ return: returnTo } as never}
            className="inline-flex items-center gap-2 bg-brand text-white px-6 py-3 text-sm"
          >
            <ShoppingBag className="h-4 w-4" aria-hidden />
            Sign in to get this book
          </Link>
          <Link to="/register" className="border border-brand/15 px-6 py-3 text-sm">
            Create account
          </Link>
        </div>
      ) : !isStudent ? (
        <p className="text-sm text-brand/60">Books are purchased from a student account.</p>
      ) : (
        <form onSubmit={submit} className="space-y-5 border border-brand/10 bg-white/50 p-5">
          {digital && printed && (
            <fieldset>
              <legend className="eyebrow text-brand/65 mb-2">Edition</legend>
              <div className="flex flex-wrap gap-2">
                {(["digital", "physical"] as const).map((f) => (
                  <label
                    key={f}
                    className={`cursor-pointer border px-4 py-2 text-sm ${
                      format === f ? "border-brand bg-brand text-white" : "border-brand/15"
                    }`}
                  >
                    <input
                      type="radio"
                      name="format"
                      value={f}
                      checked={format === f}
                      onChange={() => setFormat(f)}
                      className="sr-only"
                    />
                    {f === "digital" ? "Digital (read online)" : "Printed copy (delivered)"}
                  </label>
                ))}
              </div>
            </fieldset>
          )}

          {format === "digital" && ownsDigital ? (
            <Link
              to="/student/bookshop/reader/$bookId"
              params={{ bookId: book.id }}
              className="inline-flex items-center gap-2 bg-brand text-white px-6 py-3 text-sm"
            >
              <BookOpen className="h-4 w-4" aria-hidden />
              Read now
            </Link>
          ) : digitalMissing ? (
            <p className="text-sm text-brand/60">The digital edition isn't available yet.</p>
          ) : outOfStock ? (
            <p className="text-sm text-brand/60">
              Printed copies are out of stock. Check back later.
            </p>
          ) : pending ? (
            <div className="border border-accent/25 bg-accent/5 p-4 text-sm text-brand/75">
              Your purchase request is waiting for admin approval (sent{" "}
              {new Date(latestRequest!.created_at).toLocaleDateString()}).{" "}
              <Link to="/student/bookshop/requests" className="underline">
                View requests
              </Link>
            </div>
          ) : (
            <>
              {latestRequest?.status === "rejected" && (
                <p className="border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive">
                  Your last request was declined
                  {latestRequest.rejection_reason ? `: ${latestRequest.rejection_reason}` : "."} You
                  can send a new one.
                </p>
              )}

              {format === "physical" && <AddressFields value={address} onChange={setAddress} />}

              {!book.is_free && (
                <label className="block">
                  <span className="eyebrow text-brand/65">Payment reference</span>
                  <input
                    value={note}
                    onChange={(e) => setNote(e.target.value)}
                    maxLength={200}
                    placeholder="e.g. bKash / Nagad transaction ID"
                    className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
                  />
                  <span className="mt-1 block text-xs text-brand/65">
                    An admin checks your payment and approves the purchase.
                  </span>
                </label>
              )}

              <div className="flex flex-wrap gap-3">
                <button
                  type="submit"
                  disabled={busy}
                  className="inline-flex items-center gap-2 bg-brand text-white px-6 py-3 text-sm disabled:opacity-50"
                >
                  {format === "physical" ? (
                    <Package className="h-4 w-4" aria-hidden />
                  ) : (
                    <ShoppingBag className="h-4 w-4" aria-hidden />
                  )}
                  {busy
                    ? "Sending…"
                    : book.is_free
                      ? format === "digital"
                        ? "Add to my library"
                        : "Order free copy"
                      : "Request purchase"}
                </button>
                <button
                  type="button"
                  onClick={() => cartAdd.mutate()}
                  disabled={cartAdd.isPending}
                  className="inline-flex items-center gap-2 border border-brand/15 px-5 py-3 text-sm hover:bg-white disabled:opacity-50"
                >
                  {cartAdd.isPending ? "Adding…" : "Add to cart"}
                </button>
              </div>
            </>
          )}
        </form>
      )}
    </div>
  );
}

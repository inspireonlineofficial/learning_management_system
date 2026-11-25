import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";

import { getCart } from "@/lib/api/bookshop";

const LINKS = [
  { to: "/student/bookshop/library", label: "My library" },
  { to: "/student/bookshop/orders", label: "Orders" },
  { to: "/student/bookshop/requests", label: "Purchase requests" },
] as const;

/** Links to the student's library, orders, requests and cart. */
export function BookshopNav() {
  const cart = useQuery({ queryKey: ["cart"], queryFn: getCart, staleTime: 30_000 });
  const count = cart.data?.items.length ?? 0;
  return (
    <nav aria-label="Bookshop" className="flex flex-wrap gap-2 text-sm">
      {LINKS.map((l) => (
        <Link
          key={l.to}
          to={l.to}
          className="border border-brand/15 bg-white/50 px-3 py-1.5 hover:bg-white"
        >
          {l.label}
        </Link>
      ))}
      <Link
        to="/student/bookshop/cart"
        className="border border-brand/15 bg-white/50 px-3 py-1.5 hover:bg-white"
      >
        Cart{count > 0 ? ` (${count})` : ""}
      </Link>
    </nav>
  );
}

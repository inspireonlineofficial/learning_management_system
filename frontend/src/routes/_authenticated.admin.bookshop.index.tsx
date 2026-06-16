import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { AppShell, EmptyState, SectionHeading } from "@/components/layout/app-shell";
import { formatPrice, listAdminBooks } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/admin/bookshop/")({
  component: BookshopAdminPage,
});

function BookshopAdminPage() {
  const books = useQuery({
    queryKey: ["admin-books"],
    queryFn: () => listAdminBooks({ limit: 100 }),
  });

  return (
    <AppShell eyebrow="Bookshop" title="Bookshop admin">
      <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4 max-w-5xl">
        <Link
          to="/admin/bookshop/new"
          className="border border-brand/10 bg-white/50 p-6 hover:bg-white"
        >
          <p className="font-serif text-lg">Add book</p>
          <p className="text-xs text-brand/55 mt-1">
            Create physical, digital, or combined catalog items.
          </p>
        </Link>
        <Link
          to="/admin/bookshop/orders"
          className="border border-brand/10 bg-white/50 p-6 hover:bg-white"
        >
          <p className="font-serif text-lg">All orders</p>
          <p className="text-xs text-brand/55 mt-1">View and manage every order on the platform.</p>
        </Link>
        <Link
          to="/admin/bookshop/refunds"
          className="border border-brand/10 bg-white/50 p-6 hover:bg-white"
        >
          <p className="font-serif text-lg">Refunds</p>
          <p className="text-xs text-brand/55 mt-1">Process refund requests.</p>
        </Link>
      </div>

      <SectionHeading
        title="Inventory"
        action={
          <Link to="/admin/bookshop/new" className="text-xs bg-brand text-white px-3 py-2">
            Add book
          </Link>
        }
      />

      {books.isLoading ? (
        <div className="grid gap-2">
          {Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="h-16 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : books.isError ? (
        <div className="border border-destructive/20 bg-destructive/5 p-6 text-sm">
          <p className="font-medium text-destructive">Couldn't load inventory</p>
          <p className="mt-1 text-brand/60">{(books.error as Error).message}</p>
        </div>
      ) : !books.data || books.data.data.length === 0 ? (
        <EmptyState title="No books yet" />
      ) : (
        <div className="border border-brand/10 bg-white/40 overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="border-b border-brand/10 bg-brand/[0.02]">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-brand/60 eyebrow">Book</th>
                <th className="text-left px-4 py-3 font-medium text-brand/60 eyebrow">Subject</th>
                <th className="text-left px-4 py-3 font-medium text-brand/60 eyebrow">Format</th>
                <th className="text-left px-4 py-3 font-medium text-brand/60 eyebrow">Stock</th>
                <th className="text-left px-4 py-3 font-medium text-brand/60 eyebrow">Price</th>
                <th className="text-right px-4 py-3 font-medium text-brand/60 eyebrow">Action</th>
              </tr>
            </thead>
            <tbody>
              {books.data.data.map((book) => (
                <tr key={book.id} className="border-b border-brand/5 last:border-b-0">
                  <td className="px-4 py-3">
                    <p className="font-medium text-brand">{book.title}</p>
                    <p className="text-xs text-brand/55">{book.author}</p>
                  </td>
                  <td className="px-4 py-3 text-brand/65">{book.category ?? "—"}</td>
                  <td className="px-4 py-3 text-brand/65">{book.format ?? "—"}</td>
                  <td className="px-4 py-3 text-brand/65">{book.physical_stock ?? "—"}</td>
                  <td className="px-4 py-3 text-brand/65">
                    {formatPrice(book.price_cents, book.currency)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Link
                      to="/admin/bookshop/$bookId/edit"
                      params={{ bookId: book.id }}
                      className="text-xs border border-brand/15 px-3 py-1.5 hover:bg-brand/[0.03]"
                    >
                      Edit
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </AppShell>
  );
}

import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { useEffect, useState } from "react";

import { AppShell } from "@/components/layout/app-shell";
import { getBook, updateAdminBook, type AdminBookInput } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/admin/bookshop/$bookId/edit")({
  component: EditBookPage,
});

function EditBookPage() {
  const { bookId } = Route.useParams();
  const navigate = useNavigate();
  const book = useQuery({ queryKey: ["book", bookId], queryFn: () => getBook(bookId) });
  const [form, setForm] = useState<AdminBookInput & { title: string; author: string }>({
    title: "",
    author: "",
    subject: "",
    class_grade: "",
    description: "",
    price: 0,
    physical_stock: 0,
    is_active: true,
  });

  useEffect(() => {
    if (!book.data) return;
    setForm({
      title: book.data.title,
      author: book.data.author,
      subject: book.data.subject ?? book.data.category ?? "",
      class_grade: book.data.class_grade ?? "",
      description: book.data.description ?? "",
      price: book.data.price_cents / 100,
      currency: book.data.currency,
      physical_stock: book.data.physical_stock ?? 0,
      is_active: book.data.is_active ?? true,
    });
  }, [book.data]);

  const update = useMutation({
    mutationFn: () => updateAdminBook(bookId, form),
    onSuccess: () => {
      toast.success("Book updated");
      navigate({ to: "/admin/bookshop" });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <AppShell eyebrow="Bookshop" title={book.data?.title ? `Edit ${book.data.title}` : "Edit book"}>
      {book.isLoading ? (
        <div className="h-64 border border-brand/10 bg-white/30 animate-pulse" />
      ) : book.isError ? (
        <p className="border border-destructive/20 bg-destructive/5 p-6 text-sm text-destructive">
          {(book.error as Error)?.message}
        </p>
      ) : (
        <div className="max-w-3xl border border-brand/10 bg-white/50 p-6 space-y-5">
          <div className="grid sm:grid-cols-2 gap-4">
            <TextField
              label="Title"
              value={form.title}
              onChange={(title) => setForm({ ...form, title })}
            />
            <TextField
              label="Author"
              value={form.author}
              onChange={(author) => setForm({ ...form, author })}
            />
            <TextField
              label="Subject"
              value={form.subject ?? ""}
              onChange={(subject) => setForm({ ...form, subject })}
            />
            <TextField
              label="Class / grade"
              value={form.class_grade ?? ""}
              onChange={(class_grade) => setForm({ ...form, class_grade })}
            />
          </div>
          <label className="block">
            <span className="eyebrow text-brand/45">Description</span>
            <textarea
              value={form.description ?? ""}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
              rows={5}
              className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
          <div className="grid sm:grid-cols-3 gap-4">
            <NumberField
              label="Price"
              value={form.price ?? 0}
              onChange={(price) => setForm({ ...form, price })}
            />
            <NumberField
              label="Stock"
              value={form.physical_stock ?? 0}
              onChange={(physical_stock) => setForm({ ...form, physical_stock })}
            />
            <label className="block">
              <span className="eyebrow text-brand/45">Visibility</span>
              <select
                value={form.is_active ? "active" : "inactive"}
                onChange={(event) =>
                  setForm({ ...form, is_active: event.target.value === "active" })
                }
                className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              >
                <option value="active">Active</option>
                <option value="inactive">Hidden</option>
              </select>
            </label>
          </div>
          <button
            onClick={() => update.mutate()}
            disabled={!form.title.trim() || !form.author.trim() || update.isPending}
            className="bg-brand text-white px-6 py-3 text-sm disabled:opacity-50"
          >
            {update.isPending ? "Saving..." : "Save changes"}
          </button>
        </div>
      )}
    </AppShell>
  );
}

function TextField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/45">{label}</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
      />
    </label>
  );
}

function NumberField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (value: number) => void;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/45">{label}</span>
      <input
        type="number"
        min={0}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
        className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
      />
    </label>
  );
}

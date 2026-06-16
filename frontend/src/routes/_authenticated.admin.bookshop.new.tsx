import { useMutation } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { useState } from "react";

import { AppShell } from "@/components/layout/app-shell";
import { createAdminBook, type AdminBookInput } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/admin/bookshop/new")({
  component: CreateBookPage,
});

function CreateBookPage() {
  const navigate = useNavigate();
  const [form, setForm] = useState<
    Required<Pick<AdminBookInput, "title" | "author">> & AdminBookInput
  >({
    title: "",
    author: "",
    subject: "",
    class_grade: "",
    description: "",
    format: "both",
    price: 0,
    currency: "BDT",
    physical_stock: 0,
  });

  const create = useMutation({
    mutationFn: () => createAdminBook(form),
    onSuccess: () => {
      toast.success("Book added");
      navigate({ to: "/admin/bookshop" });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <AppShell eyebrow="Bookshop" title="Add a book">
      <BookForm
        form={form}
        setForm={setForm}
        actionLabel={create.isPending ? "Saving..." : "Add book"}
        disabled={create.isPending}
        onSubmit={() => create.mutate()}
      />
    </AppShell>
  );
}

function BookForm({
  form,
  setForm,
  actionLabel,
  disabled,
  onSubmit,
}: {
  form: AdminBookInput & { title: string; author: string };
  setForm: (form: AdminBookInput & { title: string; author: string }) => void;
  actionLabel: string;
  disabled?: boolean;
  onSubmit: () => void;
}) {
  return (
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
      <div className="grid sm:grid-cols-4 gap-4">
        <label className="block">
          <span className="eyebrow text-brand/45">Format</span>
          <select
            value={form.format ?? "both"}
            onChange={(event) =>
              setForm({ ...form, format: event.target.value as "physical" | "digital" | "both" })
            }
            className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
          >
            <option value="physical">Physical</option>
            <option value="digital">Digital</option>
            <option value="both">Both</option>
          </select>
        </label>
        <NumberField
          label="Price"
          value={form.price ?? 0}
          onChange={(price) => setForm({ ...form, price })}
        />
        <TextField
          label="Currency"
          value={form.currency ?? "BDT"}
          onChange={(currency) => setForm({ ...form, currency })}
        />
        <NumberField
          label="Stock"
          value={form.physical_stock ?? 0}
          onChange={(physical_stock) => setForm({ ...form, physical_stock })}
        />
      </div>
      <button
        onClick={onSubmit}
        disabled={!form.title.trim() || !form.author.trim() || disabled}
        className="bg-brand text-white px-6 py-3 text-sm disabled:opacity-50"
      >
        {actionLabel}
      </button>
    </div>
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

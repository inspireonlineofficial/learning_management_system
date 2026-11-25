import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRef } from "react";
import { toast } from "sonner";

import { uploadBookAsset, type BookDetail } from "@/lib/api/bookshop";

type Kind = "file" | "preview" | "cover";

const ROWS: Array<{ kind: Kind; label: string; accept: string; help: string }> = [
  {
    kind: "file",
    label: "Digital edition (PDF)",
    accept: "application/pdf",
    help: "The full book students read after purchase. PDF, up to 100 MB.",
  },
  {
    kind: "preview",
    label: "Free sample (PDF)",
    accept: "application/pdf",
    help: "A few pages anyone signed in can read. PDF, up to 20 MB.",
  },
  {
    kind: "cover",
    label: "Cover image",
    accept: "image/jpeg,image/png,image/webp",
    help: "JPEG, PNG or WebP, up to 2 MB.",
  },
];

/** Upload controls for a book's files. Files are private; students only get short-lived links. */
export function BookFilesEditor({ book }: { book: BookDetail }) {
  return (
    <section className="space-y-3 border-t border-brand/10 pt-5">
      <h2 className="eyebrow text-brand/65">Files</h2>
      {ROWS.map((row) => (
        <UploadRow key={row.kind} book={book} {...row} />
      ))}
    </section>
  );
}

function UploadRow({
  book,
  kind,
  label,
  accept,
  help,
}: {
  book: BookDetail;
  kind: Kind;
  label: string;
  accept: string;
  help: string;
}) {
  const qc = useQueryClient();
  const input = useRef<HTMLInputElement>(null);
  const mut = useMutation({
    mutationFn: (file: File) => uploadBookAsset(book.id, kind, file),
    onSuccess: (updated) => {
      toast.success(`${label} uploaded`);
      qc.setQueryData(["admin-book", book.id], updated);
      qc.invalidateQueries({ queryKey: ["book", book.id] });
    },
    onError: (e: Error) => toast.error(e.message),
    onSettled: () => {
      if (input.current) input.current.value = "";
    },
  });
  const present =
    kind === "file"
      ? book.has_digital_file
      : kind === "preview"
        ? book.has_preview
        : !!book.cover_url;
  const id = `upload-${kind}`;

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border border-brand/10 bg-white/60 p-4">
      <div className="min-w-0">
        <label htmlFor={id} className="text-sm font-medium">
          {label}
        </label>
        <p className="text-xs text-brand/70">{help}</p>
        <p className={`mt-1 text-xs ${present ? "text-emerald-700" : "text-brand/65"}`}>
          {present ? "Uploaded" : "Not uploaded yet"}
          {kind === "file" && !present && book.format !== "physical"
            ? " — students can't buy the digital edition until it is."
            : ""}
        </p>
      </div>
      <input
        id={id}
        ref={input}
        type="file"
        accept={accept}
        disabled={mut.isPending}
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) mut.mutate(file);
        }}
        className="text-xs file:mr-3 file:border file:border-brand/15 file:bg-white file:px-3 file:py-1.5 file:text-xs"
      />
      {mut.isPending && <span className="text-xs text-brand/70">Uploading…</span>}
    </div>
  );
}

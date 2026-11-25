import { useState, type FormEvent } from "react";

import { formatPrice, payoutMethodLabel, type PayoutMethod } from "@/lib/api/bookshop";
import type { PayoutInput } from "@/lib/api/refunds";

const METHODS: PayoutMethod[] = ["bkash", "nagad", "rocket", "bank", "cash", "other"];
const NEEDS_REFERENCE: PayoutMethod[] = ["bkash", "nagad", "rocket", "bank"];

/**
 * Records how a refund was paid back. The LMS does not move money; the
 * admin sends the refund first and enters the transaction ID here so it can
 * be reconciled later.
 */
export function RefundPayoutForm({
  amount,
  currency,
  submitLabel,
  busy,
  onSubmit,
  onCancel,
}: {
  amount: number;
  currency: string;
  submitLabel: string;
  busy: boolean;
  onSubmit: (payout: PayoutInput) => void;
  onCancel: () => void;
}) {
  const free = amount <= 0;
  const [method, setMethod] = useState<PayoutMethod>("bkash");
  const [reference, setReference] = useState("");
  const [partial, setPartial] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (free) {
      onSubmit({ payoutMethod: "none", note });
      return;
    }
    let value: number | undefined;
    if (partial.trim()) {
      value = Number(partial);
      if (!Number.isFinite(value) || value <= 0 || value > amount) {
        setError(`Amount must be more than 0 and at most ${amount}.`);
        return;
      }
    }
    setError("");
    onSubmit({ payoutMethod: method, payoutReference: reference.trim(), amount: value, note });
  };

  return (
    <form onSubmit={submit} className="mt-4 space-y-3 border-t border-brand/10 pt-4">
      {free ? (
        <p className="text-sm text-brand/70">Free order: nothing to pay back.</p>
      ) : (
        <>
          <p className="text-xs text-brand/70">
            Send the refund first, then record it here. Order total{" "}
            {formatPrice(Math.round(amount * 100), currency)}.
          </p>
          <div className="grid gap-3 sm:grid-cols-3">
            <label className="block">
              <span className="eyebrow text-brand/70">Paid back by</span>
              <select
                value={method}
                onChange={(e) => setMethod(e.target.value as PayoutMethod)}
                className="mt-1 w-full border border-brand/15 bg-white px-2 py-2 text-sm"
              >
                {METHODS.map((m) => (
                  <option key={m} value={m}>
                    {payoutMethodLabel[m]}
                  </option>
                ))}
              </select>
            </label>
            <label className="block">
              <span className="eyebrow text-brand/70">
                Transaction ID{NEEDS_REFERENCE.includes(method) ? "" : " (optional)"}
              </span>
              <input
                value={reference}
                onChange={(e) => setReference(e.target.value)}
                required={NEEDS_REFERENCE.includes(method)}
                maxLength={200}
                className="mt-1 w-full border border-brand/15 bg-white px-2 py-2 text-sm"
              />
            </label>
            <label className="block">
              <span className="eyebrow text-brand/70">Amount (blank = full)</span>
              <input
                value={partial}
                onChange={(e) => setPartial(e.target.value.replace(/[^0-9.]/g, ""))}
                inputMode="decimal"
                placeholder={String(amount)}
                className="mt-1 w-full border border-brand/15 bg-white px-2 py-2 text-sm"
              />
            </label>
          </div>
        </>
      )}
      <label className="block">
        <span className="eyebrow text-brand/70">Note to the student (optional)</span>
        <input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          maxLength={2000}
          className="mt-1 w-full border border-brand/15 bg-white px-2 py-2 text-sm"
        />
      </label>
      {error && (
        <p className="text-xs text-destructive" role="alert">
          {error}
        </p>
      )}
      <div className="flex gap-2">
        <button
          type="submit"
          disabled={busy}
          className="bg-brand px-4 py-2 text-xs text-white disabled:opacity-50"
        >
          {busy ? "Saving…" : submitLabel}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="border border-brand/15 px-4 py-2 text-xs"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}

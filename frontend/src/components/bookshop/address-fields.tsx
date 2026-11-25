import type { ShippingAddress } from "@/lib/api/bookshop";

/** Delivery address inputs for printed copies. */
export function AddressFields({
  value,
  onChange,
}: {
  value: ShippingAddress;
  onChange: (value: ShippingAddress) => void;
}) {
  const field = (
    key: keyof ShippingAddress,
    label: string,
    opts: { required?: boolean; type?: string; autoComplete?: string } = {},
  ) => (
    <label className="block">
      <span className="eyebrow text-brand/65">
        {label}
        {opts.required ? "" : " (optional)"}
      </span>
      <input
        value={value[key] ?? ""}
        onChange={(e) => onChange({ ...value, [key]: e.target.value })}
        required={opts.required}
        type={opts.type ?? "text"}
        autoComplete={opts.autoComplete}
        maxLength={200}
        className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
      />
    </label>
  );
  return (
    <fieldset className="space-y-3">
      <legend className="eyebrow text-brand/65 mb-1">Delivery address</legend>
      <div className="grid gap-3 sm:grid-cols-2">
        {field("name", "Recipient name", { required: true, autoComplete: "name" })}
        {field("phone", "Phone", { required: true, type: "tel", autoComplete: "tel" })}
      </div>
      {field("line1", "Address", { required: true, autoComplete: "address-line1" })}
      {field("line2", "Area / landmark", { autoComplete: "address-line2" })}
      <div className="grid gap-3 sm:grid-cols-2">
        {field("city", "City / district", { required: true, autoComplete: "address-level2" })}
        {field("postal_code", "Postal code", { autoComplete: "postal-code" })}
      </div>
    </fieldset>
  );
}

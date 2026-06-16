import type { ReactNode } from "react";

import { AppShell } from "@/components/layout/app-shell";

/**
 * Shared scaffold for a not-yet-fully-implemented page.
 * Keeps the route file tiny while still mounting under the AppShell.
 */
export function PlaceholderPage({
  eyebrow,
  title,
  description,
  children,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  children?: ReactNode;
}) {
  return (
    <AppShell eyebrow={eyebrow} title={title}>
      {description && <p className="max-w-2xl text-brand/65 leading-relaxed">{description}</p>}
      {children}
      {!children && (
        <div className="mt-10 border border-dashed border-brand/15 px-8 py-16 text-center text-sm text-brand/55">
          This view is wired to the API layer. Detailed UI is coming online.
        </div>
      )}
    </AppShell>
  );
}

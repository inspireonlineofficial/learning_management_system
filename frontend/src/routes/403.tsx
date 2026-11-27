import { useEffect, useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";

import { isImpersonating, stopImpersonation } from "@/lib/session";

export const Route = createFileRoute("/403")({
  head: () => ({
    meta: [
      { title: "Access denied — Inspire Online" },
      { name: "description", content: "You don't have permission to view this page." },
    ],
  }),
  component: ForbiddenPage,
});

function ForbiddenPage() {
  const [impersonating, setImpersonating] = useState(false);

  useEffect(() => {
    setImpersonating(isImpersonating());
  }, []);

  const handleExitImpersonation = () => {
    const origin = stopImpersonation({ silent: true });
    if (origin) {
      window.location.assign("/admin/users");
    }
  };

  return (
    <div className="min-h-screen grid place-items-center bg-canvas text-navy font-sans px-4 py-12">
      <div className="max-w-md w-full rounded-xl border border-borderui bg-white p-8 sm:p-10 text-center shadow-sm">
        <p className="text-xs font-bold tracking-[0.12em] uppercase text-bronze">403</p>
        <h1 className="mt-3 font-serif text-4xl sm:text-5xl font-bold text-navy">Access denied</h1>
        <p className="mt-4 text-sm sm:text-base text-slatebody leading-relaxed">
          You don't have permission to view this part of the platform. If you believe this is in
          error, please contact your administrator.
        </p>
        <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-3">
          {impersonating && (
            <button
              type="button"
              onClick={handleExitImpersonation}
              className="inline-flex items-center justify-center min-h-11 rounded-lg bg-bronze px-6 py-3 text-sm font-semibold text-white hover:opacity-90 shadow-sm transition-opacity"
            >
              Exit impersonation
            </button>
          )}
          <Link
            to="/"
            className="inline-flex items-center justify-center min-h-11 rounded-lg bg-navy px-6 py-3 text-sm font-semibold text-white hover:bg-navy-light shadow-sm transition-colors"
          >
            Return to home
          </Link>
        </div>
      </div>
    </div>
  );
}

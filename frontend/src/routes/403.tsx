import { createFileRoute, Link } from "@tanstack/react-router";

export const Route = createFileRoute("/403")({
  head: () => ({
    meta: [
      { title: "Access denied — Inspire" },
      { name: "description", content: "You don't have permission to view this page." },
    ],
  }),
  component: ForbiddenPage,
});

function ForbiddenPage() {
  return (
    <div className="min-h-screen grid place-items-center bg-surface text-brand font-sans px-4">
      <div className="max-w-md text-center">
        <p className="eyebrow text-accent">403</p>
        <h1 className="mt-4 font-serif text-5xl">Access denied</h1>
        <p className="mt-4 text-brand/55 leading-relaxed">
          You don't have permission to view this part of the library. If you believe this is in
          error, please contact your administrator.
        </p>
        <div className="mt-8">
          <Link
            to="/"
            className="inline-flex items-center justify-center bg-brand px-6 py-3 font-serif text-base text-white hover:bg-brand/90 transition-colors"
          >
            Return to library
          </Link>
        </div>
      </div>
    </div>
  );
}

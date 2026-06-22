import { createFileRoute, Link } from "@tanstack/react-router";

export const Route = createFileRoute("/$")({
  head: () => ({
    meta: [
      { title: "Not found — Inspire" },
      { name: "description", content: "The page you're looking for can't be found." },
    ],
  }),
  component: NotFoundPage,
});

function NotFoundPage() {
  return (
    <div className="min-h-screen grid place-items-center bg-surface text-brand font-sans px-4">
      <div className="max-w-md text-center">
        <p className="eyebrow text-accent">404</p>
        <h1 className="mt-4 font-serif text-5xl">Page not found</h1>
        <p className="mt-4 text-brand/55 leading-relaxed">
          The page you're looking for has been moved, or it never existed in our archive.
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

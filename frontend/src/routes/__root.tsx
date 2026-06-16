import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  Outlet,
  Link,
  createRootRouteWithContext,
  useRouter,
  HeadContent,
  Scripts,
} from "@tanstack/react-router";
import { useEffect, type ReactNode } from "react";
import { Toaster } from "sonner";

import appCss from "../styles.css?url";
import { reportLovableError } from "../lib/lovable-error-reporting";
import { AuthProvider } from "../context/auth-context";

function NotFoundComponent() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-surface px-4 font-sans text-brand">
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

function ErrorComponent({ error, reset }: { error: Error; reset: () => void }) {
  console.error(error);
  const router = useRouter();
  useEffect(() => {
    reportLovableError(error, { boundary: "tanstack_root_error_component" });
  }, [error]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-surface px-4 font-sans text-brand">
      <div className="max-w-md text-center">
        <p className="eyebrow text-destructive">Error</p>
        <h1 className="mt-4 font-serif text-4xl">Something went wrong</h1>
        <p className="mt-4 text-sm text-brand/55 leading-relaxed">
          We hit an unexpected problem loading this page. You can try again or head home.
        </p>
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          <button
            onClick={() => {
              router.invalidate();
              reset();
            }}
            className="inline-flex items-center justify-center bg-brand px-5 py-3 text-sm font-medium text-white hover:bg-brand/90 transition-colors"
          >
            Try again
          </button>
          <a
            href="/"
            className="inline-flex items-center justify-center border border-brand/15 bg-transparent px-5 py-3 text-sm font-medium text-brand hover:bg-brand/[0.03] transition-colors"
          >
            Go home
          </a>
        </div>
      </div>
    </div>
  );
}

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  head: () => ({
    meta: [
      { charSet: "utf-8" },
      { name: "viewport", content: "width=device-width, initial-scale=1" },
      { title: "Inspire LMS — Science tutoring for high school and college" },
      {
        name: "description",
        content:
          "Inspire LMS brings science courses, live classes, assessments, certificates, and curated study books into one tutoring platform.",
      },
      { name: "author", content: "Inspire LMS" },
      { property: "og:title", content: "Inspire LMS" },
      { property: "og:description", content: "Science tutoring for high school and college." },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary" },
    ],
    links: [
      { rel: "stylesheet", href: appCss },
      { rel: "preconnect", href: "https://fonts.googleapis.com" },
      {
        rel: "preconnect",
        href: "https://fonts.gstatic.com",
        crossOrigin: "anonymous",
      },
      {
        rel: "stylesheet",
        href: "https://fonts.googleapis.com/css2?family=Playfair+Display:ital,wght@0,400..900;1,400..900&family=Inter:wght@300;400;500;600;700&display=swap",
      },
    ],
  }),
  shellComponent: RootShell,
  component: RootComponent,
  notFoundComponent: NotFoundComponent,
  errorComponent: ErrorComponent,
});

function RootShell({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <head>
        <HeadContent />
      </head>
      <body>
        {children}
        <Scripts />
      </body>
    </html>
  );
}

function RootComponent() {
  const { queryClient } = Route.useRouteContext();

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <Outlet />
        <Toaster
          position="top-center"
          richColors
          toastOptions={{
            classNames: {
              toast: "font-sans",
            },
          }}
        />
      </AuthProvider>
    </QueryClientProvider>
  );
}

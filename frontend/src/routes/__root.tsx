import { SiteHead } from "@/components/layout/site-head";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  Outlet,
  Link,
  createRootRouteWithContext,
  useRouter,
  HeadContent,
  Scripts,
} from "@tanstack/react-router";
import { type ReactNode } from "react";
import { Toaster } from "sonner";

import appCss from "../styles.css?url";
import { AuthProvider } from "../context/auth-context";

function NotFoundComponent() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-canvas px-4 py-12 font-sans text-navy">
      <div className="max-w-md w-full rounded-2xl border border-borderui bg-white p-8 sm:p-10 text-center shadow-sm">
        <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze">404</p>
        <h1 className="mt-3 font-serif text-4xl sm:text-5xl font-bold text-navy">Page not found</h1>
        <p className="mt-4 text-sm sm:text-base text-slatebody leading-relaxed">
          The page you're looking for has been moved, or it never existed in our archive.
        </p>
        <div className="mt-8">
          <Link
            to="/"
            className="inline-flex items-center justify-center min-h-[46px] rounded-lg bg-navy px-6 py-3 text-sm font-semibold text-white hover:bg-navy-light shadow-sm transition-colors"
          >
            Return to home
          </Link>
        </div>
      </div>
    </div>
  );
}

function ErrorComponent({ error, reset }: { error: unknown; reset: () => void }) {
  console.error(error);
  const router = useRouter();

  return (
    <div className="flex min-h-screen items-center justify-center bg-canvas px-4 py-12 font-sans text-navy">
      <div className="max-w-md w-full rounded-2xl border border-borderui bg-white p-8 sm:p-10 text-center shadow-sm">
        <p className="text-xs font-bold tracking-[0.16em] uppercase text-red-700">Error</p>
        <h1 className="mt-3 font-serif text-3xl sm:text-4xl font-bold text-navy">
          Something went wrong
        </h1>
        <p className="mt-4 text-sm text-slatebody leading-relaxed">
          We hit an unexpected problem loading this page. You can try again or head home.
        </p>
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          <button
            type="button"
            onClick={() => {
              router.invalidate();
              reset();
            }}
            className="inline-flex items-center justify-center min-h-[44px] rounded-lg bg-navy px-5 py-2.5 text-sm font-semibold text-white hover:bg-navy-light shadow-sm transition-colors"
          >
            Try again
          </button>
          <a
            href="/"
            className="inline-flex items-center justify-center min-h-[44px] rounded-lg border border-borderui bg-white px-5 py-2.5 text-sm font-semibold text-navy hover:bg-stoneband transition-colors"
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
      { title: "Inspire — Science tutoring for high school and college" },
      {
        name: "description",
        content:
          "Inspire brings science courses, live classes, assessments, certificates, and curated study books into one tutoring platform.",
      },
      { name: "author", content: "Inspire" },
      { property: "og:title", content: "Inspire" },
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
        <SiteHead />
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

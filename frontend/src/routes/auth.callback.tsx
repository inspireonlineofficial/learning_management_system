import { useEffect } from "react";
import { createFileRoute, useNavigate, useSearch } from "@tanstack/react-router";
import { z } from "zod";
import { toast } from "sonner";

import { useAuth } from "@/context/auth-context";
import { getProfile } from "@/lib/api/auth";
import { setStoredSession } from "@/lib/session";

const searchSchema = z.object({
  access_token: z.string().optional(),
  refresh_token: z.string().optional(),
  return: z.string().optional(),
  return_to: z.string().optional(),
  error: z.string().optional(),
});

function defaultDestination(user: { role: "student" | "teacher" | "admin"; onboarded?: boolean }) {
  if (user.role === "admin") return "/admin";
  if (user.role === "teacher") return "/teacher";
  if (user.onboarded === false) return "/onboarding/student-profile";
  return "/student";
}

export const Route = createFileRoute("/auth/callback")({
  head: () => ({ meta: [{ title: "Signing you in — Inspire LMS" }] }),
  validateSearch: searchSchema,
  component: CallbackPage,
});

function CallbackPage() {
  const navigate = useNavigate();
  const { setSession } = useAuth();
  const search = useSearch({ from: "/auth/callback" });

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (search.error) {
        toast.error("Sign-in was cancelled.");
        navigate({ to: "/login" });
        return;
      }

      let access = search.access_token;
      let refresh = search.refresh_token;
      let returnTo = search.return || search.return_to;

      // Fallback: parse tokens from the URL hash fragment if not present in query parameters
      if ((!access || !refresh || !returnTo) && typeof window !== "undefined") {
        const hash = window.location.hash.substring(1);
        if (hash) {
          const params = new URLSearchParams(hash);
          access = params.get("access_token") || undefined;
          refresh = params.get("refresh_token") || undefined;
          returnTo = params.get("return_to") || params.get("return") || returnTo;
        }
      }

      if (!access || !refresh) {
        toast.error("Missing tokens from sign-in provider.");
        navigate({ to: "/login" });
        return;
      }

      // Persist tokens so apiRequest can attach the Authorization header,
      // then fetch the profile.
      setStoredSession({
        accessToken: access,
        refreshToken: refresh,
        user: { id: "", email: "", full_name: "Scholar", role: "student" },
      });

      try {
        const user = await getProfile();
        if (cancelled) return;
        const session = { accessToken: access, refreshToken: refresh, user };
        setSession(session);
        toast.success(`Welcome, ${user.full_name.split(" ")[0]}.`);
        const dest = returnTo || defaultDestination(user);
        navigate({ to: dest as string });
      } catch (err) {
        if (cancelled) return;
        toast.error("Could not complete sign-in.");
        navigate({ to: "/login" });
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="min-h-screen grid place-items-center bg-surface text-brand font-sans">
      <div className="text-center">
        <p className="eyebrow text-accent mb-4">Signing you in</p>
        <p className="font-serif text-2xl">Opening your learning dashboard...</p>
      </div>
    </div>
  );
}

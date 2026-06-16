import { useState } from "react";
import { createFileRoute, Link, useNavigate, useSearch } from "@tanstack/react-router";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Eye, EyeOff } from "lucide-react";

import { AuthShell, Field, inputClass, primaryButtonClass } from "@/components/auth/auth-shell";
import { GoogleButton } from "@/components/auth/google-button";
import { loginUser } from "@/lib/api/auth";
import { useAuth } from "@/context/auth-context";
import { ApiError } from "@/lib/api/client";

const schema = z.object({
  email: z.string().email("Enter a valid email"),
  password: z.string().min(1, "Password is required"),
});

type FormValues = z.infer<typeof schema>;

const searchSchema = z.object({
  return: z.string().optional(),
});

function defaultDestination(user: { role: "student" | "teacher" | "admin"; onboarded?: boolean }) {
  if (user.role === "admin") return "/admin";
  if (user.role === "teacher") return "/teacher";
  if (user.onboarded === false) return "/onboarding/student-profile";
  return "/student";
}

export const Route = createFileRoute("/login")({
  head: () => ({
    meta: [
      { title: "Sign in — Inspire LMS" },
      { name: "description", content: "Sign in to your Inspire LMS account." },
    ],
  }),
  validateSearch: searchSchema,
  component: LoginPage,
});

function LoginPage() {
  const navigate = useNavigate();
  const { setSession } = useAuth();
  const { return: returnTo } = useSearch({ from: "/login" });
  const [showPassword, setShowPassword] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "" },
  });

  const mutation = useMutation({
    mutationFn: loginUser,
    onSuccess: (session) => {
      setSession(session);
      toast.success(`Welcome back, ${session.user.full_name.split(" ")[0]}.`);
      const dest = returnTo || defaultDestination(session.user);
      navigate({ to: dest as string });
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 401) {
        toast.error("Incorrect email or password.");
        return;
      }
      toast.error(error instanceof Error ? error.message : "Sign in failed.");
    },
  });

  return (
    <AuthShell
      eyebrow="Sign in"
      heading="Welcome back"
      subheading="Enter your credentials to access your learning dashboard."
      footer={
        <p className="text-sm text-brand/50">
          New to Inspire?{" "}
          <Link
            to="/register"
            className="text-accent font-semibold underline underline-offset-4 hover:text-brand"
          >
            Enroll now
          </Link>
        </p>
      }
    >
      <form
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        className="space-y-7"
        noValidate
      >
        <Field label="Email" htmlFor="email" error={form.formState.errors.email?.message}>
          <input
            id="email"
            type="email"
            autoComplete="email"
            placeholder="student@example.com"
            disabled={mutation.isPending}
            className={inputClass}
            {...form.register("email")}
          />
        </Field>

        <Field
          label="Password"
          htmlFor="password"
          error={form.formState.errors.password?.message}
          trailing={
            <Link
              to="/forgot-password"
              className="text-[10px] uppercase tracking-wider text-accent font-bold hover:opacity-70"
            >
              Forgot?
            </Link>
          }
        >
          <div className="relative">
            <input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              placeholder="••••••••"
              disabled={mutation.isPending}
              className={inputClass + " pr-10"}
              {...form.register("password")}
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              className="absolute right-0 top-1/2 -translate-y-1/2 text-brand/40 hover:text-brand p-2"
              aria-label={showPassword ? "Hide password" : "Show password"}
            >
              {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
            </button>
          </div>
        </Field>

        <button type="submit" disabled={mutation.isPending} className={primaryButtonClass}>
          {mutation.isPending ? "Signing in…" : "Sign in"}
          {!mutation.isPending && (
            <span className="group-hover:translate-x-1 transition-transform">→</span>
          )}
        </button>
      </form>

      <div className="relative my-8">
        <div className="absolute inset-0 flex items-center">
          <div className="w-full border-t border-brand/10" />
        </div>
        <div className="relative flex justify-center">
          <span className="bg-surface px-4 eyebrow text-brand/40">Or</span>
        </div>
      </div>

      <GoogleButton returnTo={returnTo} />
    </AuthShell>
  );
}

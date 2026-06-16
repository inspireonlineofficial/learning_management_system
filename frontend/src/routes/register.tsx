import { useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Eye, EyeOff } from "lucide-react";

import { AuthShell, Field, inputClass, primaryButtonClass } from "@/components/auth/auth-shell";
import { GoogleButton } from "@/components/auth/google-button";
import { PasswordStrength } from "@/components/auth/password-strength";
import { registerUser } from "@/lib/api/auth";
import { ApiError } from "@/lib/api/client";

const schema = z
  .object({
    full_name: z.string().min(2, "Please share your full name"),
    email: z.string().email("Enter a valid email"),
    password: z
      .string()
      .min(8, "At least 8 characters")
      .regex(/\d/, "Must include at least one number"),
    confirm: z.string(),
  })
  .refine((d) => d.password === d.confirm, {
    message: "Passwords don't match",
    path: ["confirm"],
  });

type FormValues = z.infer<typeof schema>;

export const Route = createFileRoute("/register")({
  head: () => ({
    meta: [
      { title: "Enroll — Inspire LMS" },
      { name: "description", content: "Create an Inspire LMS account and begin your study." },
    ],
  }),
  component: RegisterPage,
});

function RegisterPage() {
  const navigate = useNavigate();
  const [showPassword, setShowPassword] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { full_name: "", email: "", password: "", confirm: "" },
    mode: "onBlur",
  });

  const password = form.watch("password");

  const mutation = useMutation({
    mutationFn: (v: FormValues) =>
      registerUser({ full_name: v.full_name, email: v.email, password: v.password }),
    onSuccess: (_data, vars) => {
      toast.success("We've sent a 6-digit code to your email.");
      navigate({ to: "/verify-otp", search: { email: vars.email } });
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 409) {
        form.setError("email", { message: "An account with this email already exists" });
        return;
      }
      toast.error(error instanceof Error ? error.message : "Could not create account.");
    },
  });

  return (
    <AuthShell
      eyebrow="Enroll"
      heading="Begin your study"
      subheading="Create an account to browse courses, join live classes, and track progress."
      footer={
        <p className="text-sm text-brand/50">
          Already enrolled?{" "}
          <Link
            to="/login"
            className="text-accent font-semibold underline underline-offset-4 hover:text-brand"
          >
            Sign in
          </Link>
        </p>
      }
    >
      <form
        onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
        className="space-y-7"
        noValidate
      >
        <Field
          label="Full name"
          htmlFor="full_name"
          error={form.formState.errors.full_name?.message}
        >
          <input
            id="full_name"
            type="text"
            autoComplete="name"
            placeholder="Elena Marchetti"
            disabled={mutation.isPending}
            className={inputClass}
            {...form.register("full_name")}
          />
        </Field>

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
          hint="At least 8 characters, including a number"
        >
          <div className="relative">
            <input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
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
          <PasswordStrength value={password} />
        </Field>

        <Field
          label="Confirm password"
          htmlFor="confirm"
          error={form.formState.errors.confirm?.message}
        >
          <input
            id="confirm"
            type={showPassword ? "text" : "password"}
            autoComplete="new-password"
            placeholder="••••••••"
            disabled={mutation.isPending}
            className={inputClass}
            {...form.register("confirm")}
          />
        </Field>

        <button type="submit" disabled={mutation.isPending} className={primaryButtonClass}>
          {mutation.isPending ? "Creating account…" : "Create account"}
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

      <GoogleButton label="Enroll with Google" />
    </AuthShell>
  );
}

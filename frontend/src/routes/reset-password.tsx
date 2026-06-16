import { useState } from "react";
import { createFileRoute, Link, useNavigate, useSearch } from "@tanstack/react-router";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Eye, EyeOff } from "lucide-react";

import { AuthShell, Field, inputClass, primaryButtonClass } from "@/components/auth/auth-shell";
import { PasswordStrength } from "@/components/auth/password-strength";
import { resetPassword } from "@/lib/api/auth";
import { ApiError } from "@/lib/api/client";

const schema = z
  .object({
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

const searchSchema = z.object({ token: z.string().optional() });

export const Route = createFileRoute("/reset-password")({
  head: () => ({
    meta: [
      { title: "Reset password — Inspire LMS" },
      { name: "description", content: "Set a new password for your Inspire LMS account." },
    ],
  }),
  validateSearch: searchSchema,
  component: ResetPage,
});

function ResetPage() {
  const navigate = useNavigate();
  const { token } = useSearch({ from: "/reset-password" });
  const [show, setShow] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { password: "", confirm: "" },
  });

  const password = form.watch("password");

  const mutation = useMutation({
    mutationFn: (v: FormValues) => resetPassword({ token: token ?? "", password: v.password }),
    onSuccess: () => {
      toast.success("Password updated. Please sign in.");
      navigate({ to: "/login" });
    },
    onError: (error) => {
      if (error instanceof ApiError && (error.status === 400 || error.status === 410)) {
        toast.error("This reset link is invalid or has expired.");
        return;
      }
      toast.error(error instanceof Error ? error.message : "Could not reset password.");
    },
  });

  if (!token) {
    return (
      <AuthShell
        eyebrow="Reset password"
        heading="Invalid reset link"
        subheading="The link is missing a token. Please request a new one."
      >
        <Link to="/forgot-password" className={primaryButtonClass}>
          Request a new link
        </Link>
      </AuthShell>
    );
  }

  return (
    <AuthShell
      eyebrow="Reset password"
      heading="Set a new password"
      subheading="Choose something strong. You'll use it the next time you sign in."
      footer={
        <p className="text-sm text-brand/50">
          Changed your mind?{" "}
          <Link
            to="/login"
            className="text-accent font-semibold underline underline-offset-4 hover:text-brand"
          >
            Back to sign in
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
          label="New password"
          htmlFor="password"
          error={form.formState.errors.password?.message}
          hint="At least 8 characters, including a number"
        >
          <div className="relative">
            <input
              id="password"
              type={show ? "text" : "password"}
              autoComplete="new-password"
              placeholder="••••••••"
              disabled={mutation.isPending}
              className={inputClass + " pr-10"}
              {...form.register("password")}
            />
            <button
              type="button"
              onClick={() => setShow((v) => !v)}
              className="absolute right-0 top-1/2 -translate-y-1/2 text-brand/40 hover:text-brand p-2"
              aria-label={show ? "Hide password" : "Show password"}
            >
              {show ? <EyeOff size={18} /> : <Eye size={18} />}
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
            type={show ? "text" : "password"}
            autoComplete="new-password"
            placeholder="••••••••"
            disabled={mutation.isPending}
            className={inputClass}
            {...form.register("confirm")}
          />
        </Field>

        <button type="submit" disabled={mutation.isPending} className={primaryButtonClass}>
          {mutation.isPending ? "Updating…" : "Update password"}
        </button>
      </form>
    </AuthShell>
  );
}

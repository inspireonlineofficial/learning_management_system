import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";

import { AuthShell, Field, inputClass, primaryButtonClass } from "@/components/auth/auth-shell";
import { forgotPassword } from "@/lib/api/auth";

const schema = z.object({ email: z.string().email("Enter a valid email") });
type FormValues = z.infer<typeof schema>;

export const Route = createFileRoute("/forgot-password")({
  head: () => ({
    meta: [
      { title: "Forgot password — Inspire LMS" },
      {
        name: "description",
        content: "Request a password reset link for your Inspire LMS account.",
      },
    ],
  }),
  component: ForgotPage,
});

function ForgotPage() {
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "" },
  });

  const mutation = useMutation({
    mutationFn: forgotPassword,
    onSuccess: () => {
      toast.success("If an account exists, a reset link is on its way.");
    },
    onError: () => {
      toast.success("If an account exists, a reset link is on its way.");
    },
  });

  return (
    <AuthShell
      eyebrow="Recover access"
      heading="Forgot your password?"
      subheading="Enter your email and we'll send you a single-use link to reset it. The link expires in 30 minutes."
      footer={
        <p className="text-sm text-brand/50">
          Remembered it?{" "}
          <Link
            to="/login"
            className="text-accent font-semibold underline underline-offset-4 hover:text-brand"
          >
            Back to sign in
          </Link>
        </p>
      }
    >
      {mutation.isSuccess ? (
        <div className="border border-brand/10 p-6 text-center">
          <p className="font-serif text-xl mb-2">Check your inbox</p>
          <p className="text-sm text-brand/55 leading-relaxed">
            If we found an account, you'll receive a reset link shortly. Don't forget to check your
            spam folder.
          </p>
        </div>
      ) : (
        <form
          onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
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

          <button type="submit" disabled={mutation.isPending} className={primaryButtonClass}>
            {mutation.isPending ? "Sending…" : "Send reset link"}
          </button>
        </form>
      )}
    </AuthShell>
  );
}

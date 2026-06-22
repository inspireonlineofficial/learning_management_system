import { useEffect, useState } from "react";
import { createFileRoute, useNavigate, useSearch } from "@tanstack/react-router";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { toast } from "sonner";

import { AuthShell, primaryButtonClass } from "@/components/auth/auth-shell";
import { resendOtp, verifyOtp } from "@/lib/api/auth";
import { useAuth } from "@/context/auth-context";
import { ApiError } from "@/lib/api/client";
import { OTPInput } from "input-otp";
import { cn } from "@/lib/utils";

const searchSchema = z.object({
  email: z.string().email().optional(),
});

export const Route = createFileRoute("/verify-otp")({
  head: () => ({
    meta: [
      { title: "Verify your email — Inspire" },
      {
        name: "description",
        content: "Enter the 6-digit verification code we sent to your email.",
      },
    ],
  }),
  validateSearch: searchSchema,
  component: VerifyOtpPage,
});

function VerifyOtpPage() {
  const navigate = useNavigate();
  const { setSession } = useAuth();
  const { email } = useSearch({ from: "/verify-otp" });
  const [otp, setOtp] = useState("");
  const [cooldown, setCooldown] = useState(0);

  useEffect(() => {
    if (cooldown <= 0) return;
    const t = setTimeout(() => setCooldown((c) => c - 1), 1000);
    return () => clearTimeout(t);
  }, [cooldown]);

  const verify = useMutation({
    mutationFn: verifyOtp,
    onSuccess: (session) => {
      setSession(session);
      toast.success("Email verified. Welcome to Inspire.");
      navigate({ to: session.user.onboarded === false ? "/onboarding/student-profile" : "/" });
    },
    onError: (error) => {
      setOtp("");
      if (error instanceof ApiError && error.status === 400) {
        toast.error("That code is invalid or expired.");
        return;
      }
      toast.error(error instanceof Error ? error.message : "Verification failed.");
    },
  });

  const resend = useMutation({
    mutationFn: resendOtp,
    onSuccess: () => {
      toast.success("A new code is on its way.");
      setCooldown(60);
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Could not resend code.");
    },
  });

  const onSubmit = (value: string) => {
    if (!email) {
      toast.error("Missing email. Please register again.");
      return;
    }
    verify.mutate({ email, otp: value });
  };

  if (!email) {
    return (
      <AuthShell
        eyebrow="Verify"
        heading="No email provided"
        subheading="Please return to registration and try again."
      >
        <button onClick={() => navigate({ to: "/register" })} className={primaryButtonClass}>
          Back to enroll
        </button>
      </AuthShell>
    );
  }

  return (
    <AuthShell
      eyebrow="Verify your email"
      heading="Enter the 6-digit code"
      subheading={
        <>
          We sent a code to <strong className="text-brand">{email}</strong>. It expires in 24 hours.
        </>
      }
    >
      <div className="space-y-8">
        <OTPInput
          maxLength={6}
          value={otp}
          onChange={(v) => {
            setOtp(v);
            if (v.length === 6) onSubmit(v);
          }}
          containerClassName="flex items-center justify-between gap-2"
          render={({ slots }) => (
            <>
              {slots.map((slot, idx) => (
                <div
                  key={idx}
                  className={cn(
                    "h-16 w-12 sm:w-14 flex items-center justify-center font-serif text-3xl",
                    "border-b-2 transition-colors",
                    slot.isActive ? "border-accent" : "border-brand/20",
                  )}
                >
                  {slot.char ??
                    (slot.isActive ? <span className="text-accent animate-pulse">|</span> : "")}
                </div>
              ))}
            </>
          )}
        />

        <button
          onClick={() => onSubmit(otp)}
          disabled={otp.length !== 6 || verify.isPending}
          className={primaryButtonClass}
        >
          {verify.isPending ? "Verifying…" : "Verify code"}
        </button>

        <div className="text-center">
          <button
            onClick={() => resend.mutate({ email })}
            disabled={cooldown > 0 || resend.isPending}
            className="text-sm text-brand/55 hover:text-brand disabled:opacity-50 underline underline-offset-4 decoration-accent/40"
          >
            {cooldown > 0
              ? `Resend code in ${cooldown}s`
              : resend.isPending
                ? "Sending…"
                : "Resend code"}
          </button>
        </div>
      </div>
    </AuthShell>
  );
}

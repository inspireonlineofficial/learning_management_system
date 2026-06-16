import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

import { useAuth } from "@/context/auth-context";
import { apiRequest } from "@/lib/api/client";

export const Route = createFileRoute("/onboarding/student-profile")({
  component: Page,
});

const draftKey = "inspire:onboarding-student-profile";

type StudentProfileForm = {
  school_name: string;
  class_grade: string;
  roll_number: string;
  date_of_birth: string;
  gender: string;
  guardian_name: string;
  guardian_contact: string;
};

const emptyForm: StudentProfileForm = {
  school_name: "",
  class_grade: "",
  roll_number: "",
  date_of_birth: "",
  gender: "",
  guardian_name: "",
  guardian_contact: "",
};

function Page() {
  const { user, isHydrated, refreshProfile } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState<StudentProfileForm>(emptyForm);

  useEffect(() => {
    if (isHydrated && !user) navigate({ to: "/login" });
  }, [isHydrated, user, navigate]);

  useEffect(() => {
    if (!isHydrated || !user) return;
    let cancelled = false;
    const savedDraft = readDraft();
    if (savedDraft) setForm((current) => ({ ...current, ...savedDraft }));

    apiRequest<Partial<StudentProfileForm>>("/v1/onboarding/student-profile", {
      auth: true,
    })
      .then((profile) => {
        if (!cancelled) setForm((current) => ({ ...current, ...profile, ...savedDraft }));
      })
      .catch(() => null);

    return () => {
      cancelled = true;
    };
  }, [isHydrated, user]);

  useEffect(() => {
    const timer = window.setInterval(() => writeDraft(form), 30000);
    return () => window.clearInterval(timer);
  }, [form]);

  const mut = useMutation({
    mutationFn: () =>
      apiRequest<{ ok: true }>("/v1/onboarding/student-profile", {
        method: "PUT",
        auth: true,
        body: form,
      }),
    onSuccess: async () => {
      clearDraft();
      toast.success("Welcome aboard");
      await refreshProfile();
      navigate({ to: "/student" });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <div className="min-h-screen grid place-items-center bg-surface text-brand font-sans px-4 py-12">
      <div className="max-w-md w-full">
        <p className="eyebrow text-accent">Onboarding</p>
        <h1 className="mt-3 font-serif text-4xl">Complete your profile</h1>
        <p className="mt-3 text-sm text-brand/65">
          Add your academic details to unlock enrollments and non-preview lessons.
        </p>
        <div className="mt-8 space-y-4">
          {(["school_name", "class_grade", "roll_number", "date_of_birth"] as const).map((k) => (
            <label key={k} className="block">
              <span className="text-xs eyebrow text-brand/45">{k.replace("_", " ")}</span>
              <input
                type={k === "date_of_birth" ? "date" : "text"}
                className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
                value={form[k]}
                onChange={(e) => setForm({ ...form, [k]: e.target.value })}
              />
            </label>
          ))}
          <label className="block">
            <span className="text-xs eyebrow text-brand/45">gender</span>
            <input
              className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              value={form.gender}
              onChange={(e) => setForm({ ...form, gender: e.target.value })}
            />
          </label>
          <label className="block">
            <span className="text-xs eyebrow text-brand/45">guardian name</span>
            <input
              className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              value={form.guardian_name}
              onChange={(e) => setForm({ ...form, guardian_name: e.target.value })}
            />
          </label>
          <label className="block">
            <span className="text-xs eyebrow text-brand/45">guardian contact</span>
            <input
              className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              value={form.guardian_contact}
              onChange={(e) => setForm({ ...form, guardian_contact: e.target.value })}
            />
          </label>
          <button
            onClick={() => mut.mutate()}
            disabled={
              mut.isPending ||
              !form.school_name ||
              !form.class_grade ||
              !form.roll_number ||
              !form.date_of_birth
            }
            className="w-full bg-brand text-white py-3 text-sm disabled:opacity-50"
          >
            {mut.isPending ? "Saving…" : "Continue"}
          </button>
        </div>
      </div>
    </div>
  );
}

function readDraft(): Partial<StudentProfileForm> | null {
  if (typeof window === "undefined") return null;
  try {
    return JSON.parse(
      window.localStorage.getItem(draftKey) || "null",
    ) as Partial<StudentProfileForm> | null;
  } catch {
    return null;
  }
}

function writeDraft(form: StudentProfileForm) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(draftKey, JSON.stringify(form));
}

function clearDraft() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(draftKey);
}

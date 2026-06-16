import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { AppShell, SectionHeading } from "@/components/layout/app-shell";
import {
  getMySettings,
  updateMySettings,
  changePassword,
  type UserSettings,
} from "@/lib/api/settings";

export const Route = createFileRoute("/_authenticated/student/settings")({
  component: Page,
});

function Page() {
  const qc = useQueryClient();
  const { data } = useQuery({ queryKey: ["me-settings"], queryFn: getMySettings });
  const [form, setForm] = useState<Partial<UserSettings>>({});

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  const save = useMutation({
    mutationFn: () => updateMySettings(form),
    onSuccess: () => {
      toast.success("Settings saved");
      qc.invalidateQueries({ queryKey: ["me-settings"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const [pw, setPw] = useState({ current_password: "", new_password: "" });
  const pwMut = useMutation({
    mutationFn: () => changePassword(pw),
    onSuccess: () => {
      toast.success("Password updated");
      setPw({ current_password: "", new_password: "" });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <AppShell eyebrow="Settings" title="Account settings">
      <SectionHeading title="Preferences" />
      <div className="border border-brand/10 bg-white/40 p-6 space-y-4 max-w-2xl">
        {(
          [
            ["email_notifications", "Email notifications"],
            ["push_notifications", "Push notifications"],
            ["newsletter_opt_in", "Newsletter"],
          ] as const
        ).map(([key, label]) => (
          <label key={key} className="flex items-center justify-between">
            <span className="text-sm">{label}</span>
            <input
              type="checkbox"
              checked={!!form[key]}
              onChange={(e) => setForm((f) => ({ ...f, [key]: e.target.checked }))}
            />
          </label>
        ))}
        <div className="grid grid-cols-2 gap-4 pt-2">
          <label className="block">
            <span className="text-xs eyebrow text-brand/45">Language</span>
            <input
              className="mt-1 w-full border border-brand/15 px-3 py-2 text-sm bg-white"
              value={form.language ?? ""}
              onChange={(e) => setForm((f) => ({ ...f, language: e.target.value }))}
            />
          </label>
          <label className="block">
            <span className="text-xs eyebrow text-brand/45">Timezone</span>
            <input
              className="mt-1 w-full border border-brand/15 px-3 py-2 text-sm bg-white"
              value={form.timezone ?? ""}
              onChange={(e) => setForm((f) => ({ ...f, timezone: e.target.value }))}
            />
          </label>
        </div>
        <button
          onClick={() => save.mutate()}
          disabled={save.isPending}
          className="mt-2 bg-brand text-white px-6 py-2 text-sm disabled:opacity-50"
        >
          {save.isPending ? "Saving…" : "Save"}
        </button>
      </div>

      <SectionHeading title="Change password" />
      <div className="border border-brand/10 bg-white/40 p-6 space-y-3 max-w-2xl">
        <label className="block">
          <span className="text-xs eyebrow text-brand/45">Current password</span>
          <input
            type="password"
            className="mt-1 w-full border border-brand/15 px-3 py-2 text-sm bg-white"
            value={pw.current_password}
            onChange={(e) => setPw((p) => ({ ...p, current_password: e.target.value }))}
          />
        </label>
        <label className="block">
          <span className="text-xs eyebrow text-brand/45">New password</span>
          <input
            type="password"
            className="mt-1 w-full border border-brand/15 px-3 py-2 text-sm bg-white"
            value={pw.new_password}
            onChange={(e) => setPw((p) => ({ ...p, new_password: e.target.value }))}
          />
        </label>
        <button
          onClick={() => pwMut.mutate()}
          disabled={pwMut.isPending || !pw.current_password || pw.new_password.length < 8}
          className="mt-2 bg-brand text-white px-6 py-2 text-sm disabled:opacity-50"
        >
          {pwMut.isPending ? "Updating…" : "Update password"}
        </button>
      </div>
    </AppShell>
  );
}

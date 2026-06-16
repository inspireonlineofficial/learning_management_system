import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { useAuth } from "@/context/auth-context";
import { updateProfile } from "@/lib/api/auth";
import { changePassword } from "@/lib/api/settings";

export const Route = createFileRoute("/_authenticated/admin/settings")({
  component: AdminSettingsPage,
});

function AdminSettingsPage() {
  const { user, refreshProfile } = useAuth();
  const [fullName, setFullName] = useState(user?.full_name ?? "");
  const [passwords, setPasswords] = useState({ current_password: "", new_password: "" });

  useEffect(() => {
    setFullName(user?.full_name ?? "");
  }, [user?.full_name]);

  const profile = useMutation({
    mutationFn: () => updateProfile({ full_name: fullName }),
    onSuccess: async () => {
      await refreshProfile();
      toast.success("Profile updated");
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const password = useMutation({
    mutationFn: () => changePassword(passwords),
    onSuccess: () => {
      setPasswords({ current_password: "", new_password: "" });
      toast.success("Password updated");
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <AppShell eyebrow="Account" title="Admin settings">
      <div className="mb-5">
        <Link
          to="/admin/system"
          className="inline-flex border border-brand/15 px-4 py-2 text-xs text-brand/65 hover:bg-brand/[0.04]"
        >
          Platform settings
        </Link>
      </div>

      <div className="grid gap-6 lg:grid-cols-2 max-w-5xl">
        <section className="border border-brand/10 bg-white/50 p-6 space-y-5">
          <h2 className="font-serif text-lg">Profile</h2>
          <label className="block">
            <span className="eyebrow text-brand/45">Name</span>
            <input
              value={fullName}
              onChange={(event) => setFullName(event.target.value)}
              className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
          <label className="block">
            <span className="eyebrow text-brand/45">Email</span>
            <input
              value={user?.email ?? ""}
              readOnly
              className="mt-2 w-full border border-brand/15 bg-white/70 px-3 py-2 text-sm text-brand/55"
            />
          </label>
          <button
            onClick={() => profile.mutate()}
            disabled={!fullName.trim() || profile.isPending}
            className="bg-brand text-white px-5 py-2.5 text-sm disabled:opacity-50"
          >
            {profile.isPending ? "Saving..." : "Save profile"}
          </button>
        </section>

        <section className="border border-brand/10 bg-white/50 p-6 space-y-5">
          <h2 className="font-serif text-lg">Password</h2>
          <label className="block">
            <span className="eyebrow text-brand/45">Current password</span>
            <input
              type="password"
              value={passwords.current_password}
              onChange={(event) =>
                setPasswords({ ...passwords, current_password: event.target.value })
              }
              className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
          <label className="block">
            <span className="eyebrow text-brand/45">New password</span>
            <input
              type="password"
              value={passwords.new_password}
              onChange={(event) => setPasswords({ ...passwords, new_password: event.target.value })}
              className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
          <button
            onClick={() => password.mutate()}
            disabled={
              passwords.current_password.length === 0 ||
              passwords.new_password.length < 8 ||
              password.isPending
            }
            className="bg-brand text-white px-5 py-2.5 text-sm disabled:opacity-50"
          >
            {password.isPending ? "Updating..." : "Update password"}
          </button>
        </section>
      </div>
    </AppShell>
  );
}

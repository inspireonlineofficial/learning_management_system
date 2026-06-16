import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { apiRequest } from "@/lib/api/client";

export const Route = createFileRoute("/_authenticated/admin/users/new")({
  component: Page,
});

function Page() {
  const navigate = useNavigate();
  const [form, setForm] = useState({ email: "", full_name: "", role: "student", password: "" });
  const mut = useMutation({
    mutationFn: () =>
      apiRequest<{ id: string }>("/v1/admin/users", { method: "POST", auth: true, body: form }),
    onSuccess: (u) => {
      toast.success("User created");
      navigate({ to: "/admin/users/$userId", params: { userId: u.id } });
    },
    onError: (e: Error) => toast.error(e.message),
  });
  return (
    <AppShell eyebrow="User" title="Create user">
      <div className="max-w-xl space-y-4">
        {(["email", "full_name", "password"] as const).map((k) => (
          <label key={k} className="block">
            <span className="text-xs eyebrow text-brand/45">{k.replace("_", " ")}</span>
            <input
              type={k === "password" ? "password" : "text"}
              className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              value={form[k]}
              onChange={(e) => setForm({ ...form, [k]: e.target.value })}
            />
          </label>
        ))}
        <label className="block">
          <span className="text-xs eyebrow text-brand/45">Role</span>
          <select
            className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            value={form.role}
            onChange={(e) => setForm({ ...form, role: e.target.value })}
          >
            <option>student</option>
            <option>teacher</option>
            <option>admin</option>
          </select>
        </label>
        <button
          onClick={() => mut.mutate()}
          disabled={mut.isPending || !form.email}
          className="bg-brand text-white px-6 py-2 text-sm disabled:opacity-50"
        >
          {mut.isPending ? "Creating…" : "Create user"}
        </button>
      </div>
    </AppShell>
  );
}

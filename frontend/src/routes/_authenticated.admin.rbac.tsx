import { createFileRoute } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { listRoles, updateRole } from "@/lib/api/rbac";

export const Route = createFileRoute("/_authenticated/admin/rbac")({
  component: Page,
});

function Page() {
  const qc = useQueryClient();
  const { data } = useQuery({ queryKey: ["rbac"], queryFn: listRoles });
  const mut = useMutation({
    mutationFn: (args: { name: string; permissions: string[] }) =>
      updateRole(args.name, args.permissions),
    onSuccess: () => {
      toast.success("Role updated");
      qc.invalidateQueries({ queryKey: ["rbac"] });
    },
  });

  const toggle = (roleName: string, perms: string[], p: string) => {
    const next = perms.includes(p) ? perms.filter((x) => x !== p) : [...perms, p];
    mut.mutate({ name: roleName, permissions: next });
  };

  return (
    <AppShell eyebrow="RBAC" title="Roles & permissions">
      {data && (
        <div className="space-y-6">
          {data.items.map((r) => (
            <div key={r.name} className="border border-brand/10 bg-white/40 p-6">
              <h3 className="font-serif text-xl">{r.name}</h3>
              {r.description && <p className="text-xs text-brand/55 mt-1">{r.description}</p>}
              <div className="mt-4 grid sm:grid-cols-2 lg:grid-cols-3 gap-2">
                {data.all_permissions.map((p) => (
                  <label key={p} className="flex items-center gap-2 text-xs">
                    <input
                      type="checkbox"
                      checked={r.permissions.includes(p)}
                      onChange={() => toggle(r.name, r.permissions, p)}
                    />
                    {p}
                  </label>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </AppShell>
  );
}

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, Outlet, useNavigate, useRouterState } from "@tanstack/react-router";
import { Plus, Search, Shield, ShieldOff, UserCog } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { useAuth } from "@/context/auth-context";
import {
  impersonateUser,
  listUsers,
  setUserActive,
  updateUserRole,
  type AdminUser,
} from "@/lib/api/admin";
import { startImpersonation } from "@/lib/session";

export const Route = createFileRoute("/_authenticated/admin/users")({
  component: AdminUsersPage,
});

function AdminUsersPage() {
  const qc = useQueryClient();
  const { user: currentUser, setSession } = useAuth();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const [search, setSearch] = useState("");
  const [role, setRole] = useState<"" | "student" | "teacher" | "admin">("");

  const users = useQuery({
    queryKey: ["admin-users", { search, role }],
    queryFn: () => listUsers({ search, role: role || undefined, limit: 100 }),
  });

  const changeRole = useMutation({
    mutationFn: ({ id, role }: { id: string; role: AdminUser["role"] }) => updateUserRole(id, role),
    onSuccess: () => {
      toast.success("Role updated");
      qc.invalidateQueries({ queryKey: ["admin-users"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const toggleActive = useMutation({
    mutationFn: ({ id, active }: { id: string; active: boolean }) => setUserActive(id, active),
    onSuccess: () => {
      toast.success("Updated");
      qc.invalidateQueries({ queryKey: ["admin-users"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const impersonate = useMutation({
    mutationFn: (userId: string) => impersonateUser(userId),
    onSuccess: (tok) => {
      startImpersonation({
        accessToken: tok.access_token,
        refreshToken: tok.refresh_token,
        user: tok.user,
      });
      setSession({
        accessToken: tok.access_token,
        refreshToken: tok.refresh_token,
        user: tok.user,
      });
      toast.success(`Viewing as ${tok.user.full_name}`);
      const dest =
        tok.user.role === "admin"
          ? "/admin"
          : tok.user.role === "teacher"
            ? "/teacher"
            : "/student";
      navigate({ to: dest });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (pathname !== "/admin/users") {
    return <Outlet />;
  }

  return (
    <AppShell eyebrow="Administration" title="Users.">
      <div className="flex flex-wrap gap-3 mb-6">
        <div className="flex-1 min-w-[240px] relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-brand/40" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by name or email…"
            className="w-full pl-10 pr-4 py-2.5 text-sm border border-brand/15 bg-white/50 focus:outline-none focus:border-brand/40"
          />
        </div>
        <select
          value={role}
          onChange={(e) => setRole(e.target.value as typeof role)}
          className="px-4 py-2.5 text-sm border border-brand/15 bg-white/50"
        >
          <option value="">All roles</option>
          <option value="student">Students</option>
          <option value="teacher">Teachers</option>
          <option value="admin">Admins</option>
        </select>
        <Link
          to="/admin/users/new"
          className="inline-flex items-center gap-1.5 bg-brand text-white px-4 py-2.5 text-sm"
        >
          <Plus className="h-4 w-4" /> New user
        </Link>
      </div>

      {users.isError ? (
        <div className="border border-destructive/20 bg-destructive/5 p-6 text-sm">
          <p className="font-medium text-destructive">Couldn't load users</p>
          <p className="mt-1 text-brand/60">{(users.error as Error)?.message}</p>
        </div>
      ) : users.isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-16 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : !users.data || users.data.data.length === 0 ? (
        <EmptyState title="No users found" />
      ) : (
        <div className="border border-brand/10 overflow-x-auto bg-white/40">
          <table className="min-w-full text-sm">
            <thead className="bg-brand/[0.03] text-left">
              <tr>
                <th className="px-4 py-3 font-medium text-brand/60">User</th>
                <th className="px-4 py-3 font-medium text-brand/60">Role</th>
                <th className="px-4 py-3 font-medium text-brand/60">Joined</th>
                <th className="px-4 py-3 font-medium text-brand/60">Last sign in</th>
                <th className="px-4 py-3 font-medium text-brand/60 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-brand/10">
              {users.data.data.map((u) => (
                <tr key={u.id} className={u.is_active ? "" : "opacity-60"}>
                  <td className="px-4 py-3">
                    <Link
                      to="/admin/users/$userId"
                      params={{ userId: u.id }}
                      className="font-medium hover:text-accent"
                    >
                      {u.full_name}
                    </Link>
                    <p className="text-xs text-brand/55">{u.email}</p>
                  </td>
                  <td className="px-4 py-3">
                    <select
                      value={u.role}
                      onChange={(e) =>
                        changeRole.mutate({ id: u.id, role: e.target.value as AdminUser["role"] })
                      }
                      className="px-2 py-1 text-xs border border-brand/15 bg-white"
                    >
                      <option value="student">Student</option>
                      <option value="teacher">Teacher</option>
                      <option value="admin">Admin</option>
                    </select>
                  </td>
                  <td className="px-4 py-3 text-xs text-brand/55">
                    {new Date(u.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-xs text-brand/55">
                    {u.last_sign_in_at ? new Date(u.last_sign_in_at).toLocaleDateString() : "Never"}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="inline-flex items-center gap-2">
                      {currentUser?.id !== u.id && (
                        <button
                          onClick={() => impersonate.mutate(u.id)}
                          disabled={impersonate.isPending}
                          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs border border-brand/15 hover:bg-brand/[0.03]"
                          title="Impersonate user"
                        >
                          <UserCog className="h-3 w-3" />
                          Impersonate
                        </button>
                      )}
                      <button
                        onClick={() => toggleActive.mutate({ id: u.id, active: !u.is_active })}
                        disabled={toggleActive.isPending}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs border border-brand/15 hover:bg-brand/[0.03]"
                      >
                        {u.is_active ? (
                          <>
                            <ShieldOff className="h-3 w-3" />
                            Deactivate
                          </>
                        ) : (
                          <>
                            <Shield className="h-3 w-3" />
                            Activate
                          </>
                        )}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </AppShell>
  );
}

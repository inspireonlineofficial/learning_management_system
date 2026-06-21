import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import {
  ArrowLeft,
  KeyRound,
  Mail,
  ShieldAlert,
  ShieldCheck,
  Trash2,
  User as UserIcon,
  UserCog,
} from "lucide-react";

import { AppShell } from "@/components/layout/app-shell";
import { DetailGrid } from "@/components/layout/data-page";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { useAuth } from "@/context/auth-context";
import { startImpersonation } from "@/lib/session";
import {
  deleteUser,
  getAdminUser,
  getUserActivity,
  getUserEnrollments,
  impersonateUser,
  sendPasswordReset,
  setUserActive,
  updateUserRole,
  type AdminUser,
} from "@/lib/api/admin";

export const Route = createFileRoute("/_authenticated/admin/users/$userId")({
  component: Page,
});

type Tab = "profile" | "enrollments" | "activity";

function Page() {
  const { userId } = Route.useParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { user: currentUser, setSession } = useAuth();
  const [tab, setTab] = useState<Tab>("profile");

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["admin-user", userId],
    queryFn: () => getAdminUser(userId),
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["admin-user", userId] });

  const roleMut = useMutation({
    mutationFn: (r: AdminUser["role"]) => updateUserRole(userId, r),
    onSuccess: () => {
      toast.success("Role updated");
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const activeMut = useMutation({
    mutationFn: (a: boolean) => setUserActive(userId, a),
    onSuccess: () => {
      toast.success("Updated");
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const resetMut = useMutation({
    mutationFn: () => sendPasswordReset(userId),
    onSuccess: () => toast.success("Password reset email sent"),
    onError: (e: Error) => toast.error(e.message),
  });

  const deleteMut = useMutation({
    mutationFn: () => deleteUser(userId),
    onSuccess: () => {
      toast.success("User deleted");
      qc.invalidateQueries({ queryKey: ["admin-users"] });
      navigate({ to: "/admin/users" });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const impersonateMut = useMutation({
    mutationFn: () => impersonateUser(userId),
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
      toast.success(`Now viewing as ${tok.user.full_name}`);
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

  return (
    <AppShell eyebrow="User" title={data?.full_name ?? (isLoading ? "Loading…" : "User")}>
      <Link
        to="/admin/users"
        className="inline-flex items-center gap-1.5 text-xs text-brand/55 hover:text-brand mb-5"
      >
        <ArrowLeft className="h-3 w-3" /> All users
      </Link>

      {isError && (
        <QueryErrorPanel
          error={error}
          variant="compact"
          message={(error as Error)?.message ?? "Failed to load."}
        />
      )}
      {isLoading && <div className="h-40 border border-brand/10 bg-white/40 animate-pulse" />}

      {data && (
        <>
          <header className="flex items-center gap-4 mb-6">
            <div className="h-14 w-14 bg-brand/10 grid place-items-center text-brand/60 overflow-hidden flex-shrink-0">
              {data.avatar_url ? (
                <img src={data.avatar_url} alt="" className="h-full w-full object-cover" />
              ) : (
                <UserIcon className="h-6 w-6" />
              )}
            </div>
            <div className="min-w-0">
              <p className="text-sm text-brand/60 truncate">{data.email}</p>
              <div className="mt-1 flex items-center gap-2 text-xs">
                <span className="px-2 py-0.5 border border-brand/15 capitalize">{data.role}</span>
                <span
                  className={`inline-flex items-center gap-1 px-2 py-0.5 ${
                    data.is_active
                      ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                      : "bg-amber-50 text-amber-700 border border-amber-200"
                  }`}
                >
                  {data.is_active ? (
                    <ShieldCheck className="h-3 w-3" />
                  ) : (
                    <ShieldAlert className="h-3 w-3" />
                  )}
                  {data.is_active ? "Active" : "Disabled"}
                </span>
              </div>
            </div>
          </header>

          <nav className="flex flex-wrap gap-2 mb-6">
            {(
              [
                ["profile", "Profile"],
                ["enrollments", "Enrollments"],
                ["activity", "Activity"],
              ] as [Tab, string][]
            ).map(([t, label]) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={`px-4 py-2 text-xs font-medium ${
                  tab === t
                    ? "bg-brand text-white"
                    : "border border-brand/15 text-brand/70 hover:bg-brand/[0.03]"
                }`}
              >
                {label}
              </button>
            ))}
          </nav>

          {tab === "profile" && (
            <>
              <DetailGrid
                items={[
                  { label: "Email", value: data.email },
                  { label: "Role", value: data.role },
                  { label: "Status", value: data.is_active ? "Active" : "Disabled" },
                  {
                    label: "Joined",
                    value: data.created_at ? new Date(data.created_at).toLocaleDateString() : "—",
                  },
                  {
                    label: "Last sign in",
                    value: data.last_sign_in_at
                      ? new Date(data.last_sign_in_at).toLocaleString()
                      : "Never",
                  },
                ]}
              />

              <section className="mt-8">
                <h3 className="font-serif text-lg mb-3">Role</h3>
                <div className="flex gap-2 flex-wrap">
                  {(["student", "teacher", "admin"] as const).map((r) => (
                    <button
                      key={r}
                      disabled={data.role === r || roleMut.isPending}
                      onClick={() => roleMut.mutate(r)}
                      className={`px-4 py-2 text-xs capitalize ${
                        data.role === r
                          ? "bg-brand/5 text-brand/40 border border-brand/10"
                          : "border border-brand/15 hover:bg-brand/[0.04]"
                      } disabled:opacity-40`}
                    >
                      {data.role === r ? `Current: ${r}` : `Set ${r}`}
                    </button>
                  ))}
                </div>
              </section>

              <section className="mt-8">
                <h3 className="font-serif text-lg mb-3">Account actions</h3>
                <div className="flex gap-2 flex-wrap">
                  <button
                    onClick={() => activeMut.mutate(!data.is_active)}
                    disabled={activeMut.isPending}
                    className="inline-flex items-center gap-1.5 border border-brand/15 px-4 py-2 text-xs hover:bg-brand/[0.04] disabled:opacity-60"
                  >
                    {data.is_active ? (
                      <ShieldAlert className="h-3.5 w-3.5" />
                    ) : (
                      <ShieldCheck className="h-3.5 w-3.5" />
                    )}
                    {data.is_active ? "Deactivate account" : "Activate account"}
                  </button>
                  <button
                    onClick={() => resetMut.mutate()}
                    disabled={resetMut.isPending}
                    className="inline-flex items-center gap-1.5 border border-brand/15 px-4 py-2 text-xs hover:bg-brand/[0.04] disabled:opacity-60"
                  >
                    <KeyRound className="h-3.5 w-3.5" />
                    {resetMut.isPending ? "Sending…" : "Send password reset"}
                  </button>
                  <a
                    href={`mailto:${data.email}`}
                    className="inline-flex items-center gap-1.5 border border-brand/15 px-4 py-2 text-xs hover:bg-brand/[0.04]"
                  >
                    <Mail className="h-3.5 w-3.5" />
                    Email user
                  </a>
                  {currentUser?.id !== data.id && (
                    <button
                      onClick={() => {
                        if (
                          window.confirm(
                            `Sign in as ${data.full_name}? You can return to your admin session at any time from the banner at the top.`,
                          )
                        ) {
                          impersonateMut.mutate();
                        }
                      }}
                      disabled={impersonateMut.isPending}
                      className="inline-flex items-center gap-1.5 border border-accent/40 text-accent px-4 py-2 text-xs hover:bg-accent/5 disabled:opacity-60"
                    >
                      <UserCog className="h-3.5 w-3.5" />
                      {impersonateMut.isPending ? "Switching…" : "Impersonate user"}
                    </button>
                  )}
                </div>
              </section>

              <section className="mt-10 border-t border-destructive/20 pt-6">
                <h3 className="font-serif text-lg mb-2 text-destructive">Danger zone</h3>
                <p className="text-xs text-brand/55 mb-3">
                  Permanently deletes the user and all associated records. This can't be undone.
                </p>
                <button
                  onClick={() => {
                    if (
                      window.confirm(`Permanently delete ${data.email}? This cannot be undone.`)
                    ) {
                      deleteMut.mutate();
                    }
                  }}
                  disabled={deleteMut.isPending}
                  className="inline-flex items-center gap-1.5 border border-destructive/40 text-destructive px-4 py-2 text-xs hover:bg-destructive/5 disabled:opacity-60"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  {deleteMut.isPending ? "Deleting…" : "Delete user"}
                </button>
              </section>
            </>
          )}

          {tab === "enrollments" && <EnrollmentsTab userId={userId} />}
          {tab === "activity" && <ActivityTab userId={userId} />}
        </>
      )}
    </AppShell>
  );
}

function EnrollmentsTab({ userId }: { userId: string }) {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["admin-user-enrollments", userId],
    queryFn: () => getUserEnrollments(userId),
  });

  if (isLoading) return <div className="h-40 border border-brand/10 bg-white/40 animate-pulse" />;
  if (isError)
    return <QueryErrorPanel error={error} variant="compact" message={(error as Error)?.message} />;
  if (!data || data.length === 0)
    return <p className="text-sm text-brand/55">No enrollments yet.</p>;

  return (
    <ul className="divide-y divide-brand/10 border border-brand/10 bg-white/40">
      {data.map((e) => (
        <li key={e.id} className="px-4 py-3 flex items-center justify-between gap-4">
          <div className="min-w-0">
            <Link
              to="/admin/courses/$courseId/review"
              params={{ courseId: e.course_id }}
              className="text-sm font-medium text-brand hover:underline truncate block"
            >
              {e.course_title}
            </Link>
            <p className="text-xs text-brand/55">
              Enrolled {new Date(e.enrolled_at).toLocaleDateString()}
              {e.completed_at
                ? ` · Completed ${new Date(e.completed_at).toLocaleDateString()}`
                : ""}
            </p>
          </div>
          <div className="flex items-center gap-3 flex-shrink-0">
            <div className="w-24 h-1.5 bg-brand/10 overflow-hidden">
              <div
                className="h-full bg-accent"
                style={{ width: `${Math.min(100, e.progress_percent)}%` }}
              />
            </div>
            <span className="text-xs text-brand/60 w-10 text-right">
              {Math.round(e.progress_percent)}%
            </span>
          </div>
        </li>
      ))}
    </ul>
  );
}

function ActivityTab({ userId }: { userId: string }) {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["admin-user-activity", userId],
    queryFn: () => getUserActivity(userId, 50),
  });

  if (isLoading) return <div className="h-40 border border-brand/10 bg-white/40 animate-pulse" />;
  if (isError)
    return <QueryErrorPanel error={error} variant="compact" message={(error as Error)?.message} />;
  if (!data || data.length === 0)
    return <p className="text-sm text-brand/55">No recent activity.</p>;

  return (
    <ul className="divide-y divide-brand/10 border border-brand/10 bg-white/40">
      {data.map((a) => (
        <li key={a.id} className="px-4 py-3 flex items-start gap-3">
          <span className="mt-0.5 px-1.5 py-0.5 text-[10px] bg-brand/5 border border-brand/10 text-brand/60 uppercase tracking-wide">
            {a.kind}
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-sm">{a.summary}</p>
            <p className="text-[11px] text-brand/45">{new Date(a.created_at).toLocaleString()}</p>
          </div>
        </li>
      ))}
    </ul>
  );
}

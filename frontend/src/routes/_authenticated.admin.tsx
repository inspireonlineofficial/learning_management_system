import { useEffect } from "react";
import { createFileRoute, Link, Outlet, useNavigate, useRouterState } from "@tanstack/react-router";
import {
  BarChart3,
  BookOpen,
  CheckSquare,
  ClipboardCheck,
  FileText,
  Flag,
  Settings as SettingsIcon,
  ShieldCheck,
  ShoppingBag,
  Users,
} from "lucide-react";

import { useAuth } from "@/context/auth-context";

export const Route = createFileRoute("/_authenticated/admin")({
  component: AdminLayout,
});

type Tab = { to: string; label: string; icon: typeof BarChart3; exact?: boolean };
const tabs: Tab[] = [
  { to: "/admin", label: "Overview", icon: BarChart3, exact: true },
  { to: "/admin/users", label: "Users", icon: Users },
  { to: "/admin/rbac", label: "RBAC", icon: ShieldCheck },
  { to: "/admin/courses", label: "Course review", icon: BookOpen },
  { to: "/admin/approvals", label: "Approvals", icon: CheckSquare },
  { to: "/admin/moderation", label: "Moderation", icon: Flag },
  { to: "/admin/bookshop", label: "Bookshop", icon: ShoppingBag },
  { to: "/admin/analytics", label: "Analytics", icon: BarChart3 },
  { to: "/admin/notifications", label: "Notifications", icon: ClipboardCheck },
  { to: "/admin/audit-logs", label: "Audit", icon: FileText },
  { to: "/admin/system", label: "System", icon: SettingsIcon },
];

function AdminLayout() {
  const { user, isHydrated } = useAuth();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  useEffect(() => {
    if (!isHydrated || !user) return;
    if (user.role !== "admin") navigate({ to: "/403", replace: true });
  }, [user, isHydrated, navigate]);

  if (!isHydrated || !user || user.role !== "admin") return null;

  return (
    <div>
      <div className="border-b border-brand/10 bg-white/30">
        <div className="px-6 md:px-10 lg:px-14 flex gap-1 overflow-x-auto">
          {tabs.map((t) => {
            const active = t.exact ? pathname === t.to : pathname.startsWith(t.to);
            const Icon = t.icon;
            return (
              <Link
                key={t.to}
                to={t.to}
                className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 -mb-px whitespace-nowrap ${
                  active
                    ? "border-brand text-brand"
                    : "border-transparent text-brand/55 hover:text-brand"
                }`}
              >
                <Icon className="h-4 w-4" />
                {t.label}
              </Link>
            );
          })}
        </div>
      </div>
      <Outlet />
    </div>
  );
}

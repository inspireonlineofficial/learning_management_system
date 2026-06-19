import { Link, useNavigate, useRouterState } from "@tanstack/react-router";
import {
  Award,
  BarChart3,
  Bell,
  BookOpen,
  Calendar,
  CheckSquare,
  ClipboardList,
  Download,
  FileText,
  Flag,
  LayoutDashboard,
  Library,
  LogOut,
  MessageSquare,
  Radio,
  Search,
  Settings as SettingsIcon,
  ShieldCheck,
  ShoppingBag,
  Sparkles,
  Trophy,
  User,
  UserCog,
  Users,
} from "lucide-react";
import { useEffect, useState, type ComponentType, type ReactNode } from "react";
import { toast } from "sonner";

import { useAuth } from "@/context/auth-context";
import { getImpersonationOrigin, isImpersonating, stopImpersonation } from "@/lib/session";

type NavItem = { to: string; label: string; icon: ComponentType<{ className?: string }> };

const studentNav: NavItem[] = [
  { to: "/student", label: "Dashboard", icon: LayoutDashboard },
  { to: "/student/my-courses", label: "My courses", icon: Library },
  { to: "/student/progress", label: "Progress", icon: BarChart3 },
  { to: "/student/assessments", label: "Assessments", icon: ClipboardList },
  { to: "/student/assignments", label: "Assignments", icon: FileText },
  { to: "/student/live-classes", label: "Live classes", icon: Radio },
  { to: "/student/forum", label: "Community", icon: MessageSquare },
  { to: "/student/leaderboard", label: "Leaderboard", icon: Trophy },
  { to: "/student/points", label: "Points", icon: Sparkles },
  { to: "/student/certificates", label: "Certificates", icon: Award },
  { to: "/student/calendar", label: "Calendar", icon: Calendar },
  { to: "/student/bookshop", label: "Bookshop", icon: BookOpen },
  { to: "/courses", label: "Catalog", icon: Search },
  { to: "/student/downloads", label: "Downloads", icon: Download },
  { to: "/student/notifications", label: "Notifications", icon: Bell },
  { to: "/student/settings", label: "Settings", icon: SettingsIcon },
];

const teacherNav: NavItem[] = [
  { to: "/teacher", label: "Dashboard", icon: LayoutDashboard },
  { to: "/teacher/courses", label: "Courses", icon: Library },
  { to: "/teacher/quiz-builder", label: "Quizzes", icon: ClipboardList },
  { to: "/teacher/assignments", label: "Assignments", icon: FileText },
  { to: "/teacher/live", label: "Live classes", icon: Radio },
  { to: "/teacher/analytics", label: "Analytics", icon: BarChart3 },
  { to: "/teacher/notifications", label: "Notifications", icon: Bell },
  { to: "/teacher/settings", label: "Settings", icon: SettingsIcon },
];

const adminNav: NavItem[] = [
  { to: "/admin", label: "Overview", icon: LayoutDashboard },
  { to: "/admin/users", label: "Users", icon: Users },
  { to: "/admin/rbac", label: "RBAC", icon: ShieldCheck },
  { to: "/admin/courses", label: "Course review", icon: BookOpen },
  { to: "/admin/approvals", label: "Approvals", icon: CheckSquare },
  { to: "/admin/moderation", label: "Moderation", icon: Flag },
  { to: "/admin/bookshop", label: "Bookshop", icon: ShoppingBag },
  { to: "/admin/slides", label: "Slides", icon: Sparkles },
  { to: "/admin/analytics", label: "Analytics", icon: BarChart3 },
  { to: "/admin/notifications", label: "Notifications", icon: Bell },
  { to: "/admin/audit-logs", label: "Audit logs", icon: FileText },
  { to: "/admin/system", label: "System", icon: SettingsIcon },
  { to: "/admin/settings", label: "Settings", icon: User },
];

function navForRole(role: string | undefined): NavItem[] {
  if (role === "admin") return adminNav;
  if (role === "teacher") return teacherNav;
  return studentNav;
}

export function AppShell({
  children,
  title,
  eyebrow,
}: {
  children: ReactNode;
  title?: string;
  eyebrow?: string;
}) {
  const { user, setSession, signOut } = useAuth();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const nav = navForRole(user?.role);

  const [impersonating, setImpersonating] = useState(false);
  const [originName, setOriginName] = useState<string | null>(null);

  useEffect(() => {
    const sync = () => {
      setImpersonating(isImpersonating());
      const origin = getImpersonationOrigin();
      setOriginName(origin?.user.full_name ?? origin?.user.email ?? null);
    };
    sync();
    window.addEventListener("storage", sync);
    return () => window.removeEventListener("storage", sync);
  }, [user?.id]);

  const handleSignOut = async () => {
    try {
      await signOut();
      toast.success("Signed out");
    } catch {
      toast.error("Could not sign out");
    }
  };

  const handleExitImpersonation = () => {
    const origin = stopImpersonation();
    if (origin) {
      setSession(origin);
      toast.success("Returned to your admin session");
      navigate({ to: "/admin/users" });
    }
  };

  return (
    <div className="min-h-screen overflow-x-hidden bg-surface text-brand font-sans">
      {impersonating && (
        <div className="sticky top-0 z-50 bg-accent text-white px-4 py-2 flex items-center justify-between gap-3 text-xs">
          <div className="flex items-center gap-2 min-w-0">
            <UserCog className="h-3.5 w-3.5 flex-shrink-0" />
            <span className="truncate">
              Viewing as <strong>{user?.full_name ?? user?.email}</strong>
              {originName ? ` · originally signed in as ${originName}` : ""}
            </span>
          </div>
          <button
            onClick={handleExitImpersonation}
            className="flex-shrink-0 inline-flex items-center gap-1.5 border border-white/40 px-3 py-1 hover:bg-white/10"
          >
            Exit impersonation
          </button>
        </div>
      )}
      <div className="flex min-h-screen flex-col lg:flex-row">
        <aside className="lg:w-64 lg:fixed lg:inset-y-0 lg:flex lg:flex-col border-b lg:border-b-0 lg:border-r border-brand/10 bg-surface/95 backdrop-blur">
          <div className="px-6 py-6 lg:py-8">
            <Link to="/" className="font-serif italic text-2xl text-accent tracking-tight">
              Inspire LMS
            </Link>
            {user?.role && user.role !== "student" && (
              <p className="mt-1 eyebrow text-brand/45">{user.role} portal</p>
            )}
          </div>
          <nav className="px-3 lg:px-4 flex lg:flex-col gap-1 overflow-x-auto lg:overflow-y-auto lg:overflow-x-visible lg:flex-1 pb-2">
            {nav.map(({ to, label, icon: Icon }) => {
              const sectionRoot = to === "/admin" || to === "/student" || to === "/teacher";
              const active = pathname === to || (!sectionRoot && pathname.startsWith(`${to}/`));
              return (
                <Link
                  key={to}
                  to={to}
                  className={`flex items-center gap-3 px-4 py-3 text-sm font-medium whitespace-nowrap transition-colors ${
                    active
                      ? "bg-brand text-white"
                      : "text-brand/70 hover:text-brand hover:bg-brand/[0.03]"
                  }`}
                >
                  <Icon className="h-4 w-4" />
                  {label}
                </Link>
              );
            })}
          </nav>
          <div className="hidden lg:block p-4 border-t border-brand/10">
            <div className="flex items-center gap-3 px-2 py-3">
              <div className="h-9 w-9 grid place-items-center bg-brand/10 text-brand">
                <User className="h-4 w-4" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium truncate">{user?.full_name ?? "Scholar"}</p>
                <p className="text-xs text-brand/55 truncate">{user?.email}</p>
              </div>
            </div>
            <button
              onClick={handleSignOut}
              className="mt-2 w-full flex items-center justify-center gap-2 px-3 py-2 text-xs font-medium text-brand/70 hover:text-brand border border-brand/15 hover:bg-brand/[0.03] transition-colors"
            >
              <LogOut className="h-3.5 w-3.5" />
              Sign out
            </button>
          </div>
        </aside>

        <main className="min-w-0 flex-1 lg:ml-64">
          {(title || eyebrow) && (
            <header className="px-6 md:px-10 lg:px-14 pt-10 lg:pt-14 pb-6 border-b border-brand/10">
              {eyebrow && <p className="eyebrow text-accent mb-3">{eyebrow}</p>}
              {title && <h1 className="font-serif text-4xl lg:text-5xl text-balance">{title}</h1>}
            </header>
          )}
          <div className="min-w-0 px-6 md:px-10 lg:px-14 py-10">{children}</div>
        </main>
      </div>
    </div>
  );
}

export function StatCard({
  label,
  value,
  hint,
}: {
  label: string;
  value: ReactNode;
  hint?: string;
}) {
  return (
    <div className="border border-brand/10 bg-white/40 p-6">
      <p className="eyebrow text-brand/45">{label}</p>
      <p className="mt-3 font-serif text-4xl">{value}</p>
      {hint && <p className="mt-2 text-xs text-brand/55">{hint}</p>}
    </div>
  );
}

export function SectionHeading({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="flex items-end justify-between mb-6">
      <h2 className="font-serif text-2xl">{title}</h2>
      {action}
    </div>
  );
}

export function EmptyState({
  icon: Icon = BookOpen,
  title,
  description,
  action,
}: {
  icon?: ComponentType<{ className?: string }>;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="border border-dashed border-brand/15 px-8 py-16 text-center">
      <Icon className="h-8 w-8 mx-auto text-brand/30" />
      <h3 className="mt-4 font-serif text-2xl">{title}</h3>
      {description && <p className="mt-2 text-sm text-brand/55 max-w-md mx-auto">{description}</p>}
      {action && <div className="mt-6 flex justify-center">{action}</div>}
    </div>
  );
}

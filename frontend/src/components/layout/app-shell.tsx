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
  Menu,
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
  X,
} from "lucide-react";
import { useEffect, useState, type ComponentType, type ReactNode } from "react";
import { toast } from "sonner";

import { BrandLogo } from "@/components/layout/brand-logo";
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
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

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

  useEffect(() => {
    setIsMobileMenuOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!isMobileMenuOpen) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setIsMobileMenuOpen(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [isMobileMenuOpen]);

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
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy-light font-sans antialiased">
      {impersonating && (
        <div className="sticky top-0 z-50 bg-bronze text-white px-4 py-2.5 flex items-center justify-between gap-3 text-xs">
          <div className="flex items-center gap-2 min-w-0">
            <UserCog className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span className="truncate">
              Viewing as <strong>{user?.full_name ?? user?.email}</strong>
              {originName ? ` · originally signed in as ${originName}` : ""}
            </span>
          </div>
          <button
            type="button"
            onClick={handleExitImpersonation}
            className="shrink-0 inline-flex items-center gap-1.5 rounded-lg border border-white/40 px-3 py-1.5 font-semibold hover:bg-white/15 transition-colors"
          >
            Exit impersonation
          </button>
        </div>
      )}

      {/* Mobile Header */}
      <header className="lg:hidden sticky top-0 z-40 flex items-center justify-between h-16 border-b border-borderui bg-white/95 backdrop-blur-md px-4 sm:px-6">
        <div className="flex items-center gap-3 min-w-0">
          <BrandLogo imageClassName="max-h-9 max-w-[140px]" />
          {user?.role && user.role !== "student" && (
            <span className="text-xs font-bold tracking-[0.12em] uppercase text-bronze border-l border-borderui pl-3 truncate">
              {user.role}
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
          className="inline-flex items-center justify-center min-w-[44px] min-h-11 rounded-lg border border-borderui text-navy hover:bg-stoneband transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
          aria-expanded={isMobileMenuOpen}
          aria-label="Toggle navigation menu"
        >
          {isMobileMenuOpen ? (
            <X className="h-5 w-5" aria-hidden="true" />
          ) : (
            <Menu className="h-5 w-5" aria-hidden="true" />
          )}
        </button>
      </header>

      {/* Mobile Menu Drawer Overlay */}
      {isMobileMenuOpen && (
        <div className="lg:hidden fixed inset-0 z-50 flex">
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-navy/50 backdrop-blur-xs transition-opacity"
            onClick={() => setIsMobileMenuOpen(false)}
          />

          {/* Drawer Content */}
          <div className="relative flex w-[85%] max-w-xs flex-1 flex-col bg-white border-r border-borderui h-full p-5 shadow-lg animate-in slide-in-from-left duration-200 ease-out z-50">
            <div className="flex items-center justify-between pb-4 border-b border-borderui">
              <div className="flex items-center gap-2.5 min-w-0">
                <BrandLogo imageClassName="max-h-9 max-w-[135px]" />
                {user?.role && user.role !== "student" && (
                  <span className="text-[10px] font-bold tracking-[0.12em] uppercase text-bronze border-l border-borderui pl-2.5 truncate">
                    {user.role}
                  </span>
                )}
              </div>
              <button
                type="button"
                onClick={() => setIsMobileMenuOpen(false)}
                className="inline-flex items-center justify-center min-w-[40px] min-h-[40px] rounded-lg text-navy hover:bg-stoneband transition-colors"
                aria-label="Close menu"
              >
                <X className="h-5 w-5" aria-hidden="true" />
              </button>
            </div>

            {/* Mobile Nav */}
            <nav
              className="flex-1 overflow-y-auto py-4 space-y-1 pr-1"
              aria-label="Portal navigation"
            >
              {nav.map(({ to, label, icon: Icon }) => {
                const sectionRoot = to === "/admin" || to === "/student" || to === "/teacher";
                const active = pathname === to || (!sectionRoot && pathname.startsWith(`${to}/`));
                return (
                  <Link
                    key={to}
                    to={to}
                    className={`flex items-center gap-3 min-h-11 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                      active
                        ? "bg-navy text-white shadow-xs"
                        : "text-navy-light hover:text-navy hover:bg-stoneband"
                    }`}
                  >
                    <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
                    <span className="truncate">{label}</span>
                  </Link>
                );
              })}
            </nav>

            {/* User Profile & Sign Out */}
            <div className="pt-4 border-t border-borderui">
              <div className="flex items-center gap-3 px-2 py-2">
                <div className="h-9 w-9 rounded-full grid place-items-center bg-stoneband text-navy shrink-0">
                  <User className="h-4 w-4" aria-hidden="true" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-semibold text-navy truncate">
                    {user?.full_name ?? "Scholar"}
                  </p>
                  <p className="text-xs text-slatebody truncate">{user?.email}</p>
                </div>
              </div>
              <button
                type="button"
                onClick={handleSignOut}
                className="mt-2 w-full inline-flex items-center justify-center gap-2 min-h-11 px-3 py-2 rounded-lg text-xs font-semibold text-navy border border-borderui hover:bg-stoneband transition-colors"
              >
                <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
                Sign out
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="flex min-h-screen flex-col lg:flex-row">
        {/* Desktop Sidebar */}
        <aside className="hidden lg:flex lg:w-64 lg:fixed lg:inset-y-0 lg:flex-col border-r border-borderui bg-white">
          <div className="px-6 py-6 border-b border-borderui/70">
            <BrandLogo imageClassName="max-h-11 max-w-[165px]" />
            {user?.role && user.role !== "student" && (
              <p className="mt-2 text-xs font-bold tracking-[0.12em] uppercase text-bronze">
                {user.role} portal
              </p>
            )}
          </div>
          <nav
            className="px-3.5 py-4 flex flex-col gap-1 overflow-y-auto flex-1"
            aria-label="Portal navigation"
          >
            {nav.map(({ to, label, icon: Icon }) => {
              const sectionRoot = to === "/admin" || to === "/student" || to === "/teacher";
              const active = pathname === to || (!sectionRoot && pathname.startsWith(`${to}/`));
              return (
                <Link
                  key={to}
                  to={to}
                  className={`flex items-center gap-3 min-h-[42px] px-3.5 py-2.5 rounded-lg text-sm font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-navy ${
                    active
                      ? "bg-navy text-white shadow-xs"
                      : "text-navy-light hover:text-navy hover:bg-stoneband"
                  }`}
                >
                  <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
                  <span className="truncate">{label}</span>
                </Link>
              );
            })}
          </nav>
          <div className="p-4 border-t border-borderui bg-canvas/60">
            <div className="flex items-center gap-3 px-2 py-2">
              <div className="h-9 w-9 rounded-full grid place-items-center bg-stoneband text-navy shrink-0">
                <User className="h-4 w-4" aria-hidden="true" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-semibold text-navy truncate">
                  {user?.full_name ?? "Scholar"}
                </p>
                <p className="text-xs text-slatebody truncate">{user?.email}</p>
              </div>
            </div>
            <button
              type="button"
              onClick={handleSignOut}
              className="mt-2 w-full inline-flex items-center justify-center gap-2 min-h-[40px] px-3 py-2 rounded-lg text-xs font-semibold text-navy bg-white border border-borderui hover:bg-stoneband transition-colors"
            >
              <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
              Sign out
            </button>
          </div>
        </aside>

        <main className="min-w-0 flex-1 lg:ml-64">
          {(title || eyebrow) && (
            <header className="px-4 sm:px-6 md:px-10 lg:px-12 pt-8 lg:pt-12 pb-6 border-b border-borderui bg-white/70">
              {eyebrow && (
                <p className="text-xs font-bold tracking-[0.12em] uppercase text-bronze mb-2">
                  {eyebrow}
                </p>
              )}
              {title && (
                <h1 className="font-serif text-3xl sm:text-4xl lg:text-[2.65rem] font-bold text-navy tracking-tight text-balance">
                  {title}
                </h1>
              )}
            </header>
          )}
          <div className="min-w-0 px-4 sm:px-6 md:px-10 lg:px-12 py-8 sm:py-10">{children}</div>
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
    <div className="rounded-xl border border-borderui bg-white p-5 sm:p-6 shadow-sm">
      <p className="text-xs font-bold tracking-[0.12em] uppercase text-bronze">{label}</p>
      <p className="mt-2.5 font-serif text-3xl sm:text-4xl font-bold text-navy">{value}</p>
      {hint && <p className="mt-1.5 text-xs sm:text-sm text-slatebody">{hint}</p>}
    </div>
  );
}

export function SectionHeading({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-3 mb-6">
      <h2 className="font-serif text-2xl sm:text-3xl font-bold text-navy tracking-tight">
        {title}
      </h2>
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
    <div className="rounded-lg border border-borderui bg-white px-6 py-12 sm:py-14 text-center">
      <Icon className="mx-auto h-7 w-7 text-slatebody/70" aria-hidden="true" />
      <h3 className="mt-4 font-serif text-xl font-semibold text-navy">{title}</h3>
      {description && (
        <p className="mt-2 text-sm text-slatebody max-w-md mx-auto leading-relaxed">
          {description}
        </p>
      )}
      {action && <div className="mt-6 flex justify-center">{action}</div>}
    </div>
  );
}

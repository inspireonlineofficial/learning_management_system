import { apiRequest } from "./client";
import { getStudentAnalytics } from "./analytics";
import { listAuditLogs } from "./audit";

export type AdminUser = {
  id: string;
  email: string;
  full_name: string;
  role: "student" | "teacher" | "admin";
  avatar_url?: string | null;
  is_active: boolean;
  created_at: string;
  last_sign_in_at?: string | null;
};

type BackendAdminUser = Omit<AdminUser, "is_active"> & {
  is_active?: boolean;
  status?: "active" | "inactive" | "disabled" | string;
};

export type Paginated<T> = {
  data: T[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

type UsersListResponse = Paginated<BackendAdminUser> & {
  users?: BackendAdminUser[];
};

function normalizeAdminUser(user: BackendAdminUser): AdminUser {
  return {
    ...user,
    is_active: user.is_active ?? user.status === "active",
  };
}

export function listUsers(
  params: {
    search?: string;
    role?: "student" | "teacher" | "admin";
    page?: number;
    limit?: number;
  } = {},
) {
  return apiRequest<UsersListResponse>("/v1/admin/users", { auth: true, query: params }).then(
    (result) => ({
      data: (result.data ?? result.users ?? []).map(normalizeAdminUser),
      meta: result.meta,
    }),
  );
}

export function updateUserRole(userId: string, role: AdminUser["role"]) {
  return apiRequest<AdminUser>(`/v1/admin/users/${encodeURIComponent(userId)}`, {
    method: "PATCH",
    auth: true,
    body: { role },
  });
}

export function setUserActive(userId: string, isActive: boolean) {
  return apiRequest<AdminUser>(`/v1/admin/users/${encodeURIComponent(userId)}`, {
    method: "PATCH",
    auth: true,
    body: { status: isActive ? "active" : "inactive" },
  });
}

export type AdminStats = {
  total_users: number;
  active_users_30d: number;
  total_courses: number;
  published_courses: number;
  total_enrollments: number;
  revenue_30d_cents?: number;
  currency?: string;
};

export function getAdminStats() {
  return apiRequest<AdminStats>("/v1/admin/stats", { auth: true });
}

// ---------- User detail ----------

export type AdminUserEnrollment = {
  id: string;
  course_id: string;
  course_title: string;
  progress_percent: number;
  enrolled_at: string;
  completed_at?: string | null;
};

export type AdminUserActivity = {
  id: string;
  kind: string; // e.g. "login", "course_enrolled", "lesson_completed"
  summary: string;
  created_at: string;
};

export function getAdminUser(userId: string) {
  return apiRequest<BackendAdminUser>(`/v1/admin/users/${encodeURIComponent(userId)}`, {
    auth: true,
  }).then(normalizeAdminUser);
}

export function getUserEnrollments(userId: string) {
  return getStudentAnalytics(userId)
    .then((analytics) =>
      analytics.course_progress.map((course) => ({
        id: `${userId}:${course.course_id}`,
        course_id: course.course_id,
        course_title: course.course_title,
        progress_percent: course.progress_percent,
        enrolled_at: course.enrolled_at,
        completed_at: course.progress_percent >= 100 ? course.enrolled_at : null,
      })),
    )
    .catch(() => []);
}

export function getUserActivity(userId: string, limit = 25) {
  return listAuditLogs({ actor_id: userId, limit }).then((result) =>
    result.items.map((entry) => ({
      id: entry.id,
      kind: entry.action,
      summary: `${entry.action.replaceAll("_", " ")}${
        entry.target_type ? ` on ${entry.target_type}` : ""
      }`,
      created_at: entry.created_at,
    })),
  );
}

export function sendPasswordReset(userId: string) {
  return apiRequest<{ ok: true }>(
    `/v1/admin/users/${encodeURIComponent(userId)}/force-password-reset`,
    {
      method: "POST",
      auth: true,
    },
  );
}

export function deleteUser(userId: string) {
  return setUserActive(userId, false).then(() => ({ ok: true }));
}

export type ImpersonationToken = {
  access_token: string;
  refresh_token: string;
  user: {
    id: string;
    email: string;
    full_name: string;
    role: "student" | "teacher" | "admin";
    avatar_url?: string | null;
    onboarded?: boolean;
  };
};

export function impersonateUser(userId: string) {
  return apiRequest<ImpersonationToken>(
    `/v1/admin/users/${encodeURIComponent(userId)}/impersonate`,
    {
      method: "POST",
      auth: true,
    },
  ).then((result) => ({
    ...result,
    user: {
      ...result.user,
      onboarded: result.user.onboarded ?? true,
    },
  }));
}

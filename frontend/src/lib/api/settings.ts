import { apiRequest } from "./client";

export type UserSettings = {
  email_notifications: boolean;
  push_notifications: boolean;
  newsletter_opt_in: boolean;
  language: string;
  timezone: string;
};
export const getMySettings = () =>
  apiRequest<Partial<UserSettings> & { full_name?: string; email?: string }>("/v1/auth/me", {
    auth: true,
  }).then((profile) => ({
    email_notifications: true,
    push_notifications: true,
    newsletter_opt_in: false,
    language: "en",
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    ...profile,
  }));
export const updateMySettings = (input: Partial<UserSettings>) =>
  Promise.resolve({
    email_notifications: input.email_notifications ?? true,
    push_notifications: input.push_notifications ?? true,
    newsletter_opt_in: input.newsletter_opt_in ?? false,
    language: input.language ?? "en",
    timezone: input.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone,
  });
export const changePassword = (input: { current_password: string; new_password: string }) =>
  apiRequest<{ ok: true }>("/v1/auth/me/change-password", {
    method: "POST",
    auth: true,
    body: input,
  });

// ---------- Admin platform settings ----------

export type PlatformSettings = {
  site_name: string;
  support_email: string;
  default_currency: string;
  default_language: string;
  default_timezone: string;
  allow_self_signup: boolean;
  require_email_verification: boolean;
  teacher_application_open: boolean;
  maintenance_mode: boolean;
};

export const getPlatformSettings = () =>
  apiRequest<{
    platform_name: string;
    default_timezone: string;
    maintenance_mode: boolean;
    feature_flags?: Record<string, unknown>;
  }>("/v1/admin/system/settings", { auth: true }).then((settings) => ({
    site_name: settings.platform_name,
    support_email: "support@inspire.local",
    default_currency: "USD",
    default_language: "en",
    default_timezone: settings.default_timezone,
    allow_self_signup: Boolean(settings.feature_flags?.allow_self_signup ?? true),
    require_email_verification: Boolean(settings.feature_flags?.require_email_verification ?? true),
    teacher_application_open: Boolean(settings.feature_flags?.teacher_application_open ?? true),
    maintenance_mode: settings.maintenance_mode,
  }));

export const updatePlatformSettings = (input: Partial<PlatformSettings>) =>
  apiRequest<{
    platform_name: string;
    default_timezone: string;
    maintenance_mode: boolean;
    feature_flags?: Record<string, unknown>;
  }>("/v1/admin/system/settings", {
    method: "PATCH",
    auth: true,
    body: {
      platform_name: input.site_name,
      default_timezone: input.default_timezone,
      maintenance_mode: input.maintenance_mode,
      feature_flags: {
        allow_self_signup: input.allow_self_signup,
        require_email_verification: input.require_email_verification,
        teacher_application_open: input.teacher_application_open,
      },
    },
  }).then(() => getPlatformSettings());

// ---------- Admin gamification (points-value) config ----------

export type PointsConfig = {
  lesson_completed: number;
  quiz_passed: number;
  quiz_perfect_bonus: number;
  assignment_submitted: number;
  assignment_graded_bonus: number;
  live_class_attended: number;
  daily_streak_bonus: number;
  forum_post_created: number;
  forum_helpful_vote: number;
  level_thresholds: number[]; // cumulative points required for each level
};

export const getPointsConfig = () =>
  Promise.resolve({
    lesson_completed: 10,
    quiz_passed: 25,
    quiz_perfect_bonus: 10,
    assignment_submitted: 10,
    assignment_graded_bonus: 5,
    live_class_attended: 10,
    daily_streak_bonus: 5,
    forum_post_created: 2,
    forum_helpful_vote: 1,
    level_thresholds: [0, 100, 300, 700, 1500],
  });

export const updatePointsConfig = (input: Partial<PointsConfig>) =>
  apiRequest<{
    points_per_video: number;
    points_per_quiz_pass: number;
    bonus_points_perfect_score: number;
  }>("/v1/admin/points/config", {
    method: "PATCH",
    auth: true,
    body: {
      points_per_video: input.lesson_completed,
      points_per_quiz_pass: input.quiz_passed,
      bonus_points_perfect_score: input.quiz_perfect_bonus,
    },
  }).then(() => ({ ...defaultPointsConfig, ...input }));

const defaultPointsConfig: PointsConfig = {
  lesson_completed: 10,
  quiz_passed: 25,
  quiz_perfect_bonus: 10,
  assignment_submitted: 10,
  assignment_graded_bonus: 5,
  live_class_attended: 10,
  daily_streak_bonus: 5,
  forum_post_created: 2,
  forum_helpful_vote: 1,
  level_thresholds: [0, 100, 300, 700, 1500],
};

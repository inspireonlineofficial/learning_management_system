import { apiRequest } from "./client";

export type PlatformAnalytics = {
  total_users: number;
  active_users_30d: number;
  enrollments: number;
  revenue: number;
  courses_published: number;
  trend: Array<{ date: string; users: number; revenue: number }>;
};
export type CourseAnalytics = {
  course_id: string;
  enrolled: number;
  completed: number;
  avg_progress: number;
  revenue: number;
  rating: number;
};
export type StudentAnalytics = {
  student_id: string;
  enrolled_courses: number;
  hours_learned: number;
  avg_score: number;
  streak: number;
  certificates: number;
};

export type StudentAnalyticsDetail = {
  student_id: string;
  points_history_30d: Array<{ date: string; points: number }>;
  course_progress: Array<{
    course_id: string;
    course_title: string;
    progress_percent: number;
    enrolled_at: string;
  }>;
  global_rank: number;
};

type BackendTeacherAnalytics = {
  teacher_id: string;
  courses?: Array<{
    course_id: string;
    course_title: string;
    total_enrolled: number;
    completion_rate: number;
    average_quiz_score: number;
    revenue: number;
  }>;
  total_revenue?: number;
};

type BackendPlatformAnalytics = {
  total_courses?: { all?: number; published?: number; free?: number; paid?: number };
  total_students?: number;
  active_students_30d?: number;
  total_enrollments?: { all?: number; free?: number; paid?: number };
  revenue?: { this_month?: number; this_year?: number; all_time?: number };
  daily_active_users?: Array<{ date: string; active_users: number }>;
};

export const getPlatformAnalytics = () =>
  apiRequest<BackendPlatformAnalytics>("/v1/admin/analytics/overview", { auth: true }).then(
    (result) => ({
      total_users: result.total_students ?? 0,
      active_users_30d: result.active_students_30d ?? 0,
      enrollments: result.total_enrollments?.all ?? 0,
      revenue: result.revenue?.this_month ?? 0,
      courses_published: result.total_courses?.published ?? result.total_courses?.all ?? 0,
      trend: (result.daily_active_users ?? []).map((entry) => ({
        date: entry.date,
        users: entry.active_users,
        revenue: 0,
      })),
    }),
  );
export const listCourseAnalytics = () =>
  apiRequest<{ items: CourseAnalytics[] }>("/v1/admin/analytics/courses", { auth: true });
export const getCourseAnalytics = (courseId: string) =>
  apiRequest<CourseAnalytics>(`/v1/admin/analytics/courses/${courseId}`, { auth: true });
export const listStudentAnalytics = () =>
  apiRequest<{ items: StudentAnalytics[] }>("/v1/admin/analytics/students", { auth: true });
export const getStudentAnalytics = (studentId: string) =>
  apiRequest<StudentAnalyticsDetail>(`/v1/admin/analytics/students/${studentId}`, { auth: true });
export const getTeacherAnalytics = () =>
  apiRequest<BackendTeacherAnalytics>("/v1/teacher/analytics", { auth: true }).then((result) => {
    const courses = result.courses ?? [];
    return {
      total_users: courses.reduce((sum, course) => sum + (course.total_enrolled ?? 0), 0),
      active_users_30d: courses.reduce((sum, course) => sum + (course.total_enrolled ?? 0), 0),
      enrollments: courses.reduce((sum, course) => sum + (course.total_enrolled ?? 0), 0),
      revenue:
        result.total_revenue ?? courses.reduce((sum, course) => sum + (course.revenue ?? 0), 0),
      courses_published: courses.length,
      trend: courses.map((course) => ({
        date: course.course_title,
        users: course.total_enrolled ?? 0,
        revenue: course.revenue ?? 0,
      })),
    } satisfies PlatformAnalytics;
  });
export const getTeacherStudentAnalytics = (studentId: string) =>
  apiRequest<StudentAnalyticsDetail>(`/v1/teacher/analytics/students/${studentId}`, {
    auth: true,
  });

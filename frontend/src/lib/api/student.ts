import { apiRequest } from "./client";
import type { CourseDetail, CourseSummary, Lesson, Module } from "./courses";

export type Enrollment = {
  id: string;
  course: CourseSummary;
  enrolled_at: string;
  progress_percent: number;
  last_accessed_at?: string | null;
  next_lesson?: { id: string; title: string; module_id: string } | null;
  completed_at?: string | null;
};

export type EnrollmentList = {
  data: Enrollment[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export function listMyEnrollments(
  params: { status?: "active" | "completed"; page?: number; limit?: number } = {},
) {
  return apiRequest<EnrollmentList>("/v1/student/enrollments", {
    auth: true,
    query: params,
  });
}

export function enroll(courseId: string) {
  return apiRequest<Enrollment>("/v1/enrollments", {
    method: "POST",
    auth: true,
    body: { course_id: courseId },
  });
}

export type CourseProgress = {
  course_id: string;
  progress_percent: number;
  completed_lessons: string[];
  current_lesson_id?: string | null;
  modules: Module[];
};

export function getCourseProgress(courseId: string) {
  return listMyEnrollments().then((enrollments) => {
    const enrollment = enrollments.data.find((item) => item.course.id === courseId);
    return {
      course_id: courseId,
      progress_percent: enrollment?.progress_percent ?? 0,
      completed_lessons: [],
      current_lesson_id: enrollment?.next_lesson?.id ?? null,
      modules: [],
    };
  });
}

export type LessonContent = Lesson & {
  video_url?: string | null;
  body_html?: string | null;
  resources?: { id: string; title: string; url: string }[];
};

export function getLesson(courseId: string, lessonId: string) {
  void courseId;
  return apiRequest<{ signed_url: string }>(
    `/v1/stream/lessons/${encodeURIComponent(lessonId)}/signed-url`,
    { auth: true },
  ).then(
    (result) =>
      ({
        id: lessonId,
        title: "Lesson",
        type: "video",
        duration_minutes: 0,
        video_url: result.signed_url,
      }) as LessonContent,
  );
}

export function completeLesson(courseId: string, lessonId: string) {
  return apiRequest<{ progress_percent: number }>(
    `/v1/enrollments/${encodeURIComponent(courseId)}/lessons/${encodeURIComponent(lessonId)}/progress`,
    { method: "POST", auth: true, body: { progress_percent: 100, completed: true } },
  );
}

export type StudentDashboard = {
  stats: {
    enrolled_courses: number;
    completed_courses: number;
    hours_learned: number;
    points: number;
    streak_days?: number;
  };
  continue_learning: Enrollment[];
  upcoming_live?: { id: string; title: string; starts_at: string; course_title: string }[];
  recent_achievements?: { id: string; title: string; icon?: string; earned_at: string }[];
};

export function getStudentDashboard() {
  return apiRequest<StudentDashboard>("/v1/student/dashboard", { auth: true });
}

export type CourseWithEnrollment = CourseDetail & { is_enrolled?: boolean };

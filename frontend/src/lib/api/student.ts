import { apiRequest } from "./client";
import {
  getCourse,
  type CourseDetail,
  type CourseSummary,
  type Lesson,
  type Module,
} from "./courses";
import { listMyAssignments } from "./assignments";
import { listCourseQuizzes } from "./quizzes";

export type Enrollment = {
  id: string;
  // Course may be null when the underlying course was soft-deleted (e.g. by
  // an admin using DELETE /v1/admin/courses/{id}) between the time the
  // student enrolled and now. The backend filters these out of the list, but
  // we type it as nullable so any race / older client stays safe.
  course: CourseSummary | null;
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
  total_lessons: number;
  quizzes_total: number;
  quizzes_completed: number;
  assignments_total: number;
  assignments_completed: number;
};

export async function getCourseProgress(courseId: string) {
  const [enrollments, course, quizzes, assignments] = await Promise.all([
    listMyEnrollments({ limit: 100 }),
    getCourse(courseId),
    listCourseQuizzes(courseId),
    listMyAssignments({ limit: 100 }),
  ]);
  const enrollment = enrollments.data.find((item) => item.course?.id === courseId);
  const modules = course.modules ?? [];
  const lessons = modules.flatMap((module) => module.lessons);
  // One request for all lesson progress (previously one request per lesson).
  const courseProgress = await getEnrollmentProgress(courseId).catch(() => null);
  const completedLessons = (courseProgress?.lessons ?? [])
    .filter((item) => item.completed)
    .map((item) => item.lesson_id);
  const courseAssignments = assignments.data.filter(
    (assignment) => assignment.course_id === courseId,
  );
  const completedAssignments = courseAssignments.filter((assignment) =>
    ["submitted", "graded", "revision_requested"].includes(assignment.status),
  );

  return {
    course_id: courseId,
    progress_percent: enrollment?.progress_percent ?? 0,
    completed_lessons: completedLessons,
    current_lesson_id: enrollment?.next_lesson?.id ?? null,
    modules,
    total_lessons: lessons.length,
    quizzes_total: quizzes.data.length,
    quizzes_completed: quizzes.data.filter((quiz) =>
      ["completed", "passed", "failed"].includes(quiz.status ?? ""),
    ).length,
    assignments_total: courseAssignments.length,
    assignments_completed: completedAssignments.length,
  };
}

export type LessonContent = Lesson & {
  video_url?: string | null;
  hls_url?: string | null;
  has_hls?: boolean;
  poster_url?: string | null;
  body_html?: string | null;
  resources?: { id: string; title: string; url: string }[];
};

export function getLesson(courseId: string, lessonId: string) {
  void courseId;
  return apiRequest<{
    signed_url: string;
    hls_url?: string;
    has_hls?: boolean;
    poster_url?: string;
    content_type?: string;
  }>(`/v1/stream/lessons/${encodeURIComponent(lessonId)}/signed-url`, { auth: true }).then(
    (result) =>
      ({
        id: lessonId,
        title: "Lesson",
        type: "video",
        duration_minutes: 0,
        video_url: result.signed_url,
        hls_url: result.hls_url,
        has_hls: result.has_hls,
        poster_url: result.poster_url,
      }) as LessonContent,
  );
}

export type LessonProgress = {
  lesson_id: string;
  position_seconds: number;
  watched_percent: number;
  completed: boolean;
  last_watched_at?: string;
};

export type EnrollmentProgress = {
  enrollment_id: string;
  course_id: string;
  progress_percent: number;
  completed_at?: string | null;
  lessons: LessonProgress[];
};

export function getEnrollmentProgress(courseId: string) {
  return apiRequest<EnrollmentProgress>(
    `/v1/enrollments/${encodeURIComponent(courseId)}/progress`,
    { auth: true },
  );
}

export function saveLessonProgress(
  courseId: string,
  lessonId: string,
  progress: { position_seconds: number; watched_percent: number; completed?: boolean },
  options: { keepalive?: boolean } = {},
) {
  return apiRequest<LessonProgress>(
    `/v1/enrollments/${encodeURIComponent(courseId)}/lessons/${encodeURIComponent(lessonId)}/progress`,
    {
      method: "POST",
      auth: true,
      keepalive: options.keepalive,
      body: {
        position_seconds: Math.max(0, Math.floor(progress.position_seconds)),
        watched_percent: Math.min(
          100,
          Math.max(0, Math.round(progress.watched_percent * 100) / 100),
        ),
        completed: Boolean(progress.completed),
      },
    },
  );
}

/**
 * Explicit "Mark complete". The API field is `watched_percent` — this used to
 * send `progress_percent`, which the backend ignores, so the server saw 0%
 * watched and never recorded the completion even though the UI said it had.
 * Video lessons may still be deferred by the server until a plausible amount
 * of the video has been watched; callers must check `completed`.
 */
export function completeLesson(courseId: string, lessonId: string, positionSeconds = 0) {
  return saveLessonProgress(courseId, lessonId, {
    position_seconds: positionSeconds,
    watched_percent: 100,
    completed: true,
  });
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

import { apiRequest } from "./client";
import type { Chapter, CourseDetail, CourseSummary, Lesson, Module } from "./courses";
import { normalizeCourseDetail } from "./courses";

export type TeacherCourse = CourseSummary & {
  status: "draft" | "published" | "archived";
  enrolled_count?: number;
  updated_at?: string;
};

export type Paginated<T> = {
  data: T[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export function listMyTaughtCourses(
  params: { status?: string; page?: number; limit?: number } = {},
) {
  return apiRequest<{ courses: TeacherCourse[]; meta: any }>("/v1/teacher/courses", {
    auth: true,
    query: params,
  }).then((res) => ({
    data: res.courses || [],
    meta: res.meta || {
      page: params.page || 1,
      limit: params.limit || 10,
      total: 0,
      total_pages: 1,
    },
  }));
}

export type CourseInput = {
  title: string;
  slug?: string;
  subtitle?: string;
  description?: string;
  category_id?: string;
  level?: "beginner" | "intermediate" | "advanced";
  language?: string;
  price_cents?: number;
  cover_url?: string;
};

export function createCourse(input: CourseInput) {
  return apiRequest<CourseDetail>("/v1/teacher/courses", {
    method: "POST",
    auth: true,
    body: {
      ...input,
      short_description: input.subtitle,
      price_type: input.price_cents && input.price_cents > 0 ? "paid" : "free",
      price: input.price_cents ? input.price_cents / 100 : 0,
    },
  });
}

export function getTeacherCourse(courseId: string) {
  return apiRequest<CourseDetail & { status: string }>(
    `/v1/teacher/courses/${encodeURIComponent(courseId)}/preview`,
    { auth: true },
  ).then(normalizeCourseDetail);
}

export function updateCourse(courseId: string, input: Partial<CourseInput>) {
  return apiRequest<CourseDetail>(`/v1/teacher/courses/${encodeURIComponent(courseId)}`, {
    method: "PATCH",
    auth: true,
    body: input,
  });
}

export function publishCourse(courseId: string) {
  return apiRequest<{ status: string }>(
    `/v1/teacher/courses/${encodeURIComponent(courseId)}/submit`,
    { method: "POST", auth: true },
  );
}

export function deleteCourse(courseId: string) {
  return apiRequest<{ ok: true }>(`/v1/teacher/courses/${encodeURIComponent(courseId)}`, {
    method: "DELETE",
    auth: true,
  });
}

// Modules
export function createModule(courseId: string, input: { title: string; position?: number }) {
  return apiRequest<Module>(`/v1/teacher/courses/${encodeURIComponent(courseId)}/modules`, {
    method: "POST",
    auth: true,
    body: input,
  });
}

export function updateModule(
  moduleId: string,
  input: Partial<{ title: string; position: number }>,
) {
  return apiRequest<Module>(`/v1/teacher/modules/${encodeURIComponent(moduleId)}`, {
    method: "PATCH",
    auth: true,
    body: input,
  });
}

export function deleteModule(moduleId: string) {
  return apiRequest<{ ok: true }>(`/v1/teacher/modules/${encodeURIComponent(moduleId)}`, {
    method: "DELETE",
    auth: true,
  });
}

// Chapters
export function createChapter(moduleId: string, input: { title: string; position?: number }) {
  return apiRequest<Chapter>(`/v1/teacher/modules/${encodeURIComponent(moduleId)}/chapters`, {
    method: "POST",
    auth: true,
    body: input,
  });
}

export function updateChapter(
  chapterId: string,
  input: Partial<{ title: string; position: number }>,
) {
  return apiRequest<Chapter>(`/v1/teacher/chapters/${encodeURIComponent(chapterId)}`, {
    method: "PATCH",
    auth: true,
    body: input,
  });
}

export function deleteChapter(chapterId: string) {
  return apiRequest<{ ok: true }>(`/v1/teacher/chapters/${encodeURIComponent(chapterId)}`, {
    method: "DELETE",
    auth: true,
  });
}

// Lessons
export type LessonInput = {
  title: string;
  type?: "video" | "text" | "attachment";
  video_id?: string | null;
  duration_seconds?: number;
  is_free_preview?: boolean;
  is_downloadable?: boolean;
  body_html?: string;
  position?: number;
  status?: "draft" | "published";
};

export function createLesson(chapterId: string, input: LessonInput) {
  return apiRequest<Lesson>(`/v1/teacher/chapters/${encodeURIComponent(chapterId)}/lessons`, {
    method: "POST",
    auth: true,
    body: input,
  });
}

export function updateLesson(lessonId: string, input: Partial<LessonInput>) {
  return apiRequest<Lesson>(`/v1/teacher/lessons/${encodeURIComponent(lessonId)}`, {
    method: "PATCH",
    auth: true,
    body: input,
  });
}

export function deleteLesson(lessonId: string) {
  return apiRequest<{ ok: true }>(`/v1/teacher/lessons/${encodeURIComponent(lessonId)}`, {
    method: "DELETE",
    auth: true,
  });
}

export type UploadedVideo = {
  video_id: string;
  status: "processing" | "ready" | "failed";
};

export function uploadLessonVideo(courseId: string, file: File) {
  const formData = new FormData();
  formData.append("course_id", courseId);
  formData.append("file", file);
  return apiRequest<UploadedVideo>("/v1/uploads/video", {
    method: "POST",
    auth: true,
    body: formData,
  });
}

export function getVideoUploadStatus(videoId: string) {
  return apiRequest<UploadedVideo>(`/v1/uploads/video/${encodeURIComponent(videoId)}/status`, {
    auth: true,
  });
}

export type UploadedFile = {
  file_id: string;
  presigned_url: string;
  expires_at: string;
};

export function uploadLessonFile(file: File) {
  const formData = new FormData();
  formData.append("file", file);
  return apiRequest<UploadedFile>("/v1/uploads/file", {
    method: "POST",
    auth: true,
    body: formData,
  });
}

export function reorderContent(input: {
  type: "module" | "chapter" | "lesson";
  parent_id: string;
  positions: Record<string, number>;
}) {
  return apiRequest<{ ok: true }>("/v1/teacher/content/reorder", {
    method: "PATCH",
    auth: true,
    body: input,
  });
}

// Students on a course
export type CourseStudent = {
  id: string;
  user_id: string;
  full_name: string;
  email: string;
  avatar_url?: string | null;
  enrolled_at: string;
  progress_percent: number;
  last_active_at?: string | null;
};

export function listCourseStudents(
  courseId: string,
  params: { page?: number; limit?: number } = {},
) {
  return apiRequest<{
    students?: Array<{
      student_id: string;
      student_name: string;
      student_email?: string;
      overall_progress_percent: number;
      enrolled_at: string;
      last_active_at?: string | null;
    }>;
    data?: Array<{
      student_id: string;
      student_name: string;
      student_email?: string;
      overall_progress_percent: number;
      enrolled_at: string;
      last_active_at?: string | null;
    }>;
    meta?: Paginated<CourseStudent>["meta"];
  }>(`/v1/teacher/courses/${encodeURIComponent(courseId)}/students`, {
    auth: true,
    query: params,
  }).then((result) => {
    const rows = result.students ?? result.data ?? [];
    return {
      data: rows.map((student) => ({
        id: student.student_id,
        user_id: student.student_id,
        full_name: student.student_name,
        email: student.student_email ?? "",
        enrolled_at: student.enrolled_at,
        progress_percent: Math.round(student.overall_progress_percent),
        last_active_at: student.last_active_at ?? null,
      })),
      meta: result.meta ?? {
        page: params.page ?? 1,
        limit: params.limit ?? 20,
        total: rows.length,
        total_pages: 1,
      },
    };
  });
}

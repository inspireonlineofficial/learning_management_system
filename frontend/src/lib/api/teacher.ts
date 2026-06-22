import { apiRequest } from "./client";
import type { Chapter, CourseDetail, CourseNote, CourseSummary, Lesson, Module } from "./courses";
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
  subject?: string;
  level?: "beginner" | "intermediate" | "advanced";
  language?: string;
  price_cents?: number;
  price_type?: "free" | "paid";
  visibility?: "public" | "unlisted" | "private";
  learning_outcomes?: string;
  requirements?: string;
  prerequisites?: string;
  target_audience?: string;
  estimated_duration_minutes?: number;
  cover_url?: string;
};

export function createCourse(input: CourseInput) {
  return apiRequest<CourseDetail>("/v1/teacher/courses", {
    method: "POST",
    auth: true,
    body: {
      ...input,
      short_description: input.subtitle,
      subject: input.subject ?? input.category_id,
      price_type: input.price_cents && input.price_cents > 0 ? "paid" : "free",
      price: input.price_cents ? input.price_cents / 100 : 0,
      thumbnail_url: input.cover_url,
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
    body: {
      ...input,
      short_description: input.subtitle,
      subject: input.subject ?? input.category_id,
      price_type:
        input.price_type ??
        (typeof input.price_cents === "number" && input.price_cents > 0 ? "paid" : undefined),
      price: typeof input.price_cents === "number" ? input.price_cents / 100 : undefined,
      thumbnail_url: input.cover_url,
    },
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
export function createModule(
  courseId: string,
  input: {
    title: string;
    description?: string;
    position?: number;
    is_free?: boolean;
    is_published?: boolean;
  },
) {
  return apiRequest<Module>(`/v1/teacher/courses/${encodeURIComponent(courseId)}/modules`, {
    method: "POST",
    auth: true,
    body: input,
  });
}

export function updateModule(
  moduleId: string,
  input: Partial<{
    title: string;
    description: string;
    position: number;
    is_free: boolean;
    is_published: boolean;
  }>,
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
  description?: string;
  type?: "video" | "text" | "attachment";
  video_id?: string | null;
  duration_seconds?: number;
  is_free_preview?: boolean;
  is_free?: boolean;
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

/**
 * Resumable multipart upload. Survives page refresh: per-part state is
 * persisted to IndexedDB, and the server only marks the video as "ready"
 * after the final CompleteMultipartUpload call lands. Use for any upload
 * over 50 MB; the part count climbs quickly and serial PUT retries are
 * noticeably slower.
 */
export async function uploadLessonVideoMultipart(
  courseId: string,
  file: File,
  onProgress?: (loaded: number, total: number, completedParts: number, totalParts: number) => void,
  signal?: AbortSignal,
  existingVideoId?: string,
): Promise<{ video_id: string }> {
  const { uploadMultipart } = await import("@/lib/multipart-upload");
  return uploadMultipart({
    courseId,
    file,
    signal,
    existingVideoId,
    onProgress: onProgress
      ? ({ loaded, total, completedParts, totalParts }) =>
          onProgress(loaded, total, completedParts, totalParts)
      : undefined,
  });
}

/**
 * Direct-to-RustFS upload. The bytes travel browser -> storage directly so
 * the Go API process never sees the file body. Use this for anything over
 * ~10 MB; below that the legacy `uploadLessonVideo` is fine.
 */
export async function uploadLessonVideoDirect(
  courseId: string,
  file: File,
  onProgress?: (loaded: number, total: number) => void,
  signal?: AbortSignal,
): Promise<{ video_id: string }> {
  const { uploadVideoDirect } = await import("@/lib/video-upload");
  return uploadVideoDirect({
    courseId,
    file,
    signal,
    onProgress: onProgress ? ({ loaded, total }) => onProgress(loaded, total) : undefined,
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

export type CourseNoteInput = {
  module_id?: string;
  lesson_id?: string;
  title: string;
  content: string;
  file_url?: string;
  is_free?: boolean;
  is_published?: boolean;
};

export function createCourseNote(courseId: string, input: CourseNoteInput) {
  return apiRequest<CourseNote>(`/v1/teacher/courses/${encodeURIComponent(courseId)}/notes`, {
    method: "POST",
    auth: true,
    body: input,
  });
}

export function updateCourseNote(noteId: string, input: CourseNoteInput) {
  return apiRequest<CourseNote>(`/v1/teacher/notes/${encodeURIComponent(noteId)}`, {
    method: "PATCH",
    auth: true,
    body: input,
  });
}

export function deleteCourseNote(noteId: string) {
  return apiRequest<{ ok: true }>(`/v1/teacher/notes/${encodeURIComponent(noteId)}`, {
    method: "DELETE",
    auth: true,
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

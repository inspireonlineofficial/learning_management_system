import { apiRequest } from "./client";

export type CourseLevel = "beginner" | "intermediate" | "advanced";

export type CourseSummary = {
  id: string;
  title: string;
  slug?: string;
  subtitle?: string;
  cover_url?: string | null;
  category?: { id: string; name: string } | null;
  level?: CourseLevel;
  price_type?: "free" | "paid";
  price?: number;
  currency?: string;
  visibility?: "public" | "unlisted" | "private";
  learning_outcomes?: string;
  requirements_text?: string;
  target_audience?: string;
  estimated_duration_minutes?: number;
  rating?: number;
  enrollment_count?: number;
  duration_minutes?: number;
  teacher?: { id: string; full_name: string; avatar_url?: string | null } | null;
};

export type Lesson = {
  id: string;
  title: string;
  description?: string;
  chapter_id?: string;
  duration_minutes?: number;
  duration_seconds?: number;
  has_video?: boolean;
  type?: "video" | "text" | "attachment" | string;
  is_preview?: boolean;
  is_free_preview?: boolean;
  is_free?: boolean;
  is_downloadable?: boolean;
  status?: string;
  video_url?: string | null;
  body_html?: string | null;
  resources?: { id: string; title: string; url: string }[];
};

export type CourseNote = {
  id: string;
  course_id: string;
  module_id?: string | null;
  lesson_id?: string | null;
  title: string;
  content: string;
  file_url?: string;
  is_free: boolean;
  is_published: boolean;
  is_locked?: boolean;
  created_at: string;
  updated_at: string;
};

export type CourseComment = {
  id: string;
  course_id: string;
  module_id?: string | null;
  lesson_id?: string | null;
  quiz_id?: string | null;
  user_id: string;
  parent_comment_id?: string | null;
  content: string;
  is_pinned: boolean;
  created_at: string;
  updated_at: string;
};

export type Chapter = {
  id: string;
  module_id?: string;
  title: string;
  position?: number;
  lessons: Lesson[];
};

export type Module = {
  id: string;
  course_id?: string;
  title: string;
  description?: string;
  position?: number;
  is_free?: boolean;
  is_published?: boolean;
  chapters?: Chapter[];
  lessons: Lesson[];
};

export type CourseDetail = CourseSummary & {
  description?: string;
  outcomes?: string[];
  requirements?: string[];
  learning_outcomes?: string;
  prerequisites?: string;
  target_audience?: string;
  estimated_duration_minutes?: number;
  modules?: Module[];
  notes?: CourseNote[];
  comments?: CourseComment[];
  is_enrolled?: boolean;
};

export type Paginated<T> = {
  data: T[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export type CourseReview = {
  id: string;
  course_id: string;
  student_id: string;
  rating: number;
  comment: string;
  created_at: string;
  updated_at: string;
};

export type CourseReviewsResponse = {
  reviews: CourseReview[];
  distribution?: Record<string, number>;
  meta?: Paginated<CourseReview>["meta"];
};

type CoursesListResponse = Paginated<CourseSummary> & {
  courses?: RawCourseSummary[];
};

export type CourseListParams = {
  search?: string;
  category?: string;
  level?: CourseLevel;
  price_type?: "free" | "paid";
  page?: number;
  limit?: number;
  sort?: string;
  order?: "asc" | "desc";
};

export function listCourses(params: CourseListParams = {}) {
  const query = {
    search: params.search,
    subject: params.category,
    level: params.level,
    price_type: params.price_type,
    page: params.page,
    limit: params.limit,
    sort_by: params.sort,
    order: params.order,
  };
  return apiRequest<CoursesListResponse>("/v1/courses", { query }).then((result) => ({
    ...result,
    data: (result.data ?? result.courses ?? []).map(normalizeCourseSummary),
  }));
}

export function getCourse(idOrSlug: string) {
  return apiRequest<CourseDetail>(`/v1/courses/${encodeURIComponent(idOrSlug)}`, {
    auth: true,
  }).then(normalizeCourseDetail);
}

export type Category = { id: string; name: string; slug: string; course_count?: number };

export function listCategories() {
  return listCourses({ limit: 100 }).then((result) => {
    const counts = new Map<string, number>();
    for (const course of result.data) {
      const name = course.category?.name;
      if (name) counts.set(name, (counts.get(name) ?? 0) + 1);
    }
    return {
      data: Array.from(counts.entries()).map(([name, course_count]) => ({
        id: name,
        name,
        slug: name,
        course_count,
      })),
    };
  });
}

export function listCourseReviews(
  courseId: string,
  params: { page?: number; limit?: number } = {},
) {
  return apiRequest<CourseReviewsResponse>(`/v1/courses/${encodeURIComponent(courseId)}/reviews`, {
    query: params,
  });
}

export function upsertCourseReview(courseId: string, input: { rating: number; comment: string }) {
  return apiRequest<CourseReview>(`/v1/courses/${encodeURIComponent(courseId)}/reviews`, {
    method: "POST",
    auth: true,
    body: input,
  });
}

export function deleteMyCourseReview(courseId: string) {
  return apiRequest<void>(`/v1/courses/${encodeURIComponent(courseId)}/reviews/me`, {
    method: "DELETE",
    auth: true,
  });
}

export type CourseCommentsResponse = {
  comments: CourseComment[];
  meta?: Paginated<CourseComment>["meta"];
};

export function listCourseComments(
  courseId: string,
  params: { page?: number; limit?: number } = {},
) {
  return apiRequest<CourseCommentsResponse>(
    `/v1/courses/${encodeURIComponent(courseId)}/comments`,
    { query: params },
  );
}

export function createCourseComment(
  courseId: string,
  input: {
    content: string;
    module_id?: string;
    lesson_id?: string;
    quiz_id?: string;
    parent_comment_id?: string;
  },
) {
  return apiRequest<CourseComment>(`/v1/courses/${encodeURIComponent(courseId)}/comments`, {
    method: "POST",
    auth: true,
    body: input,
  });
}

export function updateCourseComment(
  commentId: string,
  input: { content?: string; is_pinned?: boolean },
) {
  return apiRequest<CourseComment>(`/v1/courses/comments/${encodeURIComponent(commentId)}`, {
    method: "PATCH",
    auth: true,
    body: input,
  });
}

export function deleteCourseComment(commentId: string) {
  return apiRequest<void>(`/v1/courses/comments/${encodeURIComponent(commentId)}`, {
    method: "DELETE",
    auth: true,
  });
}

export type LessonPreview = {
  url: string;
  mime_type?: string;
  duration_seconds?: number;
  captions_url?: string | null;
};

export function getLessonPreview(courseId: string, lessonId: string) {
  void courseId;
  return apiRequest<{ signed_url: string; expires_in?: number }>(
    `/v1/stream/lessons/${encodeURIComponent(lessonId)}/signed-url`,
    { auth: true },
  ).then((result) => ({ url: result.signed_url }));
}

type RawCourseDetail = Omit<CourseDetail, "modules" | "outcomes" | "requirements"> & {
  short_description?: string;
  subject?: string;
  thumbnail_url?: string | null;
  learning_outcomes?: string;
  requirements?: string;
  prerequisites?: string;
  estimated_duration_minutes?: number;
  rating_average?: number;
  total_enrolled?: number;
  modules?: Array<
    Omit<Module, "lessons"> & {
      chapters?: Array<
        Omit<Chapter, "lessons"> & {
          lessons?: Array<
            Omit<Lesson, "duration_minutes" | "is_preview"> & {
              duration_seconds?: number;
              is_free_preview?: boolean;
            }
          >;
        }
      >;
    }
  >;
};

type RawCourseSummary = CourseSummary & {
  short_description?: string;
  subject?: string;
  thumbnail_url?: string | null;
  estimated_duration_minutes?: number;
  rating_average?: number;
  total_enrolled?: number;
};

function normalizeCourseSummary(course: RawCourseSummary): CourseSummary {
  return {
    ...course,
    subtitle: course.subtitle ?? course.short_description,
    cover_url: course.cover_url ?? course.thumbnail_url,
    category:
      course.category ?? (course.subject ? { id: course.subject, name: course.subject } : null),
    rating: course.rating ?? course.rating_average,
    enrollment_count: course.enrollment_count ?? course.total_enrolled,
    duration_minutes: course.duration_minutes ?? course.estimated_duration_minutes,
  };
}

export function normalizeCourseDetail(course: RawCourseDetail): CourseDetail {
  const normalized = normalizeCourseSummary(course);
  return {
    ...course,
    ...normalized,
    outcomes: course.outcomes ?? splitLines(course.learning_outcomes),
    requirements: Array.isArray(course.requirements)
      ? course.requirements
      : splitLines(course.requirements ?? course.prerequisites),
    modules: course.modules?.map((module) => {
      const chapters = module.chapters ?? [];
      const lessons = chapters.flatMap((chapter) =>
        (chapter.lessons ?? []).map((lesson) => ({
          ...lesson,
          chapter_id: lesson.chapter_id ?? chapter.id,
          duration_minutes:
            typeof lesson.duration_minutes === "number"
              ? lesson.duration_minutes
              : typeof lesson.duration_seconds === "number"
                ? Math.max(1, Math.round(lesson.duration_seconds / 60))
                : undefined,
          is_preview: lesson.is_preview ?? lesson.is_free_preview ?? false,
        })),
      );

      return {
        ...module,
        chapters,
        lessons,
      };
    }),
  };
}

function splitLines(value?: string | string[]) {
  if (Array.isArray(value)) return value;
  return (value ?? "")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
}

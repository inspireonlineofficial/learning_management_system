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
  price?: number;
  currency?: string;
  rating?: number;
  enrollment_count?: number;
  duration_minutes?: number;
  teacher?: { id: string; full_name: string; avatar_url?: string | null } | null;
};

export type Lesson = {
  id: string;
  title: string;
  chapter_id?: string;
  duration_minutes?: number;
  duration_seconds?: number;
  type?: "video" | "text" | "attachment" | string;
  is_preview?: boolean;
  is_free_preview?: boolean;
  is_downloadable?: boolean;
  status?: string;
  video_url?: string | null;
  body_html?: string | null;
  resources?: { id: string; title: string; url: string }[];
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
  position?: number;
  chapters?: Chapter[];
  lessons: Lesson[];
};

export type CourseDetail = CourseSummary & {
  description?: string;
  outcomes?: string[];
  requirements?: string[];
  modules?: Module[];
  is_enrolled?: boolean;
};

export type Paginated<T> = {
  data: T[];
  meta: { page: number; limit: number; total: number; total_pages: number };
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

type RawCourseDetail = Omit<CourseDetail, "modules"> & {
  short_description?: string;
  subject?: string;
  thumbnail_url?: string | null;
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
  };
}

export function normalizeCourseDetail(course: RawCourseDetail): CourseDetail {
  const normalized = normalizeCourseSummary(course);
  return {
    ...course,
    ...normalized,
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

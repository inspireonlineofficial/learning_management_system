import { apiRequest } from "./client";

export type CourseTrackItem = {
  title: string;
  detail: string;
  query?: string;
};

export type BookCategoryItem = {
  label: string;
  query?: string;
};

export type HomepageStatItem = {
  label: string;
  value: string;
};

export type HomepageInfoPanel = {
  eyebrow: string;
  title: string;
  body: string;
};

export type HomepageConfig = {
  course_catalog: {
    enabled: boolean;
    eyebrow: string;
    title: string;
    action_label: string;
    tracks: CourseTrackItem[];
  };
  featured_courses: {
    enabled: boolean;
    eyebrow: string;
    title: string;
    action_label: string;
    empty_title: string;
    empty_body: string;
    empty_action_label: string;
    max_items: number;
  };
  bookshop_catalog: {
    enabled: boolean;
    eyebrow: string;
    title: string;
    action_label: string;
    categories: BookCategoryItem[];
  };
  featured_books: {
    enabled: boolean;
    eyebrow: string;
    title: string;
    offer_title: string;
    action_label: string;
    empty_title: string;
    empty_body: string;
    empty_action_label: string;
    max_items: number;
  };
  outcomes: {
    enabled: boolean;
    eyebrow: string;
    title: string;
    body: string;
    stats: HomepageStatItem[];
  };
  info_panels: {
    enabled: boolean;
    items: HomepageInfoPanel[];
  };
  footer: {
    brand_name: string;
    tagline: string;
  };
  bookshop_subjects: string[];
};

export const DEFAULT_HOMEPAGE_CONFIG: HomepageConfig = {
  course_catalog: {
    enabled: true,
    eyebrow: "Course catalog",
    title: "Choose your science track",
    action_label: "View all courses",
    tracks: [
      { title: "Physics", detail: "Mechanics, waves, electricity, and modern physics" },
      { title: "Chemistry", detail: "Physical, organic, inorganic, and lab problem solving" },
      { title: "Biology", detail: "Cell biology, genetics, physiology, and ecology" },
      { title: "Mathematics", detail: "Algebra, calculus, statistics, and exam practice" },
      { title: "ICT", detail: "Programming fundamentals, systems, and digital literacy" },
      { title: "Exam Prep", detail: "Board, admission, and college readiness practice" },
    ],
  },
  featured_courses: {
    enabled: true,
    eyebrow: "Featured courses",
    title: "Start with guided lessons",
    action_label: "Browse catalog",
    empty_title: "Courses are being prepared",
    empty_body:
      "Published science courses will appear here as instructors submit them for approval.",
    empty_action_label: "Open catalog",
    max_items: 6,
  },
  bookshop_catalog: {
    enabled: true,
    eyebrow: "Bookshop catalog",
    title: "Study materials by need",
    action_label: "Visit bookshop",
    categories: [
      { label: "Textbooks" },
      { label: "Workbooks" },
      { label: "Solution guides" },
      { label: "Lab manuals" },
      { label: "Admission prep" },
      { label: "Digital notes" },
    ],
  },
  featured_books: {
    enabled: true,
    eyebrow: "Featured books",
    title: "Recent arrivals",
    offer_title: "Free and offer books",
    action_label: "See all books",
    empty_title: "Books are being prepared",
    empty_body:
      "Featured textbooks, workbooks, and lab manuals will appear here after admin approval.",
    empty_action_label: "Open bookshop",
    max_items: 6,
  },
  outcomes: {
    enabled: true,
    eyebrow: "Outcomes",
    title: "Built for repeatable study habits",
    body: "Inspire combines recorded lessons, live classes, quizzes, assignments, points, certificates, and book access into one practical LMS for science tutoring.",
    stats: [
      { label: "Science tracks", value: "6" },
      { label: "Practice focused", value: "100%" },
      { label: "Live support", value: "Weekly" },
    ],
  },
  info_panels: {
    enabled: true,
    items: [
      {
        eyebrow: "Instructors",
        title: "Teacher-led science paths",
        body: "Courses are organized by subject, level, modules, chapters, and lessons so students always know what to do next.",
      },
      {
        eyebrow: "Student Dashboard",
        title: "Clarity before decoration",
        body: "Dashboards prioritize next lesson, live class, pending assignment, approval status, and progress without distracting clutter.",
      },
      {
        eyebrow: "Enrollment",
        title: "Free and approval-based access",
        body: "Students can browse previews publicly, enroll in free courses instantly, and request admin approval for paid courses or books.",
      },
    ],
  },
  footer: {
    brand_name: "Inspire",
    tagline: "Science tutoring for high school and college learners.",
  },
  bookshop_subjects: ["Physics", "Chemistry", "Biology", "Mathematics", "ICT", "Exam Prep"],
};

function clampItems(value: unknown, fallback: number): number {
  const n = Number(value);
  if (!Number.isFinite(n) || n < 1) return fallback;
  return Math.min(24, Math.floor(n));
}

export function mergeHomepageConfig(raw?: Partial<HomepageConfig> | null): HomepageConfig {
  const r = raw ?? {};
  const d = DEFAULT_HOMEPAGE_CONFIG;

  return {
    course_catalog: {
      enabled: r.course_catalog?.enabled ?? d.course_catalog.enabled,
      eyebrow: r.course_catalog?.eyebrow ?? d.course_catalog.eyebrow,
      title: r.course_catalog?.title ?? d.course_catalog.title,
      action_label: r.course_catalog?.action_label ?? d.course_catalog.action_label,
      tracks: Array.isArray(r.course_catalog?.tracks)
        ? r.course_catalog.tracks
        : d.course_catalog.tracks,
    },
    featured_courses: {
      enabled: r.featured_courses?.enabled ?? d.featured_courses.enabled,
      eyebrow: r.featured_courses?.eyebrow ?? d.featured_courses.eyebrow,
      title: r.featured_courses?.title ?? d.featured_courses.title,
      action_label: r.featured_courses?.action_label ?? d.featured_courses.action_label,
      empty_title: r.featured_courses?.empty_title ?? d.featured_courses.empty_title,
      empty_body: r.featured_courses?.empty_body ?? d.featured_courses.empty_body,
      empty_action_label:
        r.featured_courses?.empty_action_label ?? d.featured_courses.empty_action_label,
      max_items: clampItems(r.featured_courses?.max_items, d.featured_courses.max_items),
    },
    bookshop_catalog: {
      enabled: r.bookshop_catalog?.enabled ?? d.bookshop_catalog.enabled,
      eyebrow: r.bookshop_catalog?.eyebrow ?? d.bookshop_catalog.eyebrow,
      title: r.bookshop_catalog?.title ?? d.bookshop_catalog.title,
      action_label: r.bookshop_catalog?.action_label ?? d.bookshop_catalog.action_label,
      categories: Array.isArray(r.bookshop_catalog?.categories)
        ? r.bookshop_catalog.categories
        : d.bookshop_catalog.categories,
    },
    featured_books: {
      enabled: r.featured_books?.enabled ?? d.featured_books.enabled,
      eyebrow: r.featured_books?.eyebrow ?? d.featured_books.eyebrow,
      title: r.featured_books?.title ?? d.featured_books.title,
      offer_title: r.featured_books?.offer_title ?? d.featured_books.offer_title,
      action_label: r.featured_books?.action_label ?? d.featured_books.action_label,
      empty_title: r.featured_books?.empty_title ?? d.featured_books.empty_title,
      empty_body: r.featured_books?.empty_body ?? d.featured_books.empty_body,
      empty_action_label:
        r.featured_books?.empty_action_label ?? d.featured_books.empty_action_label,
      max_items: clampItems(r.featured_books?.max_items, d.featured_books.max_items),
    },
    outcomes: {
      enabled: r.outcomes?.enabled ?? d.outcomes.enabled,
      eyebrow: r.outcomes?.eyebrow ?? d.outcomes.eyebrow,
      title: r.outcomes?.title ?? d.outcomes.title,
      body: r.outcomes?.body ?? d.outcomes.body,
      stats: Array.isArray(r.outcomes?.stats) ? r.outcomes.stats : d.outcomes.stats,
    },
    info_panels: {
      enabled: r.info_panels?.enabled ?? d.info_panels.enabled,
      items: Array.isArray(r.info_panels?.items) ? r.info_panels.items : d.info_panels.items,
    },
    footer: {
      brand_name: r.footer?.brand_name ?? d.footer.brand_name,
      tagline: r.footer?.tagline ?? d.footer.tagline,
    },
    bookshop_subjects: Array.isArray(r.bookshop_subjects)
      ? r.bookshop_subjects
      : d.bookshop_subjects,
  };
}

export type PublicHomepageResponse = {
  platform_name: string;
  homepage_config: HomepageConfig;
  updated_at?: string;
};

export const getPublicHomepage = () =>
  apiRequest<{
    platform_name?: string;
    homepage_config?: Partial<HomepageConfig> | null;
    updated_at?: string;
  }>("/v1/public/homepage").then((res) => ({
    platform_name: res.platform_name ?? DEFAULT_HOMEPAGE_CONFIG.footer.brand_name,
    homepage_config: mergeHomepageConfig(res.homepage_config),
    updated_at: res.updated_at,
  }));

export const getAdminHomepageConfig = () =>
  apiRequest<{
    homepage_config?: Partial<HomepageConfig> | null;
  }>("/v1/admin/system/settings", { auth: true }).then((res) =>
    mergeHomepageConfig(res.homepage_config),
  );

export const updateAdminHomepageConfig = (config: HomepageConfig) =>
  apiRequest<{
    homepage_config?: Partial<HomepageConfig> | null;
  }>("/v1/admin/system/settings", {
    method: "PATCH",
    auth: true,
    body: {
      homepage_config: config,
    },
  }).then((res) => mergeHomepageConfig(res.homepage_config));

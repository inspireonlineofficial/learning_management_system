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

export type SiteLink = {
  label: string;
  href: string;
};

export type FooterColumn = {
  title: string;
  links: SiteLink[];
};

export type FaqItem = {
  question: string;
  answer: string;
};

export type PageHeroCopy = {
  eyebrow: string;
  title: string;
  body: string;
};

export type AuthPageCopy = {
  eyebrow: string;
  heading: string;
  subheading: string;
};

export type SiteConfig = {
  logo_url: string;
  favicon_url: string;
  seo_title: string;
  seo_description: string;
  nav_links: SiteLink[];
  about_text: string;
  show_tracks_column: boolean;
  tracks_column_title: string;
  footer_columns: FooterColumn[];
  contact_title: string;
  support_email: string;
  phone: string;
  phone_label: string;
  address: string;
  social_links: SiteLink[];
  legal_links: SiteLink[];
  bottom_note: string;
};

export type HomepageConfig = {
  site: SiteConfig;
  faq: {
    enabled: boolean;
    eyebrow: string;
    title: string;
    items: FaqItem[];
  };
  pages: {
    courses: PageHeroCopy;
    bookshop: PageHeroCopy;
    forum: PageHeroCopy;
    login: AuthPageCopy;
    register: AuthPageCopy;
  };
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
  site: {
    logo_url: "",
    favicon_url: "",
    seo_title: "Inspire — Science tutoring for high school and college",
    seo_description:
      "Inspire brings science courses, live classes, assessments, certificates, and curated study books into one tutoring platform.",
    nav_links: [
      { label: "Courses", href: "/courses" },
      { label: "Bookshop", href: "/bookshop" },
      { label: "Outcomes", href: "/#outcomes" },
      { label: "FAQ", href: "/#faq" },
      { label: "Forum", href: "/forum" },
    ],
    about_text:
      "Practical, teacher-led science tutoring and curated study materials designed for high school, college, and admission learners.",
    show_tracks_column: true,
    tracks_column_title: "Science Tracks",
    footer_columns: [
      {
        title: "Bookshop & Community",
        links: [
          { label: "Textbooks & Workbooks", href: "/bookshop" },
          { label: "Student Science Forum", href: "/forum" },
          { label: "Learning Outcomes", href: "/#outcomes" },
          { label: "FAQ", href: "/#faq" },
        ],
      },
    ],
    contact_title: "Support & Contact",
    support_email: "support@inspireonlinebd.com",
    phone: "",
    phone_label: "Academic Helpdesk",
    address: "",
    social_links: [],
    legal_links: [],
    bottom_note: "",
  },
  faq: {
    enabled: true,
    eyebrow: "Frequently Asked Questions",
    title: "Answers for students and parents",
    items: [
      {
        question: "How do free previews and approval-based paid courses work?",
        answer:
          "Every course track offers public syllabus outlines and free preview lessons so you can evaluate teaching clarity first. Free courses unlock immediately after registration, while paid cohorts use a simple admin approval workflow once enrollment details are verified.",
      },
      {
        question: "When are weekly live support classes held?",
        answer:
          "Live support sessions run every week in evening slots (BST) so they never clash with school or college hours. If you miss a live Q&A, the annotated recording and problem sheet appear directly inside your student dashboard.",
      },
      {
        question: "How do bookshop orders and digital study materials work?",
        answer:
          "Digital notes and formula sheets become accessible inside your LMS library immediately upon approval. Printed textbooks, workbooks, and lab manuals can be requested through the Bookshop with nationwide courier delivery across Bangladesh.",
      },
      {
        question: "Which science levels and curricula are supported?",
        answer:
          "Inspire Online currently supports SSC (Class 9–10), HSC (Class 11–12), and University Admission Preparation tracks across Physics, Chemistry, Biology, Higher Mathematics, and HSC ICT.",
      },
    ],
  },
  pages: {
    courses: {
      eyebrow: "Course Catalog",
      title: "A curated science catalog.",
      body: "Browse structured courses by subject, level, and instructor. Sign in to enroll and track your weekly study progress.",
    },
    bookshop: {
      eyebrow: "Inspire Bookshop",
      title: "Expand your learning with curated reading materials.",
      body: "Access textbooks, workbooks, solution guides, and lab manuals aligned with your academic syllabus.",
    },
    forum: {
      eyebrow: "Community Forum",
      title: "Read science questions and answers from the Inspire community.",
      body: "",
    },
    login: {
      eyebrow: "Sign in",
      heading: "Welcome back",
      subheading: "Enter your credentials to access your learning dashboard.",
    },
    register: {
      eyebrow: "Enroll",
      heading: "Begin your study",
      subheading: "Create an account to browse courses, join live classes, and track progress.",
    },
  },
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

function str(value: unknown, fallback: string): string {
  return typeof value === "string" ? value : fallback;
}

function links(value: unknown, fallback: SiteLink[]): SiteLink[] {
  if (!Array.isArray(value)) return fallback;
  return value
    .filter((l): l is SiteLink => !!l && typeof l === "object")
    .map((l) => ({ label: str(l.label, ""), href: str(l.href, "") }));
}

function hero(value: Partial<PageHeroCopy> | undefined, d: PageHeroCopy): PageHeroCopy {
  return {
    eyebrow: str(value?.eyebrow, d.eyebrow),
    title: str(value?.title, d.title),
    body: str(value?.body, d.body),
  };
}

function authCopy(value: Partial<AuthPageCopy> | undefined, d: AuthPageCopy): AuthPageCopy {
  return {
    eyebrow: str(value?.eyebrow, d.eyebrow),
    heading: str(value?.heading, d.heading),
    subheading: str(value?.subheading, d.subheading),
  };
}

function mergeSite(r: Partial<SiteConfig> | undefined, d: SiteConfig): SiteConfig {
  return {
    logo_url: str(r?.logo_url, d.logo_url),
    favicon_url: str(r?.favicon_url, d.favicon_url),
    seo_title: str(r?.seo_title, d.seo_title),
    seo_description: str(r?.seo_description, d.seo_description),
    nav_links: links(r?.nav_links, d.nav_links),
    about_text: str(r?.about_text, d.about_text),
    show_tracks_column:
      typeof r?.show_tracks_column === "boolean" ? r.show_tracks_column : d.show_tracks_column,
    tracks_column_title: str(r?.tracks_column_title, d.tracks_column_title),
    footer_columns: Array.isArray(r?.footer_columns)
      ? r.footer_columns.map((c) => ({ title: str(c?.title, ""), links: links(c?.links, []) }))
      : d.footer_columns,
    contact_title: str(r?.contact_title, d.contact_title),
    support_email: str(r?.support_email, d.support_email),
    phone: str(r?.phone, d.phone),
    phone_label: str(r?.phone_label, d.phone_label),
    address: str(r?.address, d.address),
    social_links: links(r?.social_links, d.social_links),
    legal_links: links(r?.legal_links, d.legal_links),
    bottom_note: str(r?.bottom_note, d.bottom_note),
  };
}

/**
 * Only same-site paths, anchors, mailto/tel and http(s) URLs are rendered as
 * links. Anything else (e.g. "javascript:") is dropped so an admin typo or a
 * compromised admin account cannot inject script through a nav link.
 */
export function safeHref(href: string): string | null {
  const h = href.trim();
  if (!h) return null;
  if (h.startsWith("/") && !h.startsWith("//")) return h;
  if (h.startsWith("#")) return h;
  if (/^(https?:|mailto:|tel:)/i.test(h)) return h;
  return null;
}

export function isExternalHref(href: string): boolean {
  return /^https?:/i.test(href.trim());
}

export function mergeHomepageConfig(raw?: Partial<HomepageConfig> | null): HomepageConfig {
  const r = raw ?? {};
  const d = DEFAULT_HOMEPAGE_CONFIG;

  return {
    site: mergeSite(r.site, d.site),
    faq: {
      enabled: r.faq?.enabled ?? d.faq.enabled,
      eyebrow: str(r.faq?.eyebrow, d.faq.eyebrow),
      title: str(r.faq?.title, d.faq.title),
      items: Array.isArray(r.faq?.items)
        ? r.faq.items.map((i) => ({ question: str(i?.question, ""), answer: str(i?.answer, "") }))
        : d.faq.items,
    },
    pages: {
      courses: hero(r.pages?.courses, d.pages.courses),
      bookshop: hero(r.pages?.bookshop, d.pages.bookshop),
      forum: hero(r.pages?.forum, d.pages.forum),
      login: authCopy(r.pages?.login, d.pages.login),
      register: authCopy(r.pages?.register, d.pages.register),
    },
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

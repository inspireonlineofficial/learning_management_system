import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
  Atom,
  BookOpen,
  Check,
  CheckCircle2,
  ChevronDown,
  ClipboardCheck,
  Dna,
  FileEdit,
  FileText,
  FlaskConical,
  GraduationCap,
  Info,
  LineChart,
  Monitor,
  Search,
  TestTube2,
  X,
} from "lucide-react";
import { toast } from "sonner";

import libraryHero from "@/assets/library-hero.jpg";
import { BookCard } from "@/components/bookshop/book-card";
import { CourseCard, CourseCardSkeleton } from "@/components/course/course-card";
import { PublicFooter } from "@/components/layout/public-footer";
import { PublicHeader } from "@/components/layout/public-header";
import { SlideCarousel } from "@/components/marketing/ad-carousel";
import { listBooks } from "@/lib/api/bookshop";
import { listCourses } from "@/lib/api/courses";
import { DEFAULT_HOMEPAGE_CONFIG, getPublicHomepage } from "@/lib/api/homepage";
import { listSlides, type Slide } from "@/lib/api/slides";

type LevelFilter = "all" | "SSC" | "HSC" | "Admission";

type EnrichedTrack = {
  id: string;
  title: string;
  detail: string;
  query: string;
  levels: Array<"SSC" | "HSC" | "Admission">;
  countLabel: string;
  modules: string[];
};

const TRACK_METADATA: Record<
  string,
  {
    levels: Array<"SSC" | "HSC" | "Admission">;
    countLabel: string;
    modules: string[];
  }
> = {
  physics: {
    levels: ["SSC", "HSC", "Admission"],
    countLabel: "14 Modules",
    modules: [
      "Module 1: Vectors, Kinematics & Graphical Analysis (Free Preview)",
      "Module 2: Newtonian Mechanics & Conservation Laws",
      "Module 3: Waves, Optics & Electromagnetism Lab Problems",
      "Module 4: Modern Physics & Board Question Solving",
    ],
  },
  chemistry: {
    levels: ["SSC", "HSC", "Admission"],
    countLabel: "12 Modules",
    modules: [
      "Module 1: Atomic Structure & Periodic Trends (Free Preview)",
      "Module 2: Chemical Bonding, Kinetics & Equilibrium",
      "Module 3: Organic Reaction Mechanisms & Synthesis",
      "Module 4: Quantitative Analysis & Volumetric Titration",
    ],
  },
  biology: {
    levels: ["SSC", "HSC", "Admission"],
    countLabel: "10 Modules",
    modules: [
      "Module 1: Cell Structure, Microscopy & Division (Free Preview)",
      "Module 2: Plant & Human Physiology Systems",
      "Module 3: Mendelian Genetics & Molecular Biology",
      "Module 4: Ecology, Taxonomy & Medical MCQ Drills",
    ],
  },
  mathematics: {
    levels: ["SSC", "HSC", "Admission"],
    countLabel: "16 Modules",
    modules: [
      "Module 1: Matrices, Determinants & Linear Systems (Free Preview)",
      "Module 2: Differential & Integral Calculus Mastery",
      "Module 3: Coordinate Geometry & Conic Sections",
      "Module 4: Probability, Statistics & Timed Problem Sets",
    ],
  },
  ict: {
    levels: ["HSC"],
    countLabel: "8 Modules",
    modules: [
      "Module 1: Number Systems & Digital Logic Gates (Free Preview)",
      "Module 2: Web Design Fundamentals (HTML & Semantic Structure)",
      "Module 3: C Programming, Flowcharts & Algorithms",
      "Module 4: Relational Databases & Board Practicals",
    ],
  },
  "exam prep": {
    levels: ["SSC", "HSC", "Admission"],
    countLabel: "20+ Mock Tests",
    modules: [
      "Module 1: Diagnostic Science Readiness Test (Free Preview)",
      "Module 2: Chapter-Wise Creative Question (CQ) Workshops",
      "Module 3: Negative-Marking Admission MCQ Simulations",
      "Module 4: Full-Length Model Tests with Teacher Feedback",
    ],
  },
};

const BOOKSHOP_CATEGORY_DESCRIPTIONS: Record<string, string> = {
  textbooks: "Core syllabus & board editions",
  workbooks: "Structured chapter problem sets",
  "solution guides": "Step-by-step worked answers",
  "lab manuals": "Practical experiments & viva prep",
  "admission prep": "Engineering, medical & varsity sets",
  "digital notes": "Concise formula sheets & PDFs",
};

const OUTCOME_STAT_SUBTEXT: Record<string, string> = {
  "science tracks": "Core high school & college disciplines",
  "practice focused": "Quizzes & assignments in every module",
  "live support": "Scheduled teacher Q&A & doubt clearing",
};

const GUIDED_LESSON_STEPS = [
  {
    id: "step-video",
    title: "1. Recorded Concept Video (18 min)",
    detail: "Instructor-led breakdown with board-style derivations and chapter notes.",
    defaultChecked: true,
  },
  {
    id: "step-quiz",
    title: "2. Practice Quiz & Instant Score",
    detail: "10 timed MCQs with worked explanations and mastery points.",
    defaultChecked: false,
  },
  {
    id: "step-live",
    title: "3. Weekly Live Q&A Session",
    detail: "Submit tricky textbook problems ahead of time for live teacher walkthroughs.",
    defaultChecked: false,
  },
  {
    id: "step-cert",
    title: "4. Verified Certificate",
    detail: "Complete module assignments and quizzes to earn your track completion record.",
    defaultChecked: false,
  },
];

const FAQ_ITEMS = [
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
];

const fallbackSlides: Slide[] = [
  {
    id: "fallback-science",
    title: "Structured science tutoring paired with curated study materials.",
    subtitle:
      "Combine recorded lessons, weekly live classes, quizzes, assignments, and textbooks into one focused learning platform built for consistent exam readiness.",
    link_url: "#courses",
    media_url: libraryHero,
    media_type: "image",
    duration_ms: 5000,
    position: 1,
    is_active: true,
  },
  {
    id: "fallback-bookshop",
    title: "Pair every lesson with curated study materials.",
    subtitle: "Find textbooks, workbooks, solution guides, lab manuals, and digital notes.",
    link_url: "/bookshop",
    media_url: libraryHero,
    media_type: "image",
    duration_ms: 5000,
    position: 2,
    is_active: true,
  },
  {
    id: "fallback-live",
    title: "Build repeatable study habits with teacher-led support.",
    subtitle: "Use dashboards, quizzes, assignments, points, and live classes to stay on track.",
    link_url: "/register",
    media_url: libraryHero,
    media_type: "image",
    duration_ms: 5000,
    position: 3,
    is_active: true,
  },
];

function isValidEmail(email: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function TrackIcon({ title }: { title: string }) {
  const key = title.trim().toLowerCase();
  if (key.includes("physics")) return <Atom className="w-6 h-6" aria-hidden="true" />;
  if (key.includes("chemistry")) return <FlaskConical className="w-6 h-6" aria-hidden="true" />;
  if (key.includes("biology")) return <Dna className="w-6 h-6" aria-hidden="true" />;
  if (key.includes("math")) return <LineChart className="w-6 h-6" aria-hidden="true" />;
  if (key.includes("ict")) return <Monitor className="w-6 h-6" aria-hidden="true" />;
  return <ClipboardCheck className="w-6 h-6" aria-hidden="true" />;
}

function BookCategoryIcon({ label }: { label: string }) {
  const key = label.trim().toLowerCase();
  if (key.includes("textbook")) return <BookOpen className="w-5 h-5" aria-hidden="true" />;
  if (key.includes("workbook")) return <FileEdit className="w-5 h-5" aria-hidden="true" />;
  if (key.includes("solution")) return <CheckCircle2 className="w-5 h-5" aria-hidden="true" />;
  if (key.includes("lab")) return <TestTube2 className="w-5 h-5" aria-hidden="true" />;
  if (key.includes("admission")) return <GraduationCap className="w-5 h-5" aria-hidden="true" />;
  return <FileText className="w-5 h-5" aria-hidden="true" />;
}

export const Route = createFileRoute("/")({
  head: () => ({
    meta: [
      { title: "Inspire Online — Science Tutoring LMS for High School & College Learners" },
      {
        name: "description",
        content:
          "Structured science tutoring for SSC, HSC, and admission prep. Combine recorded lessons, weekly live classes, quizzes, assignments, and curated study materials.",
      },
      { property: "og:title", content: "Inspire Online" },
      {
        property: "og:description",
        content: "Science tutoring, course learning, live classes, and curated study books.",
      },
    ],
  }),
  component: Landing,
});

function Landing() {
  const { data: homepageData } = useQuery({
    queryKey: ["homepage", "public"],
    queryFn: getPublicHomepage,
    retry: false,
    staleTime: 5 * 60_000,
  });

  const cfg = homepageData?.homepage_config ?? DEFAULT_HOMEPAGE_CONFIG;

  const { data: slidesData } = useQuery({
    queryKey: ["slides", "public"],
    queryFn: listSlides,
    retry: false,
    staleTime: 5 * 60_000,
  });

  const { data: coursesData, isLoading: coursesLoading } = useQuery({
    queryKey: ["courses", { recent: true, limit: cfg.featured_courses.max_items }],
    queryFn: () =>
      listCourses({ sort: "published_at", order: "desc", limit: cfg.featured_courses.max_items }),
    retry: false,
    staleTime: 60_000,
  });

  const { data: booksData, isLoading: booksLoading } = useQuery({
    queryKey: ["books", { offers: true, limit: Math.max(12, cfg.featured_books.max_items) }],
    queryFn: () => listBooks({ sort: "newest", limit: Math.max(12, cfg.featured_books.max_items) }),
    retry: false,
    staleTime: 60_000,
  });

  const slides = slidesData?.slides?.length ? slidesData.slides : fallbackSlides;
  const featuredCourses = coursesData?.data ?? [];
  const offerBooks =
    booksData?.data?.filter((book) => book.is_free === true || book.price_cents === 0) ?? [];
  const featuredBooks =
    offerBooks.length > 0
      ? offerBooks.slice(0, cfg.featured_books.max_items)
      : (booksData?.data?.slice(0, cfg.featured_books.max_items) ?? []);

  // Quick-find filter state
  const [searchQuery, setSearchQuery] = useState("");
  const [activeLevel, setActiveLevel] = useState<LevelFilter>("all");
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Track preview modal state
  const [selectedTrack, setSelectedTrack] = useState<EnrichedTrack | null>(null);
  const [modalFeedback, setModalFeedback] = useState<string | null>(null);
  const closeTrackModalRef = useRef<HTMLButtonElement>(null);

  // Guided lesson interactive checklist state
  const [checkedSteps, setCheckedSteps] = useState<Record<string, boolean>>(() =>
    Object.fromEntries(GUIDED_LESSON_STEPS.map((s) => [s.id, s.defaultChecked])),
  );

  // Early access form state
  const [eaTrack, setEaTrack] = useState("Physics");
  const [eaEmail, setEaEmail] = useState("");
  const [eaError, setEaError] = useState("");
  const [eaFeedback, setEaFeedback] = useState("");

  // Bookshop request form state
  const [bookPanelOpen, setBookPanelOpen] = useState(false);
  const [brCategory, setBrCategory] = useState(
    cfg.bookshop_catalog.categories[0]?.label ?? "Textbooks",
  );
  const [brEmail, setBrEmail] = useState("");
  const [brFeedback, setBrFeedback] = useState<{ type: "error" | "success"; text: string } | null>(
    null,
  );
  const bookRequestPanelRef = useRef<HTMLDivElement>(null);
  const brEmailInputRef = useRef<HTMLInputElement>(null);

  // Enrich configured tracks with level tags, module counts, and sample outlines
  const enrichedTracks = useMemo<EnrichedTrack[]>(() => {
    return cfg.course_catalog.tracks.map((track, idx) => {
      const meta = TRACK_METADATA[track.title.trim().toLowerCase()] ?? {
        levels: ["SSC", "HSC", "Admission"] as Array<"SSC" | "HSC" | "Admission">,
        countLabel: "10+ Modules",
        modules: [
          `Module 1: Core Foundations of ${track.title} (Free Preview)`,
          `Module 2: Guided Practice & Concept Quizzes`,
          `Module 3: Weekly Live Q&A & Problem Walkthroughs`,
          `Module 4: Full-Length Exam Readiness Assessment`,
        ],
      };
      return {
        id: `${track.title.toLowerCase().replace(/\s+/g, "-")}-${idx}`,
        title: track.title,
        detail: track.detail,
        query: track.query?.trim() || track.title,
        levels: meta.levels,
        countLabel: meta.countLabel,
        modules: meta.modules,
      };
    });
  }, [cfg.course_catalog.tracks]);

  const filteredTracks = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return enrichedTracks.filter((track) => {
      const matchesLevel = activeLevel === "all" || track.levels.includes(activeLevel);
      const searchable =
        `${track.title} ${track.detail} ${track.levels.join(" ")} ${track.modules.join(" ")}`.toLowerCase();
      const matchesQuery = !q || searchable.includes(q);
      return matchesLevel && matchesQuery;
    });
  }, [enrichedTracks, searchQuery, activeLevel]);

  // Modal keyboard & scroll-lock handling
  useEffect(() => {
    if (!selectedTrack) return;
    closeTrackModalRef.current?.focus();
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setSelectedTrack(null);
        setModalFeedback(null);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => {
      document.body.style.overflow = prevOverflow;
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [selectedTrack]);

  const completedStepCount = Object.values(checkedSteps).filter(Boolean).length;
  const checklistPercent = Math.round((completedStepCount / GUIDED_LESSON_STEPS.length) * 100);

  const resetTrackFilters = () => {
    setSearchQuery("");
    setActiveLevel("all");
  };

  const handleEarlyAccessSubmit = (e: FormEvent) => {
    e.preventDefault();
    const trimmed = eaEmail.trim();
    if (!isValidEmail(trimmed)) {
      setEaError("Please enter a valid email address (for example, name@example.com).");
      setEaFeedback("");
      return;
    }
    setEaError("");
    setEaFeedback(
      `You are on the early-access list for ${eaTrack}. We will notify ${trimmed} as soon as approved cohorts open.`,
    );
    setEaEmail("");
    toast.success(`Early-access alert saved for ${eaTrack}`);
  };

  const openBookRequestForCategory = (categoryLabel?: string) => {
    if (categoryLabel) setBrCategory(categoryLabel);
    setBookPanelOpen(true);
    setBrFeedback(null);
    window.setTimeout(() => {
      bookRequestPanelRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
      brEmailInputRef.current?.focus();
    }, 60);
  };

  const handleBookRequestSubmit = (e: FormEvent) => {
    e.preventDefault();
    const trimmed = brEmail.trim();
    if (!isValidEmail(trimmed)) {
      setBrFeedback({
        type: "error",
        text: "Please provide a valid email address so we can send your book list.",
      });
      brEmailInputRef.current?.focus();
      return;
    }
    setBrFeedback({
      type: "success",
      text: `Request received for ${brCategory}. Our academic desk will email the curated list to ${trimmed}.`,
    });
    setBrEmail("");
    toast.success(`Book list request received for ${brCategory}`);
  };

  return (
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy-light font-sans antialiased flex flex-col">
      {/* 1. Skip to Main Content Link (WCAG 2.1 AA Bypass Blocks) */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[100] focus:inline-flex focus:items-center focus:justify-center focus:min-h-[44px] focus:px-5 focus:py-2.5 focus:rounded-lg focus:bg-navy focus:text-white focus:font-semibold focus:shadow-lg focus:outline-none focus:ring-2 focus:ring-gold focus:ring-offset-2"
      >
        Skip to main content
      </a>

      {/* STICKY ACCESSIBLE HEADER (with existing Inspire Logo preserved) */}
      <PublicHeader />

      {/* MAIN CONTENT */}
      <main id="main-content" tabIndex={-1} className="flex-1 focus:outline-none">
        {/* 2. HERO WITH CURRENT SLIDING IMAGES + ANCHORED QUICK-FIND BAR */}
        <div className="hero-atmosphere">
          <SlideCarousel slides={slides} />

          {/* Interactive Quick-Find Bar anchored at the bottom of the Hero */}
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
            <div className="bg-white rounded-xl p-4 sm:p-5 shadow-xl border border-borderui text-navy-light">
              <div className="flex flex-col lg:flex-row lg:items-center gap-4">
                {/* Live Search Input */}
                <div className="flex-1">
                  <label htmlFor="track-search" className="sr-only">
                    Search science subjects or topics
                  </label>
                  <div className="relative">
                    <Search
                      className="w-5 h-5 text-slatebody absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
                      aria-hidden="true"
                    />
                    <input
                      ref={searchInputRef}
                      type="search"
                      id="track-search"
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      placeholder="Search subjects or topics (e.g., Physics, Organic, Calculus, ICT)..."
                      className="w-full min-h-[46px] pl-11 pr-4 py-2.5 rounded-lg border border-borderui bg-canvas text-navy text-sm sm:text-base placeholder:text-slatebody focus:bg-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-navy"
                      autoComplete="off"
                    />
                  </div>
                </div>

                {/* Level Filter Pills */}
                <div
                  className="flex flex-wrap items-center gap-2"
                  role="group"
                  aria-label="Filter science tracks by academic level"
                >
                  {(
                    [
                      { id: "all", label: "All Levels" },
                      { id: "SSC", label: "SSC" },
                      { id: "HSC", label: "HSC" },
                      { id: "Admission", label: "Admission Prep" },
                    ] as const
                  ).map((pill) => {
                    const isSelected = activeLevel === pill.id;
                    return (
                      <button
                        key={pill.id}
                        type="button"
                        onClick={() => setActiveLevel(pill.id)}
                        aria-pressed={isSelected}
                        className={`inline-flex items-center justify-center min-h-[44px] px-4 rounded-lg text-sm font-semibold transition-colors border focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy ${
                          isSelected
                            ? "bg-navy text-white border-navy"
                            : "bg-white text-navy-light border-borderui hover:border-navy"
                        }`}
                      >
                        {pill.label}
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* 3. COURSE CATALOG ("Choose your science track") */}
        {cfg.course_catalog.enabled && (
          <section
            id="courses"
            className="py-16 sm:py-20 lg:py-24 bg-canvas"
            aria-labelledby="courses-heading"
          >
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
              <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4 mb-10">
                <div>
                  <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-2">
                    {cfg.course_catalog.eyebrow}
                  </p>
                  <h2
                    id="courses-heading"
                    className="font-serif text-3xl sm:text-4xl font-bold text-navy tracking-tight"
                  >
                    {cfg.course_catalog.title}
                  </h2>
                </div>
                <Link
                  to="/courses"
                  className="self-start sm:self-auto inline-flex items-center gap-1.5 min-h-[44px] px-2 text-sm font-semibold text-navy underline underline-offset-4 decoration-gold decoration-2 hover:text-bronze rounded-lg transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                >
                  {cfg.course_catalog.action_label}
                  <span aria-hidden="true">&rarr;</span>
                </Link>
              </div>

              {/* Live Filter Status Region */}
              <div aria-live="polite" className="sr-only">
                {filteredTracks.length === 0
                  ? "No science tracks match your current search or level filter."
                  : `Showing ${filteredTracks.length} of ${enrichedTracks.length} science tracks.`}
              </div>

              {filteredTracks.length > 0 ? (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                  {filteredTracks.map((track) => (
                    <button
                      key={track.id}
                      type="button"
                      onClick={() => {
                        setSelectedTrack(track);
                        setModalFeedback(null);
                      }}
                      aria-label={`${track.title} track, ${track.countLabel}. Click to preview syllabus.`}
                      className="group text-left bg-white rounded-xl p-6 border border-borderui hover:border-navy hover:-translate-y-0.5 hover:shadow-md transition-all flex flex-col justify-between min-h-[210px] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                    >
                      <div>
                        <div className="flex items-center justify-between gap-3 mb-4">
                          <span className="w-11 h-11 rounded-lg bg-stoneband text-navy flex items-center justify-center group-hover:bg-navy group-hover:text-white transition-colors">
                            <TrackIcon title={track.title} />
                          </span>
                          <span className="text-xs font-bold tracking-wider uppercase text-bronze">
                            {track.countLabel}
                          </span>
                        </div>
                        <h3 className="font-serif text-2xl font-bold text-navy group-hover:text-bronze transition-colors">
                          {track.title}
                        </h3>
                        <p className="mt-2 text-sm sm:text-base text-slatebody leading-relaxed">
                          {track.detail}
                        </p>
                      </div>
                      <div className="mt-6 pt-4 border-t border-borderui/70 flex items-center justify-between gap-2">
                        <div className="flex flex-wrap gap-1.5">
                          {track.levels.map((lvl) => (
                            <span
                              key={lvl}
                              className="inline-flex items-center px-2.5 py-0.5 rounded-md bg-stoneband text-navy text-xs font-semibold"
                            >
                              {lvl}
                            </span>
                          ))}
                        </div>
                        <span className="inline-flex items-center gap-1 text-sm font-semibold text-navy shrink-0">
                          Preview{" "}
                          <span
                            className="transition-transform group-hover:translate-x-1"
                            aria-hidden="true"
                          >
                            &rarr;
                          </span>
                        </span>
                      </div>
                    </button>
                  ))}
                </div>
              ) : (
                <div className="mt-4 bg-white border border-borderui rounded-xl p-8 text-center">
                  <p className="font-serif text-xl font-bold text-navy">
                    No matching science tracks found
                  </p>
                  <p className="mt-2 text-slatebody text-sm sm:text-base max-w-md mx-auto">
                    Try searching for another subject keyword such as Physics, Organic, Calculus, or
                    ICT, or reset your active filters.
                  </p>
                  <button
                    type="button"
                    onClick={() => {
                      resetTrackFilters();
                      searchInputRef.current?.focus();
                    }}
                    className="mt-5 inline-flex items-center justify-center min-h-[44px] px-5 py-2.5 rounded-lg bg-navy text-white text-sm font-semibold hover:bg-navy-light transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                  >
                    Reset filters
                  </button>
                </div>
              )}
            </div>
          </section>
        )}

        {/* 4. GUIDED LESSONS PREVIEW, FEATURED COURSES & EARLY ACCESS */}
        {cfg.featured_courses.enabled && (
          <section className="pb-16 sm:pb-20 lg:pb-24 bg-canvas" aria-labelledby="guided-heading">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-10">
              {/* Live published courses when available */}
              {(coursesLoading || featuredCourses.length > 0) && (
                <div>
                  <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                    <div>
                      <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-2">
                        {cfg.featured_courses.eyebrow}
                      </p>
                      <h2 className="font-serif text-3xl sm:text-4xl font-bold text-navy tracking-tight">
                        {cfg.featured_courses.title}
                      </h2>
                    </div>
                    <Link
                      to="/courses"
                      className="self-start sm:self-auto inline-flex items-center gap-1.5 min-h-[44px] px-2 text-sm font-semibold text-navy underline underline-offset-4 decoration-gold decoration-2 hover:text-bronze rounded-lg transition-colors"
                    >
                      {cfg.featured_courses.action_label}
                      <span aria-hidden="true">&rarr;</span>
                    </Link>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                    {coursesLoading
                      ? Array.from({ length: 3 }).map((_, index) => (
                          <CourseCardSkeleton key={index} />
                        ))
                      : featuredCourses.map((course) => (
                          <CourseCard key={course.id} course={course} />
                        ))}
                  </div>
                </div>
              )}

              {/* Interactive Guided Learning Flow + Early Access Card */}
              <div className="bg-white rounded-2xl border border-borderui shadow-sm overflow-hidden grid grid-cols-1 lg:grid-cols-12">
                {/* Left Column: Interactive Sample Lesson Checklist */}
                <div className="lg:col-span-7 p-6 sm:p-10 lg:p-12 border-b lg:border-b-0 lg:border-r border-borderui">
                  <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-2">
                    {featuredCourses.length > 0
                      ? "Guided Learning Flow"
                      : cfg.featured_courses.eyebrow}
                  </p>
                  <h2
                    id="guided-heading"
                    className="font-serif text-2xl sm:text-3xl font-bold text-navy tracking-tight"
                  >
                    {featuredCourses.length > 0
                      ? "Start with guided lessons"
                      : cfg.featured_courses.title}
                  </h2>
                  <p className="mt-3 text-slatebody text-base leading-relaxed max-w-[58ch]">
                    Every module follows a repeatable four-step study cycle. Toggle the sample steps
                    below to preview how student progress is tracked inside the LMS.
                  </p>

                  {/* Progress Indicator */}
                  <div className="mt-6">
                    <div className="flex items-center justify-between text-sm font-semibold text-navy mb-2">
                      <span>Sample Module: Kinematics &amp; Motion Graphs</span>
                      <span className="text-bronze" aria-live="polite">
                        {completedStepCount} of {GUIDED_LESSON_STEPS.length} completed
                      </span>
                    </div>
                    <div
                      className="w-full h-2.5 bg-stoneband rounded-full overflow-hidden"
                      role="progressbar"
                      aria-valuenow={checklistPercent}
                      aria-valuemin={0}
                      aria-valuemax={100}
                      aria-label="Sample module completion progress"
                    >
                      <div
                        className="h-full bg-navy rounded-full transition-all duration-300"
                        style={{ width: `${checklistPercent}%` }}
                      />
                    </div>
                  </div>

                  {/* Interactive Checklist */}
                  <ul className="mt-6 space-y-3" aria-label="Sample guided lesson steps">
                    {GUIDED_LESSON_STEPS.map((step) => (
                      <li key={step.id}>
                        <label className="flex items-start gap-3.5 p-3.5 rounded-xl border border-borderui bg-canvas/60 hover:border-navy cursor-pointer transition-colors has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-navy">
                          <input
                            type="checkbox"
                            checked={Boolean(checkedSteps[step.id])}
                            onChange={(e) =>
                              setCheckedSteps((prev) => ({
                                ...prev,
                                [step.id]: e.target.checked,
                              }))
                            }
                            className="mt-1 w-5 h-5 rounded border-borderui accent-navy shrink-0 cursor-pointer"
                          />
                          <span className="flex-1">
                            <span className="block text-sm sm:text-base font-semibold text-navy">
                              {step.title}
                            </span>
                            <span className="block text-sm text-slatebody mt-0.5">
                              {step.detail}
                            </span>
                          </span>
                        </label>
                      </li>
                    ))}
                  </ul>
                </div>

                {/* Right Column: Early-Access Cohort Notification Form */}
                <div className="lg:col-span-5 p-6 sm:p-10 lg:p-12 bg-stoneband/55 flex flex-col justify-center">
                  <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-2">
                    Early Access
                  </p>
                  <h3 className="font-serif text-2xl font-bold text-navy">
                    {featuredCourses.length === 0
                      ? cfg.featured_courses.empty_title
                      : "New cohorts opening soon"}
                  </h3>
                  <p className="mt-2.5 text-sm sm:text-base text-slatebody leading-relaxed">
                    {featuredCourses.length === 0
                      ? cfg.featured_courses.empty_body
                      : "Instructors are finalizing upcoming SSC, HSC, and Admission cohorts. Select your primary track to receive an instant email alert as soon as approved courses publish."}
                  </p>

                  <form onSubmit={handleEarlyAccessSubmit} noValidate className="mt-6 space-y-4">
                    <div>
                      <label
                        htmlFor="ea-track"
                        className="block text-sm font-semibold text-navy mb-1.5"
                      >
                        Preferred Science Track
                      </label>
                      <select
                        id="ea-track"
                        name="track"
                        value={eaTrack}
                        onChange={(e) => setEaTrack(e.target.value)}
                        className="w-full min-h-[46px] px-3.5 py-2.5 rounded-lg border border-borderui bg-white text-navy text-sm sm:text-base focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-navy"
                      >
                        <option value="Physics">Physics (SSC / HSC / Admission)</option>
                        <option value="Chemistry">Chemistry (SSC / HSC / Admission)</option>
                        <option value="Biology">Biology (SSC / HSC / Admission)</option>
                        <option value="Mathematics">Mathematics (SSC / HSC / Admission)</option>
                        <option value="ICT">ICT (HSC)</option>
                        <option value="Exam Prep">Exam Prep &amp; Mock Tests</option>
                      </select>
                    </div>

                    <div>
                      <label
                        htmlFor="ea-email"
                        className="block text-sm font-semibold text-navy mb-1.5"
                      >
                        Student or Parent Email
                      </label>
                      <input
                        type="email"
                        id="ea-email"
                        name="email"
                        required
                        value={eaEmail}
                        onChange={(e) => {
                          setEaEmail(e.target.value);
                          if (eaError) setEaError("");
                        }}
                        placeholder="name@example.com"
                        aria-invalid={eaError ? "true" : undefined}
                        aria-describedby={eaError ? "ea-email-error" : undefined}
                        className="w-full min-h-[46px] px-3.5 py-2.5 rounded-lg border border-borderui bg-white text-navy text-sm sm:text-base placeholder:text-slatebody focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-navy"
                      />
                      {eaError && (
                        <p
                          id="ea-email-error"
                          className="mt-1.5 text-sm font-medium text-red-700"
                          role="alert"
                        >
                          {eaError}
                        </p>
                      )}
                    </div>

                    <button
                      type="submit"
                      className="w-full inline-flex items-center justify-center min-h-[48px] px-5 py-3 rounded-lg bg-navy text-white text-sm sm:text-base font-semibold hover:bg-navy-light shadow-sm transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                    >
                      Notify me when courses open
                    </button>

                    {eaFeedback && (
                      <div
                        aria-live="polite"
                        className="p-3.5 rounded-lg bg-emerald-50 border border-emerald-700/30 text-emerald-900 text-sm font-medium"
                      >
                        {eaFeedback}
                      </div>
                    )}
                  </form>
                </div>
              </div>
            </div>
          </section>
        )}

        {/* 5. BOOKSHOP CATALOG ("Study materials by need") */}
        {cfg.bookshop_catalog.enabled && (
          <section
            id="bookshop"
            className="py-16 sm:py-20 lg:py-24 bg-stoneband border-y border-borderui"
            aria-labelledby="bookshop-heading"
          >
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
              <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4 mb-10">
                <div>
                  <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-2">
                    {cfg.bookshop_catalog.eyebrow}
                  </p>
                  <h2
                    id="bookshop-heading"
                    className="font-serif text-3xl sm:text-4xl font-bold text-navy tracking-tight"
                  >
                    {cfg.bookshop_catalog.title}
                  </h2>
                </div>
                <Link
                  to="/bookshop"
                  className="self-start sm:self-auto inline-flex items-center gap-1.5 min-h-[44px] px-2 text-sm font-semibold text-navy underline underline-offset-4 decoration-gold decoration-2 hover:text-bronze rounded-lg transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                >
                  {cfg.bookshop_catalog.action_label}
                  <span aria-hidden="true">&rarr;</span>
                </Link>
              </div>

              {/* 6 Interactive Bookshop Category Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-5">
                {cfg.bookshop_catalog.categories.map((category, idx) => {
                  const description =
                    BOOKSHOP_CATEGORY_DESCRIPTIONS[category.label.trim().toLowerCase()] ??
                    "Curated syllabus study materials";

                  if (featuredBooks.length > 0) {
                    return (
                      <Link
                        key={`${category.label}-${idx}`}
                        to="/bookshop"
                        search={{ q: category.query?.trim() || category.label } as never}
                        className="group text-left bg-white rounded-xl p-5 border border-borderui hover:border-navy hover:shadow-md transition-all flex items-center justify-between gap-4 min-h-[84px] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                      >
                        <div className="flex items-center gap-4">
                          <span className="w-11 h-11 rounded-lg bg-stoneband text-navy flex items-center justify-center shrink-0 group-hover:bg-navy group-hover:text-white transition-colors">
                            <BookCategoryIcon label={category.label} />
                          </span>
                          <div>
                            <h3 className="font-sans font-semibold text-base text-navy">
                              {category.label}
                            </h3>
                            <p className="text-sm text-slatebody">{description}</p>
                          </div>
                        </div>
                        <span
                          className="text-navy font-semibold transition-transform group-hover:translate-x-1"
                          aria-hidden="true"
                        >
                          &rarr;
                        </span>
                      </Link>
                    );
                  }

                  return (
                    <button
                      key={`${category.label}-${idx}`}
                      type="button"
                      onClick={() => openBookRequestForCategory(category.label)}
                      className="group text-left bg-white rounded-xl p-5 border border-borderui hover:border-navy hover:shadow-md transition-all flex items-center justify-between gap-4 min-h-[84px] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                    >
                      <div className="flex items-center gap-4">
                        <span className="w-11 h-11 rounded-lg bg-stoneband text-navy flex items-center justify-center shrink-0 group-hover:bg-navy group-hover:text-white transition-colors">
                          <BookCategoryIcon label={category.label} />
                        </span>
                        <div>
                          <h3 className="font-sans font-semibold text-base text-navy">
                            {category.label}
                          </h3>
                          <p className="text-sm text-slatebody">{description}</p>
                        </div>
                      </div>
                      <span
                        className="text-navy font-semibold transition-transform group-hover:translate-x-1"
                        aria-hidden="true"
                      >
                        &rarr;
                      </span>
                    </button>
                  );
                })}
              </div>

              {/* Featured Books Grid when books are published */}
              {cfg.featured_books.enabled && (booksLoading || featuredBooks.length > 0) && (
                <div className="mt-12">
                  <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
                    <div>
                      <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-1">
                        {cfg.featured_books.eyebrow}
                      </p>
                      <h3 className="font-serif text-2xl sm:text-3xl font-bold text-navy">
                        {offerBooks.length > 0
                          ? cfg.featured_books.offer_title
                          : cfg.featured_books.title}
                      </h3>
                    </div>
                    <Link
                      to="/bookshop"
                      className="self-start sm:self-auto inline-flex items-center gap-1.5 min-h-[44px] px-2 text-sm font-semibold text-navy underline underline-offset-4 decoration-gold decoration-2 hover:text-bronze rounded-lg transition-colors"
                    >
                      {cfg.featured_books.action_label}
                      <span aria-hidden="true">&rarr;</span>
                    </Link>
                  </div>
                  {booksLoading ? (
                    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-5">
                      {Array.from({ length: 6 }).map((_, index) => (
                        <div
                          key={index}
                          className="animate-pulse bg-white rounded-xl p-3 border border-borderui"
                        >
                          <div className="aspect-[2/3] rounded-lg bg-stoneband" />
                          <div className="mt-3 h-3 bg-stoneband rounded w-3/4" />
                          <div className="mt-2 h-3 bg-stoneband rounded w-1/2" />
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-5">
                      {featuredBooks.map((book) => (
                        <BookCard key={book.id} book={book} />
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* Compact Inline Review Banner + Expandable Book List Request */}
              <div
                ref={bookRequestPanelRef}
                className="mt-8 bg-white rounded-xl border border-borderui p-5 sm:p-6 shadow-sm"
              >
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                  <div className="flex items-start gap-3.5">
                    <span
                      className="mt-0.5 w-9 h-9 rounded-full bg-amber-100 text-bronze flex items-center justify-center shrink-0"
                      aria-hidden="true"
                    >
                      <Info className="w-5 h-5" />
                    </span>
                    <div>
                      <p className="font-semibold text-navy text-base">
                        {featuredBooks.length === 0
                          ? cfg.featured_books.empty_title
                          : "Curated books are reviewed by faculty before listing"}
                      </p>
                      <p className="text-sm text-slatebody mt-0.5">
                        {featuredBooks.length === 0
                          ? cfg.featured_books.empty_body
                          : "Our faculty verifies every textbook and lab manual against the current board syllabus. Request the upcoming book list for your class."}
                      </p>
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => setBookPanelOpen((prev) => !prev)}
                    aria-expanded={bookPanelOpen}
                    aria-controls="book-request-panel"
                    className="self-start sm:self-auto shrink-0 inline-flex items-center justify-center min-h-[44px] px-5 py-2.5 rounded-lg bg-navy text-white text-sm font-semibold hover:bg-navy-light transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                  >
                    Request a Book List
                  </button>
                </div>

                {bookPanelOpen && (
                  <div id="book-request-panel" className="mt-5 pt-5 border-t border-borderui">
                    <form
                      onSubmit={handleBookRequestSubmit}
                      noValidate
                      className="grid grid-cols-1 sm:grid-cols-3 gap-4 items-end"
                    >
                      <div>
                        <label
                          htmlFor="br-category"
                          className="block text-sm font-semibold text-navy mb-1.5"
                        >
                          Material Category
                        </label>
                        <select
                          id="br-category"
                          value={brCategory}
                          onChange={(e) => setBrCategory(e.target.value)}
                          className="w-full min-h-[44px] px-3.5 py-2 rounded-lg border border-borderui bg-canvas text-navy text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
                        >
                          {cfg.bookshop_catalog.categories.map((cat) => (
                            <option key={cat.label} value={cat.label}>
                              {cat.label}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div>
                        <label
                          htmlFor="br-email"
                          className="block text-sm font-semibold text-navy mb-1.5"
                        >
                          Your Email
                        </label>
                        <input
                          ref={brEmailInputRef}
                          type="email"
                          id="br-email"
                          required
                          value={brEmail}
                          onChange={(e) => setBrEmail(e.target.value)}
                          placeholder="student@example.com"
                          className="w-full min-h-[44px] px-3.5 py-2 rounded-lg border border-borderui bg-canvas text-navy text-sm placeholder:text-slatebody focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
                        />
                      </div>
                      <div>
                        <button
                          type="submit"
                          className="w-full inline-flex items-center justify-center min-h-[44px] px-5 py-2.5 rounded-lg bg-bronze text-white text-sm font-semibold hover:bg-navy transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                        >
                          Send Book List Request
                        </button>
                      </div>
                    </form>
                    {brFeedback && (
                      <div
                        aria-live="polite"
                        className={`mt-3 text-sm font-medium ${
                          brFeedback.type === "error" ? "text-red-700" : "text-emerald-800"
                        }`}
                      >
                        {brFeedback.text}
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          </section>
        )}

        {/* 6. OUTCOMES ("Built for repeatable study habits") */}
        {cfg.outcomes.enabled && (
          <section
            id="outcomes"
            className="py-16 sm:py-20 lg:py-24 bg-canvas"
            aria-labelledby="outcomes-heading"
          >
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
              <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-12 items-center">
                <div className="lg:col-span-5">
                  <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-2">
                    {cfg.outcomes.eyebrow}
                  </p>
                  <h2
                    id="outcomes-heading"
                    className="font-serif text-3xl sm:text-4xl font-bold text-navy tracking-tight leading-tight"
                  >
                    {cfg.outcomes.title}
                  </h2>
                  <p className="mt-4 text-base text-slatebody leading-relaxed max-w-[60ch]">
                    {cfg.outcomes.body}
                  </p>
                </div>

                <div className="lg:col-span-7 grid grid-cols-1 sm:grid-cols-3 gap-4 sm:gap-5">
                  {cfg.outcomes.stats.map((stat, idx) => {
                    const subtext = OUTCOME_STAT_SUBTEXT[stat.label.trim().toLowerCase()];
                    return (
                      <div
                        key={`${stat.label}-${idx}`}
                        className="bg-white rounded-xl p-6 border border-borderui shadow-sm"
                      >
                        <p className="text-xs font-bold tracking-[0.14em] uppercase text-bronze">
                          {stat.label}
                        </p>
                        <p className="mt-3 font-serif text-4xl font-bold text-navy">{stat.value}</p>
                        {subtext && <p className="mt-1.5 text-sm text-slatebody">{subtext}</p>}
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          </section>
        )}

        {/* 7. PLATFORM PILLARS */}
        {cfg.info_panels.enabled && cfg.info_panels.items.length > 0 && (
          <section className="pb-16 sm:pb-20 lg:pb-24 bg-canvas" aria-label="Platform pillars">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {cfg.info_panels.items.map((panel, idx) => (
                  <article
                    key={`${panel.eyebrow}-${idx}`}
                    className="bg-white rounded-xl p-7 border border-borderui shadow-sm flex flex-col"
                  >
                    <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-2.5">
                      {panel.eyebrow}
                    </p>
                    <h3 className="font-serif text-2xl font-bold text-navy">{panel.title}</h3>
                    <p className="mt-3 text-slatebody text-sm sm:text-base leading-relaxed">
                      {panel.body}
                    </p>
                  </article>
                ))}
              </div>
            </div>
          </section>
        )}

        {/* 8. ACCESSIBLE FAQ SECTION (Native Semantic <details>/<summary>) */}
        <section
          id="faq"
          className="py-16 sm:py-20 lg:py-24 bg-stoneband border-t border-borderui"
          aria-labelledby="faq-heading"
        >
          <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="mb-10">
              <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-2">
                Frequently Asked Questions
              </p>
              <h2
                id="faq-heading"
                className="font-serif text-3xl sm:text-4xl font-bold text-navy tracking-tight"
              >
                Answers for students and parents
              </h2>
            </div>

            <div className="space-y-3.5">
              {FAQ_ITEMS.map((item) => (
                <details
                  key={item.question}
                  className="group bg-white rounded-xl border border-borderui open:border-navy shadow-sm transition-colors"
                >
                  <summary className="flex items-center justify-between gap-4 min-h-[56px] px-6 py-4 cursor-pointer font-semibold text-base sm:text-lg text-navy rounded-xl focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy">
                    <span>{item.question}</span>
                    <ChevronDown
                      className="w-5 h-5 text-navy shrink-0 transition-transform duration-200 group-open:rotate-180"
                      aria-hidden="true"
                    />
                  </summary>
                  <div className="px-6 pb-5 pt-2 text-slatebody text-sm sm:text-base leading-relaxed border-t border-borderui/60">
                    {item.answer}
                  </div>
                </details>
              ))}
            </div>
          </div>
        </section>
      </main>

      {/* 9. UTILITY FOOTER (Dark Navy 4-Column, preserving Inspire Logo) */}
      <PublicFooter
        brandName={cfg.footer.brand_name}
        tagline={cfg.footer.tagline}
        tracks={cfg.course_catalog.tracks}
        onTrackSelect={(title) => {
          setSearchQuery(title);
          setActiveLevel("all");
        }}
      />

      {/* ACCESSIBLE TRACK PREVIEW MODAL */}
      {selectedTrack && (
        <div
          className="fixed inset-0 z-50 overflow-y-auto bg-navy/70 backdrop-blur-sm p-4 sm:p-6 flex items-center justify-center"
          role="dialog"
          aria-modal="true"
          aria-labelledby="modal-track-title"
          onClick={(e) => {
            if (e.target === e.currentTarget) {
              setSelectedTrack(null);
              setModalFeedback(null);
            }
          }}
        >
          <div className="bg-white rounded-2xl border border-borderui shadow-2xl max-w-xl w-full overflow-hidden my-auto">
            {/* Modal Header */}
            <div className="px-6 py-5 bg-stoneband/70 border-b border-borderui flex items-start justify-between gap-4">
              <div>
                <p className="text-xs font-bold tracking-[0.14em] uppercase text-bronze mb-1">
                  {selectedTrack.countLabel} &bull; {selectedTrack.levels.join(" / ")}
                </p>
                <h3 id="modal-track-title" className="font-serif text-2xl font-bold text-navy">
                  {selectedTrack.title} Track
                </h3>
              </div>
              <button
                ref={closeTrackModalRef}
                type="button"
                onClick={() => {
                  setSelectedTrack(null);
                  setModalFeedback(null);
                }}
                className="inline-flex items-center justify-center min-w-[44px] min-h-[44px] rounded-lg text-navy hover:bg-white border border-transparent hover:border-borderui focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
                aria-label="Close track preview modal"
              >
                <X className="w-5 h-5" aria-hidden="true" />
              </button>
            </div>

            {/* Modal Body */}
            <div className="p-6">
              <p className="text-slatebody text-sm sm:text-base leading-relaxed">
                {selectedTrack.detail}
              </p>

              <div className="mt-5">
                <h4 className="font-sans text-xs font-bold tracking-[0.14em] uppercase text-navy mb-3">
                  Sample Module Outline
                </h4>
                <ul className="space-y-2.5 text-sm text-navy-light">
                  {selectedTrack.modules.map((mod) => (
                    <li
                      key={mod}
                      className="flex items-start gap-2.5 p-2.5 rounded-lg bg-canvas border border-borderui/80"
                    >
                      <Check className="w-4 h-4 text-bronze mt-0.5 shrink-0" aria-hidden="true" />
                      <span>{mod}</span>
                    </li>
                  ))}
                </ul>
              </div>

              {modalFeedback && (
                <div
                  aria-live="polite"
                  className="mt-5 p-3.5 rounded-lg bg-emerald-50 border border-emerald-700/30 text-emerald-900 text-sm font-medium"
                >
                  {modalFeedback}
                </div>
              )}

              {/* Modal Actions */}
              <div className="mt-6 pt-5 border-t border-borderui flex flex-col sm:flex-row gap-3">
                <Link
                  to="/courses"
                  search={{ q: selectedTrack.query } as never}
                  className="flex-1 inline-flex items-center justify-center min-h-[46px] px-4 py-2.5 rounded-lg border border-navy text-navy font-semibold text-sm hover:bg-stoneband transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                >
                  Browse {selectedTrack.title} Courses
                </Link>
                <Link
                  to="/register"
                  onClick={() =>
                    setModalFeedback(
                      `Redirecting to registration for ${selectedTrack.title} enrollment.`,
                    )
                  }
                  className="flex-1 inline-flex items-center justify-center min-h-[46px] px-4 py-2.5 rounded-lg bg-navy text-white font-semibold text-sm hover:bg-navy-light transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
                >
                  Request Enrollment
                </Link>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

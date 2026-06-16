import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";

import libraryHero from "@/assets/library-hero.jpg";
import { BookCard } from "@/components/bookshop/book-card";
import { CourseCard, CourseCardSkeleton } from "@/components/course/course-card";
import { SlideCarousel } from "@/components/marketing/ad-carousel";
import { listBooks } from "@/lib/api/bookshop";
import { listCourses } from "@/lib/api/courses";
import { listSlides, type Slide } from "@/lib/api/slides";

const scienceTracks = [
  { title: "Physics", detail: "Mechanics, waves, electricity, and modern physics" },
  { title: "Chemistry", detail: "Physical, organic, inorganic, and lab problem solving" },
  { title: "Biology", detail: "Cell biology, genetics, physiology, and ecology" },
  { title: "Mathematics", detail: "Algebra, calculus, statistics, and exam practice" },
  { title: "ICT", detail: "Programming fundamentals, systems, and digital literacy" },
  { title: "Exam Prep", detail: "Board, admission, and college readiness practice" },
];

const bookCategories = [
  "Textbooks",
  "Workbooks",
  "Solution guides",
  "Lab manuals",
  "Admission prep",
  "Digital notes",
];

const stats = [
  { label: "Science tracks", value: "6" },
  { label: "Practice focused", value: "100%" },
  { label: "Live support", value: "Weekly" },
];

const fallbackSlides: Slide[] = [
  {
    id: "fallback-science",
    title: "Master physics, chemistry, biology, math, and exam prep.",
    subtitle:
      "Structured courses, live classes, assessments, progress tracking, and certificates in one LMS.",
    link_url: "/courses",
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

export const Route = createFileRoute("/")({
  head: () => ({
    meta: [
      { title: "Inspire LMS - Science tutoring for high school and college" },
      {
        name: "description",
        content:
          "Inspire LMS provides high school and college-level science tutoring, live classes, course catalogs, assessments, certificates, and a study bookshop.",
      },
      { property: "og:title", content: "Inspire LMS" },
      {
        property: "og:description",
        content: "Science tutoring, course learning, live classes, and curated study books.",
      },
    ],
  }),
  component: Landing,
});

function Landing() {
  const { data: slidesData } = useQuery({
    queryKey: ["slides", "public"],
    queryFn: listSlides,
    retry: false,
    staleTime: 5 * 60_000,
  });

  const { data: coursesData, isLoading: coursesLoading } = useQuery({
    queryKey: ["courses", { recent: true }],
    queryFn: () => listCourses({ sort: "published_at", order: "desc", limit: 6 }),
    retry: false,
    staleTime: 60_000,
  });

  const { data: booksData, isLoading: booksLoading } = useQuery({
    queryKey: ["books", { offers: true }],
    queryFn: () => listBooks({ sort: "newest", limit: 12 }),
    retry: false,
    staleTime: 60_000,
  });

  const slides = slidesData?.slides?.length ? slidesData.slides : fallbackSlides;
  const featuredCourses = coursesData?.data ?? [];
  const hasCourseError = !coursesLoading && !coursesData;
  const hasBookError = !booksLoading && !booksData;
  const offerBooks =
    booksData?.data?.filter((book) => book.is_free === true || book.price_cents === 0) ?? [];
  const featuredBooks = offerBooks.length > 0 ? offerBooks : (booksData?.data?.slice(0, 6) ?? []);

  return (
    <div className="min-h-screen overflow-x-hidden bg-surface text-brand font-sans flex flex-col">
      <header className="px-6 md:px-12 lg:px-20 py-5 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-brand/10">
        <Link to="/" className="font-serif italic text-2xl text-accent tracking-tight">
          Inspire LMS
        </Link>
        <nav className="flex w-full flex-wrap items-center gap-x-4 gap-y-2 text-sm sm:w-auto">
          <Link to="/courses" className="text-brand/60 hover:text-brand transition-colors">
            Courses
          </Link>
          <Link to="/bookshop" className="text-brand/60 hover:text-brand transition-colors">
            Bookshop
          </Link>
          <Link
            to="/forum"
            className="hidden sm:inline text-brand/60 hover:text-brand transition-colors"
          >
            Forum
          </Link>
          <Link to="/login" className="text-brand/60 hover:text-brand transition-colors">
            Sign in
          </Link>
          <Link
            to="/register"
            className="bg-brand text-white px-4 py-2.5 hover:bg-brand/90 transition-colors"
          >
            Register
          </Link>
        </nav>
      </header>

      <SlideCarousel slides={slides} />

      <main className="flex-1">
        <section className="px-6 md:px-12 lg:px-20 py-14 border-t border-brand/10">
          <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between mb-8 gap-4">
            <div>
              <p className="eyebrow text-accent mb-2">Course catalog</p>
              <h2 className="font-serif text-2xl sm:text-3xl md:text-4xl">
                Choose your science track
              </h2>
            </div>
            <Link
              to="/courses"
              className="text-sm text-brand/70 hover:text-brand underline underline-offset-4"
            >
              View all courses
            </Link>
          </div>
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {scienceTracks.map((track) => (
              <Link
                key={track.title}
                to="/courses"
                search={{ q: track.title } as never}
                className="border border-brand/10 bg-white/45 p-5 hover:bg-white transition-colors"
              >
                <p className="font-serif text-xl">{track.title}</p>
                <p className="mt-2 text-sm text-brand/60 leading-relaxed">{track.detail}</p>
              </Link>
            ))}
          </div>
        </section>

        <section className="px-6 md:px-12 lg:px-20 py-14 border-t border-brand/10 bg-white/30">
          <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow text-accent mb-2">Featured courses</p>
              <h2 className="font-serif text-2xl sm:text-3xl md:text-4xl">
                Start with guided lessons
              </h2>
            </div>
            <Link
              to="/courses"
              className="text-sm text-brand/70 hover:text-brand underline underline-offset-4"
            >
              Browse catalog
            </Link>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {coursesLoading
              ? Array.from({ length: 3 }).map((_, index) => <CourseCardSkeleton key={index} />)
              : featuredCourses.map((course) => <CourseCard key={course.id} course={course} />)}
          </div>
          {!coursesLoading && featuredCourses.length === 0 && (
            <EmptyFeatureState
              title={
                hasCourseError
                  ? "Course catalog is temporarily unavailable"
                  : "Courses are being prepared"
              }
              body={
                hasCourseError
                  ? "The LMS could not reach the course service. Try the catalog again shortly."
                  : "Published science courses will appear here as instructors submit them for approval."
              }
              to="/courses"
              action="Open catalog"
            />
          )}
        </section>

        <section className="px-6 md:px-12 lg:px-20 py-14 border-t border-brand/10">
          <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow text-accent mb-2">Bookshop catalog</p>
              <h2 className="font-serif text-2xl sm:text-3xl md:text-4xl">
                Study materials by need
              </h2>
            </div>
            <Link
              to="/bookshop"
              className="text-sm text-brand/70 hover:text-brand underline underline-offset-4"
            >
              Visit bookshop
            </Link>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
            {bookCategories.map((category) => (
              <Link
                key={category}
                to="/bookshop"
                search={{ q: category } as never}
                className="border border-brand/10 bg-white/45 px-4 py-5 text-sm font-medium hover:bg-white transition-colors"
              >
                {category}
              </Link>
            ))}
          </div>
        </section>

        <section className="px-6 md:px-12 lg:px-20 py-14 border-t border-brand/10 bg-brand/[0.02]">
          <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow text-accent mb-2">Featured books</p>
              <h2 className="font-serif text-2xl sm:text-3xl md:text-4xl">
                {offerBooks.length > 0 ? "Free and offer books" : "Recent arrivals"}
              </h2>
            </div>
            <Link
              to="/bookshop"
              className="text-sm text-brand/70 hover:text-brand underline underline-offset-4"
            >
              See all books
            </Link>
          </div>
          {booksLoading ? (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-6">
              {Array.from({ length: 6 }).map((_, index) => (
                <div key={index} className="animate-pulse">
                  <div className="aspect-[2/3] bg-brand/5" />
                  <div className="mt-3 h-3 bg-brand/5 w-3/4" />
                  <div className="mt-2 h-3 bg-brand/5 w-1/2" />
                </div>
              ))}
            </div>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-6">
              {featuredBooks.slice(0, 6).map((book) => (
                <BookCard key={book.id} book={book} />
              ))}
            </div>
          )}
          {!booksLoading && featuredBooks.length === 0 && (
            <EmptyFeatureState
              title={
                hasBookError ? "Bookshop is temporarily unavailable" : "Books are being prepared"
              }
              body={
                hasBookError
                  ? "The LMS could not reach the book catalog. Try the bookshop again shortly."
                  : "Featured textbooks, workbooks, and lab manuals will appear here after admin approval."
              }
              to="/bookshop"
              action="Open bookshop"
            />
          )}
        </section>

        <section className="px-6 md:px-12 lg:px-20 py-14 border-t border-brand/10">
          <div className="grid lg:grid-cols-[1fr_1.2fr] gap-10 items-start">
            <div>
              <p className="eyebrow text-accent mb-2">Outcomes</p>
              <h2 className="font-serif text-2xl sm:text-3xl md:text-4xl">
                Built for repeatable study habits
              </h2>
              <p className="mt-4 text-brand/65 leading-relaxed max-w-xl">
                Inspire combines recorded lessons, live classes, quizzes, assignments, points,
                certificates, and book access into one practical LMS for science tutoring.
              </p>
            </div>
            <div className="grid sm:grid-cols-3 gap-4">
              {stats.map((stat) => (
                <div key={stat.label} className="border border-brand/10 bg-white/50 p-5">
                  <p className="eyebrow text-brand/45">{stat.label}</p>
                  <p className="mt-3 font-serif text-3xl">{stat.value}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="px-6 md:px-12 lg:px-20 py-14 border-t border-brand/10 bg-white/30">
          <div className="grid lg:grid-cols-3 gap-6">
            <InfoPanel
              eyebrow="Instructors"
              title="Teacher-led science paths"
              body="Courses are organized by subject, level, modules, chapters, and lessons so students always know what to do next."
            />
            <InfoPanel
              eyebrow="Testimonials"
              title="Clarity before decoration"
              body="Dashboards prioritize next lesson, live class, pending assignment, approval status, and progress."
            />
            <InfoPanel
              eyebrow="FAQ"
              title="Free and approval-based access"
              body="Students can browse previews publicly, enroll in free courses instantly, and request admin approval for paid courses or books."
            />
          </div>
        </section>
      </main>

      <footer className="px-6 md:px-12 lg:px-20 py-8 border-t border-brand/5 flex flex-wrap justify-between items-center gap-4 text-sm text-brand/45">
        <span>© {new Date().getFullYear()} Inspire LMS</span>
        <span>Science tutoring for high school and college learners.</span>
      </footer>
    </div>
  );
}

function InfoPanel({ eyebrow, title, body }: { eyebrow: string; title: string; body: string }) {
  return (
    <article className="border border-brand/10 bg-white/50 p-6">
      <p className="eyebrow text-accent mb-3">{eyebrow}</p>
      <h3 className="font-serif text-2xl">{title}</h3>
      <p className="mt-3 text-sm text-brand/65 leading-relaxed">{body}</p>
    </article>
  );
}

function EmptyFeatureState({
  title,
  body,
  to,
  action,
}: {
  title: string;
  body: string;
  to: "/courses" | "/bookshop";
  action: string;
}) {
  return (
    <div className="mt-6 border border-dashed border-brand/15 bg-white/45 px-6 py-8 text-center">
      <p className="font-serif text-2xl">{title}</p>
      <p className="mx-auto mt-2 max-w-xl text-sm leading-relaxed text-brand/60">{body}</p>
      <Link
        to={to}
        className="mt-5 inline-flex bg-brand px-5 py-2.5 text-sm font-medium text-white hover:bg-brand/90"
      >
        {action}
      </Link>
    </div>
  );
}

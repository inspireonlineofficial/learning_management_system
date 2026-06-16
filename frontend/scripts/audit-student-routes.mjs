import { chromium } from "@playwright/test";
import fs from "node:fs/promises";
import path from "node:path";
import process from "node:process";

const baseURL = process.env.AUDIT_BASE_URL ?? "http://127.0.0.1:3000";
const outputDir =
  process.env.AUDIT_OUTPUT_DIR ?? path.resolve(process.cwd(), "../screenshots/route-audit/student");

const ids = {
  courseId: "11111111-1111-4111-8111-111111111111",
  bookId: "22222222-2222-4222-8222-222222222222",
  orderId: "33333333-3333-4333-8333-333333333333",
  quizId: "44444444-4444-4444-8444-444444444444",
  attemptId: "55555555-5555-4555-8555-555555555555",
  assignmentId: "66666666-6666-4666-8666-666666666666",
  sessionId: "77777777-7777-4777-8777-777777777777",
  threadId: "88888888-8888-4888-8888-888888888888",
};

const routeSpecs = [
  ["/onboarding/student-profile", "student-onboarding-profile"],
  ["/student/search", "student-search"],
  ["/student/calendar", "student-calendar"],
  ["/student/my-courses", "student-my-courses"],
  ["/student/progress", "student-progress"],
  [`/student/progress/${ids.courseId}`, "student-course-progress"],
  [`/student/player/${ids.courseId}`, "student-player"],
  [`/student/checkout/course/${ids.courseId}`, "student-checkout-compat-course"],
  [`/student/checkout/book/${ids.bookId}`, "student-checkout-compat-book"],
  ["/student/checkout/unsupported/example", "student-checkout-unsupported"],
  [`/student/checkout/${ids.courseId}`, "student-course-checkout"],
  [`/student/bookshop/checkout/${ids.bookId}`, "student-book-checkout"],
  ["/student/bookshop", "student-bookshop"],
  [`/student/bookshop/${ids.bookId}`, "student-book-detail"],
  ["/student/bookshop/cart", "student-bookshop-cart"],
  ["/student/bookshop/library", "student-bookshop-library"],
  [`/student/bookshop/reader/${ids.bookId}`, "student-book-reader"],
  ["/student/bookshop/orders", "student-bookshop-orders"],
  [`/student/bookshop/orders/${ids.orderId}`, "student-bookshop-order-detail"],
  ["/student/bookshop/requests", "student-bookshop-requests"],
  ["/student/assessments", "student-assessments"],
  [`/student/assessments/${ids.quizId}`, "student-quiz-detail"],
  [`/student/assessments/${ids.quizId}/attempt`, "student-quiz-attempt"],
  [`/student/assessments/${ids.quizId}/result/${ids.attemptId}`, "student-quiz-result"],
  [`/student/assessments/result/${ids.attemptId}`, "student-quiz-result-compat"],
  ["/student/assignments", "student-assignments"],
  [`/student/assignments/${ids.assignmentId}`, "student-assignment-detail"],
  [`/student/assignments/${ids.assignmentId}/submit`, "student-assignment-submit"],
  ["/student/live-classes", "student-live-classes"],
  [`/student/live-classes/${ids.sessionId}`, "student-live-session"],
  [`/student/live-classes/${ids.sessionId}/room`, "student-live-room"],
  ["/student/points", "student-points"],
  ["/student/leaderboard", "student-leaderboard"],
  ["/student/achievements", "student-achievements"],
  ["/student/certificates", "student-certificates"],
  [`/student/certificates/${ids.courseId}`, "student-certificate-detail"],
  ["/student/downloads", "student-downloads"],
  ["/student/forum", "student-forum"],
  [`/student/forum/${ids.threadId}`, "student-forum-thread"],
  ["/student/notifications", "student-notifications"],
  ["/student/settings", "student-settings"],
];

const now = new Date("2026-06-16T12:00:00.000Z").toISOString();
const course = {
  id: ids.courseId,
  title: "Advanced Physics for STEM Students",
  slug: "advanced-physics-stem",
  description: "Mechanics, waves, electricity, and lab-style problem solving.",
  subject: "Physics",
  level: "college",
  price: 49,
  currency: "USD",
  thumbnail_url: "https://images.unsplash.com/photo-1636466497217-26a8cbeaf0aa",
  instructor: { id: "teacher-1", full_name: "Dr. Farah Rahman" },
  instructor_name: "Dr. Farah Rahman",
  average_rating: 4.8,
  enrolled_count: 128,
  is_free: false,
  status: "published",
  modules: [
    {
      id: "module-1",
      title: "Forces and Motion",
      lessons: [
        {
          id: "lesson-1",
          title: "Newtonian mechanics",
          type: "video",
          duration_minutes: 18,
          is_preview: true,
        },
      ],
    },
  ],
};

const book = {
  id: ids.bookId,
  title: "Organic Chemistry Companion",
  author: "Nadia Karim",
  subject: "Chemistry",
  class_grade: "College",
  description: "A concise companion for reactions, mechanisms, and practice sets.",
  format: "digital",
  price: 24,
  currency: "USD",
  cover_url: "https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c",
  physical_stock: 8,
  is_active: true,
};

const assignment = {
  id: ids.assignmentId,
  course_id: ids.courseId,
  course_title: course.title,
  title: "Lab analysis: projectile motion",
  description: "Submit your measured values, error analysis, and conclusion.",
  due_at: "2026-06-24T18:00:00.000Z",
  submission_type: "both",
  total_points: 100,
  total_marks: 100,
  status: "not_submitted",
  brief: "Analyze lab data and explain uncertainty.",
  instructions_html: "<p>Include graphs, sample calculations, and final interpretation.</p>",
  resources: [],
  submission: null,
};

const quiz = {
  id: ids.quizId,
  course_id: ids.courseId,
  title: "Kinematics checkpoint",
  time_limit_seconds: 1800,
  max_attempts: 2,
  passing_score_percent: 70,
  attempts_used: 1,
  latest_attempt: {
    id: ids.attemptId,
    status: "submitted",
    score_percent: 84,
    passed: true,
  },
};

const liveSession = {
  id: ids.sessionId,
  course_id: ids.courseId,
  course_title: course.title,
  title: "Live problem solving: energy conservation",
  description: "Work through conservation problems with instructor feedback.",
  scheduled_at: "2026-06-20T14:00:00.000Z",
  starts_at: "2026-06-20T14:00:00.000Z",
  ends_at: "2026-06-20T15:00:00.000Z",
  duration_minutes: 60,
  status: "scheduled",
  host_name: "Dr. Farah Rahman",
  join_url: "https://meet.example.test/inspire",
  recording_url: null,
};

const thread = {
  id: ids.threadId,
  title: "How do I choose a sign convention for force diagrams?",
  body: "I keep mixing up positive and negative directions in mechanics problems.",
  content: "I keep mixing up positive and negative directions in mechanics problems.",
  author: { id: "student-2", full_name: "Amina H.", role: "student" },
  course_id: ids.courseId,
  course_title: course.title,
  created_at: now,
  updated_at: now,
  reply_count: 2,
  upvotes: 5,
  status: "active",
};

function responseFor(url, method) {
  const { pathname, searchParams } = new URL(url);
  if (!pathname.startsWith("/v1/")) return undefined;

  if (pathname === "/v1/auth/me") {
    return {
      id: "00000000-0000-4000-8000-000000000001",
      email: "student.audit@example.com",
      full_name: "Student Route Auditor",
      role: "student",
      onboarded: true,
      profile_complete: true,
    };
  }
  if (pathname === "/v1/auth/me/settings") {
    return {
      email_notifications: true,
      push_notifications: true,
      newsletter_opt_in: false,
      language: "en",
      timezone: "Asia/Dhaka",
    };
  }
  if (pathname === "/v1/onboarding/student-profile") {
    return {
      school_name: "Inspire Science College",
      class_grade: "College freshman",
      roll_number: "PHY-101",
      date_of_birth: "2005-05-10",
      gender: "",
      guardian_name: "Rahman Karim",
      guardian_contact: "+8801000000000",
      profile_complete: true,
    };
  }
  if (pathname === "/v1/student/dashboard") {
    return {
      stats: {
        enrolled_courses: 1,
        completed_courses: 0,
        hours_learned: 12,
        points: 420,
        streak_days: 5,
      },
      continue_learning: [enrollment()],
      upcoming_live: [
        {
          id: ids.sessionId,
          title: liveSession.title,
          starts_at: liveSession.starts_at,
          course_title: course.title,
        },
      ],
      recent_achievements: [
        { id: "ach-1", title: "Physics starter", earned_at: now, icon: "spark" },
      ],
    };
  }
  if (pathname === "/v1/search") {
    return {
      items: [
        {
          type: "course",
          id: ids.courseId,
          title: course.title,
          snippet: "Physics course for high school and college students.",
          url: `/courses/${ids.courseId}`,
        },
        {
          type: "book",
          id: ids.bookId,
          title: book.title,
          snippet: "Chemistry reference for practice and review.",
          url: `/student/bookshop/${ids.bookId}`,
        },
      ],
    };
  }
  if (pathname === "/v1/student/enrollments") {
    return paginated([enrollment()]);
  }
  if (
    pathname === `/v1/courses/${ids.courseId}` ||
    pathname === `/v1/courses/slug/${ids.courseId}`
  ) {
    return course;
  }
  if (pathname === `/v1/stream/lessons/lesson-1/signed-url`) {
    return { signed_url: "https://storage.example.test/lesson-1.mp4" };
  }
  if (pathname === "/v1/student/live-sessions") {
    return { sessions: [liveSession] };
  }
  if (pathname === `/v1/student/live-sessions/${ids.sessionId}`) {
    return liveSession;
  }
  if (pathname === `/v1/live-sessions/${ids.sessionId}/join`) {
    return { session_id: ids.sessionId, room_token: "local-room-token" };
  }
  if (pathname === "/v1/bookshop/books") {
    return paginated([book]);
  }
  if (pathname === `/v1/bookshop/books/${ids.bookId}`) {
    return book;
  }
  if (pathname === "/v1/bookshop/cart") {
    return { id: "cart-1", items: [], subtotal_cents: 0, total_cents: 0, currency: "USD" };
  }
  if (pathname === `/v1/student/bookshop/reader/${ids.bookId}/access`) {
    return { access_url: "https://reader.example.test/book.pdf", last_page_read: 12 };
  }
  if (pathname === "/v1/student/bookshop/orders") {
    return {
      items: [
        {
          id: ids.orderId,
          status: "placed",
          total: 24,
          currency: "USD",
          created_at: now,
          items: [{ id: "item-1", title: book.title, quantity: 1, unit_price: 24 }],
        },
      ],
    };
  }
  if (pathname === `/v1/student/bookshop/orders/${ids.orderId}`) {
    return {
      id: ids.orderId,
      status: "placed",
      total: 24,
      currency: "USD",
      created_at: now,
      shipping_address: "House 12, Science Road, Dhaka",
      items: [{ id: "item-1", title: book.title, quantity: 1, unit_price: 24 }],
    };
  }
  if (pathname === "/v1/student/requests") {
    return {
      items: [
        {
          id: "request-1",
          kind: "book_purchase",
          item_id: ids.bookId,
          item_title: book.title,
          status: "pending",
          amount_cents: 2400,
          currency: "USD",
          created_at: now,
          file_name: "student-id.pdf",
        },
      ],
    };
  }
  if (pathname === "/v1/student/assessments") {
    return { quizzes: [quiz] };
  }
  if (pathname === `/v1/student/assessments/${ids.quizId}`) {
    return quiz;
  }
  if (pathname === `/v1/quizzes/${ids.quizId}/attempts`) {
    return {
      id: ids.attemptId,
      quiz_id: ids.quizId,
      started_at: now,
      expires_at: "2026-06-16T12:30:00.000Z",
      questions: [
        {
          id: "question-1",
          body: "A car accelerates uniformly from rest. Which graph is linear?",
          type: "single",
          options: [
            { id: "option-1", body: "Velocity vs time" },
            { id: "option-2", body: "Position vs time" },
          ],
        },
      ],
    };
  }
  if (pathname === `/v1/student/assessments/${ids.quizId}/attempts/${ids.attemptId}`) {
    return {
      id: ids.attemptId,
      status: "submitted",
      started_at: now,
      submitted_at: now,
      score_percent: 84,
      passed: true,
      points_awarded: 25,
    };
  }
  if (pathname === "/v1/student/assignments") {
    return { assignments: [assignment], meta: meta(1) };
  }
  if (pathname === `/v1/student/assignments/${ids.assignmentId}`) {
    return assignment;
  }
  if (pathname === "/v1/student/points") {
    return {
      total: 420,
      level: 3,
      rank: 8,
      streak_days: 5,
      longest_streak_days: 9,
      this_week: 85,
      this_month: 240,
      daily: [
        { date: "2026-06-12", points: 20 },
        { date: "2026-06-13", points: 45 },
        { date: "2026-06-14", points: 10 },
        { date: "2026-06-15", points: 0 },
        { date: "2026-06-16", points: 30 },
      ],
      by_source: [
        { source: "lesson_completed", points: 180 },
        { source: "quiz_passed", points: 160 },
        { source: "assignment_submitted", points: 80 },
      ],
      milestones: [
        { id: "milestone-1", label: "First 100 points", threshold: 100, achieved_at: now },
        { id: "milestone-2", label: "500 point scholar", threshold: 500, achieved_at: null },
      ],
      recent_events: [
        { id: "pts-1", points: 25, reason: "Quiz passed", created_at: now },
        { id: "pts-2", points: 10, reason: "Lesson completed", created_at: now },
      ],
    };
  }
  if (pathname === "/v1/leaderboard") {
    return {
      entries: [
        { rank: 1, student_id: "student-9", display_name: "Top Scholar", score: 920 },
        {
          rank: 8,
          student_id: "00000000-0000-4000-8000-000000000001",
          display_name: "You",
          score: 420,
        },
      ],
    };
  }
  if (pathname === `/v1/student/certificates/${ids.courseId}`) {
    return {
      id: "cert-1",
      course_id: ids.courseId,
      course_title: course.title,
      student_name: "Student Route Auditor",
      issued_at: now,
      verification_code: "INSPIRE-PHY-001",
      certificate_url: "https://certificates.example.test/cert-1.pdf",
    };
  }
  if (pathname === "/v1/forum/posts") {
    return { posts: [thread], meta: meta(1) };
  }
  if (pathname === `/v1/forum/posts/${ids.threadId}`) {
    return { ...thread, comments: [comment()] };
  }
  if (pathname === `/v1/forum/posts/${ids.threadId}/comments`) {
    return { comments: [comment()] };
  }
  if (pathname === "/v1/notifications") {
    return {
      items: [
        {
          id: "notif-1",
          type: "assignment_due",
          title: "Assignment due soon",
          body: "Projectile motion lab is due this week.",
          created_at: now,
          action_url: `/student/assignments/${ids.assignmentId}`,
        },
      ],
      unread_count: 1,
    };
  }

  if (method !== "GET") {
    return { ok: true, id: "audit-action" };
  }

  return { __unmocked: pathname, search: searchParams.toString() };
}

function enrollment() {
  return {
    id: "enrollment-1",
    course,
    enrolled_at: now,
    progress_percent: 48,
    last_accessed_at: now,
    next_lesson: { id: "lesson-1", title: "Newtonian mechanics", module_id: "module-1" },
    completed_at: null,
  };
}

function comment() {
  return {
    id: "comment-1",
    post_id: ids.threadId,
    body: "Pick a positive axis first, then stay consistent.",
    author: { id: "teacher-1", full_name: "Dr. Farah Rahman", role: "teacher" },
    created_at: now,
    status: "active",
  };
}

function meta(total) {
  return { page: 1, limit: 20, total, total_pages: 1 };
}

function paginated(data) {
  return { data, meta: meta(data.length) };
}

function pathToSlug(route) {
  return route
    .replace(/^\//, "")
    .replace(/[/$]/g, "-")
    .replace(/[^a-zA-Z0-9-]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/-$/, "");
}

async function main() {
  await fs.mkdir(outputDir, { recursive: true });
  const browser = await chromium.launch({ headless: true });
  const report = [];

  for (const viewport of [
    { label: "desktop", width: 1440, height: 950 },
    { label: "mobile", width: 390, height: 844 },
  ]) {
    const context = await browser.newContext({
      viewport: { width: viewport.width, height: viewport.height },
      baseURL,
    });

    await context.route("**/*", async (route) => {
      const request = route.request();
      const url = request.url();
      if (new URL(url).pathname.startsWith("/v1/")) {
        const body = responseFor(url, request.method());
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(body ?? { ok: true }),
        });
        return;
      }
      if (!url.startsWith(baseURL) && ["image", "font", "media"].includes(request.resourceType())) {
        await route.fulfill({
          status: 204,
          contentType: request.resourceType() === "font" ? "font/woff2" : "image/png",
          body: "",
        });
        return;
      }
      await route.continue();
    });

    await context.addInitScript(
      (session) => {
        window.localStorage.setItem("inspire.session", JSON.stringify(session));
        window.localStorage.setItem(
          "inspire:quiz-attempt:55555555-5555-4555-8555-555555555555",
          "44444444-4444-4444-8444-444444444444",
        );
      },
      {
        accessToken: "audit-access-token",
        refreshToken: "audit-refresh-token",
        user: {
          id: "00000000-0000-4000-8000-000000000001",
          email: "student.audit@example.com",
          full_name: "Student Route Auditor",
          role: "student",
          onboarded: true,
          profile_complete: true,
        },
      },
    );

    const page = await context.newPage();

    for (const [routePath, name] of routeSpecs) {
      const errors = [];
      const failedRequests = [];
      const consoleHandler = (message) => {
        if (message.type() === "error") errors.push(message.text());
      };
      const requestFailedHandler = (request) => {
        failedRequests.push(`${request.method()} ${request.url()} ${request.failure()?.errorText}`);
      };
      page.on("console", consoleHandler);
      page.on("requestfailed", requestFailedHandler);

      const url = routePath;
      let status = "ok";
      let finalURL = "";
      let title = "";
      let heading = "";
      let bodyText = "";
      try {
        await page.goto(url, { waitUntil: "domcontentloaded", timeout: 15000 });
        await page.waitForTimeout(700);
        finalURL = page.url();
        title = await page.title();
        ({ heading, bodyText } = await page.evaluate(() => ({
          heading: document.querySelector("h1")?.textContent ?? "",
          bodyText: document.body?.textContent ?? "",
        })));
        await page.screenshot({
          path: path.join(outputDir, `${name}-${viewport.label}.png`),
          fullPage: true,
        });
      } catch (error) {
        status = "failed";
        errors.push(error instanceof Error ? error.message : String(error));
      }

      const normalizedBody = bodyText.toLowerCase();
      const flags = [];
      if (normalizedBody.includes("404") || normalizedBody.includes("not found"))
        flags.push("not-found");
      if (normalizedBody.includes("something went wrong")) flags.push("generic-error");
      if (normalizedBody.includes("placeholder") || normalizedBody.includes("coming soon")) {
        flags.push("placeholder-copy");
      }
      if (finalURL.includes("/login")) flags.push("redirected-to-login");
      if (finalURL.includes("/403")) flags.push("redirected-to-403");

      report.push({
        route: routePath,
        screenshot: `${name}-${viewport.label}.png`,
        viewport: viewport.label,
        status,
        finalURL: finalURL.replace(baseURL, ""),
        title,
        heading: heading?.trim() ?? "",
        flags,
        consoleErrors: errors,
        failedRequests,
      });

      page.off("console", consoleHandler);
      page.off("requestfailed", requestFailedHandler);
    }

    await context.close();
  }

  await browser.close();

  const jsonPath = path.join(outputDir, "student-route-audit.json");
  await fs.writeFile(jsonPath, JSON.stringify(report, null, 2));
  await fs.writeFile(path.join(outputDir, "student-route-audit.md"), toMarkdown(report));
  console.log(`Saved ${report.length} route captures to ${outputDir}`);
  console.log(`Report: ${jsonPath}`);

  const failures = report.filter(
    (item) =>
      item.status !== "ok" ||
      item.flags.length ||
      item.consoleErrors.length ||
      item.failedRequests.length,
  );
  if (failures.length > 0) {
    console.log(`Audit findings: ${failures.length}`);
    for (const item of failures.slice(0, 30)) {
      console.log(
        `- ${item.viewport} ${item.route}: ${[
          item.status,
          ...item.flags,
          ...item.consoleErrors,
          ...item.failedRequests,
        ].join("; ")}`,
      );
    }
  }
}

function toMarkdown(report) {
  const lines = [
    "# Student Route Audit",
    "",
    `Base URL: ${baseURL}`,
    `Generated: ${new Date().toISOString()}`,
    "",
    "| Route | Viewport | Final URL | Heading | Issues | Screenshot |",
    "| :--- | :--- | :--- | :--- | :--- | :--- |",
  ];
  for (const item of report) {
    const issues = [
      item.status !== "ok" ? item.status : "",
      ...item.flags,
      ...item.consoleErrors,
      ...item.failedRequests,
    ]
      .filter(Boolean)
      .join("<br>");
    lines.push(
      `| ${item.route} | ${item.viewport} | ${item.finalURL} | ${item.heading.replaceAll("|", "\\|")} | ${
        issues || "None"
      } | ${item.screenshot} |`,
    );
  }
  lines.push("");
  return lines.join("\n");
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});

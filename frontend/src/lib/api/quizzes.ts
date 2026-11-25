import { apiRequest, resolveAssetUrl } from "./client";

// Student-side assessment API. Field names on the right of each mapping are
// the backend contract (internal/application/assessments/responses.go).

export type QuestionType =
  "single_choice" | "multi_select" | "true_false" | "short_answer" | "numeric" | "long_answer";

export type QuizOption = {
  id: string;
  text: string;
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  is_correct?: boolean;
};

export type QuizQuestion = {
  id: string;
  type: QuestionType;
  prompt: string;
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  points: number;
  position?: number;
  is_required?: boolean;
  options?: QuizOption[];
  hint?: string;
  unit?: string;
  max_words?: number;
  // present only on teacher views / released reviews
  correct_option_ids?: string[];
  correct_text?: string;
  explanation?: string;
};

export type QuizMode = "exam" | "practice";

export type QuizSummary = {
  id: string;
  course_id: string;
  course_title?: string;
  title: string;
  description?: string;
  instructions?: string;
  mode: QuizMode;
  time_limit_minutes?: number | null;
  time_limit_seconds: number;
  total_questions: number;
  total_points: number;
  passing_score: number;
  is_free?: boolean;
  is_published?: boolean;
  is_locked?: boolean;
  attempts_allowed?: number | null;
  attempts_used?: number;
  remaining_attempts?: number | null;
  best_score?: number | null;
  in_progress_attempt_id?: string | null;
  status?: "not_started" | "in_progress" | "completed" | "passed" | "failed" | "pending_grading";
  availability: "open" | "not_open" | "closed";
  available_from?: string | null;
  available_until?: string | null;
  /** Closing time (alias of available_until, used by lists and the calendar). */
  due_at?: string | null;
  calculator_policy: "none" | "basic" | "scientific" | "graphing";
  reference_material?: string;
  show_answers_after_submission?: boolean;
  latest_attempt_id?: string | null;
};

type BackendAttemptSummary = {
  id: string;
  quiz_id: string;
  status: string;
  started_at: string;
  submitted_at?: string | null;
  score_percent?: number | null;
  passed?: boolean | null;
  time_taken_seconds?: number | null;
  points_awarded?: number;
  grading_status?: "auto" | "pending" | "graded";
  score_points?: number | null;
  max_points?: number | null;
};

type BackendQuizSummary = {
  id: string;
  course_id: string;
  lesson_id?: string | null;
  title: string;
  time_limit_seconds: number;
  max_attempts: number;
  passing_score_percent: number;
  is_free?: boolean;
  is_published?: boolean;
  is_locked?: boolean;
  attempts_used?: number;
  latest_attempt?: BackendAttemptSummary | null;
  course_title?: string;
  description?: string;
  instructions?: string;
  mode?: QuizMode;
  question_count?: number;
  total_marks?: number;
  available_from?: string | null;
  available_until?: string | null;
  availability?: "open" | "not_open" | "closed";
  calculator_policy?: QuizSummary["calculator_policy"];
  reference_material?: string;
  remaining_attempts?: number | null;
  in_progress_attempt_id?: string | null;
  best_score_percent?: number | null;
  show_answers_after_submission?: boolean;
};

const attemptQuizKey = (attemptId: string) => `inspire:quiz-attempt:${attemptId}`;

function safeStorage(): Storage | null {
  try {
    return typeof window === "undefined" ? null : window.localStorage;
  } catch {
    return null;
  }
}

function rememberAttempt(attemptId: string, quizId: string) {
  try {
    safeStorage()?.setItem(attemptQuizKey(attemptId), quizId);
  } catch {
    /* storage full or blocked: the quiz ID is re-fetched when needed */
  }
}

async function resolveAttemptQuizId(attemptId: string) {
  const local = safeStorage()?.getItem(attemptQuizKey(attemptId));
  if (local) return local;
  const attempt = await apiRequest<{ quiz_id: string }>(
    `/v1/student/assessments/attempts/${encodeURIComponent(attemptId)}`,
    { auth: true },
  );
  rememberAttempt(attemptId, attempt.quiz_id);
  return attempt.quiz_id;
}

function toQuizSummary(q: BackendQuizSummary): QuizSummary {
  const latest = q.latest_attempt;
  let status: QuizSummary["status"] = "not_started";
  if (q.in_progress_attempt_id) status = "in_progress";
  else if (latest && latest.status !== "in_progress") {
    status =
      latest.grading_status === "pending" ? "pending_grading" : latest.passed ? "passed" : "failed";
  }
  return {
    id: q.id,
    course_id: q.course_id,
    course_title: q.course_title || undefined,
    title: q.title,
    description: q.description || undefined,
    instructions: q.instructions || undefined,
    mode: q.mode === "practice" ? "practice" : "exam",
    time_limit_seconds: q.time_limit_seconds,
    time_limit_minutes: q.time_limit_seconds ? Math.ceil(q.time_limit_seconds / 60) : null,
    total_questions: q.question_count ?? 0,
    total_points: q.total_marks ?? 0,
    passing_score: q.passing_score_percent,
    is_free: q.is_free ?? true,
    is_published: q.is_published ?? true,
    is_locked: q.is_locked ?? false,
    attempts_allowed: q.max_attempts || null,
    attempts_used: q.attempts_used ?? 0,
    remaining_attempts: q.remaining_attempts ?? null,
    best_score: q.best_score_percent ?? null,
    in_progress_attempt_id: q.in_progress_attempt_id ?? null,
    latest_attempt_id: latest?.id ?? null,
    status,
    availability: q.availability ?? "open",
    available_from: q.available_from ?? null,
    available_until: q.available_until ?? null,
    due_at: q.available_until ?? null,
    calculator_policy: q.calculator_policy ?? "none",
    reference_material: q.reference_material || undefined,
    show_answers_after_submission: q.show_answers_after_submission,
  };
}

type BackendQuestion = {
  id: string;
  body: string;
  type: string;
  position?: number;
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  marks?: number;
  is_required?: boolean;
  hint?: string;
  settings?: { unit?: string; max_words?: number };
  options?: Array<{
    id: string;
    body: string;
    content_type?: "text" | "image" | "text_image";
    image_url?: string;
    is_correct?: boolean;
  }>;
  correct_option_ids?: string[];
  explanation?: string;
};

export function toFrontendType(type: string): QuestionType {
  switch (type) {
    case "multiple":
    case "multi_select":
      return "multi_select";
    case "single":
    case "single_choice":
      return "single_choice";
    case "short_answer":
    case "numeric":
    case "long_answer":
    case "true_false":
      return type;
    default:
      return "true_false";
  }
}

function toQuestion(q: BackendQuestion): QuizQuestion {
  return {
    id: q.id,
    type: toFrontendType(q.type),
    prompt: q.body,
    position: q.position,
    content_type: q.content_type ?? (q.image_url ? (q.body ? "text_image" : "image") : "text"),
    image_url: q.image_url || undefined,
    points: q.marks ?? 1,
    is_required: q.is_required ?? true,
    hint: q.hint || undefined,
    unit: q.settings?.unit || undefined,
    max_words: q.settings?.max_words || undefined,
    options: q.options?.map((option) => ({
      id: option.id,
      text: option.body,
      content_type:
        option.content_type ?? (option.image_url ? (option.body ? "text_image" : "image") : "text"),
      image_url: option.image_url || undefined,
      is_correct: option.is_correct,
    })),
    correct_option_ids: q.correct_option_ids,
    explanation: q.explanation,
  };
}

export type Quiz = QuizSummary;

export type ShortAnswerValue = {
  text?: string;
  image_url?: string;
};

export type AnswerValue = string[] | string | ShortAnswerValue;
export type AnswerMap = Record<string, AnswerValue>;

export type QuizAttempt = {
  id: string;
  quiz_id: string;
  started_at: string;
  expires_at?: string | null;
  server_now?: string;
  submitted_at?: string | null;
  status: "in_progress" | "submitted" | "auto_submitted" | "graded";
  draft_revision: number;
  quiz_title?: string;
  mode?: QuizMode;
  instructions?: string;
  time_limit_seconds?: number;
  calculator_policy?: QuizSummary["calculator_policy"];
  reference_material?: string;
  total_marks?: number;
  current_question_index?: number;
  questions: QuizQuestion[];
  answers: AnswerMap;
};

type BackendAttempt = {
  id: string;
  quiz_id: string;
  started_at: string;
  expires_at?: string | null;
  server_now?: string;
  status?: string;
  draft_revision?: number;
  quiz_title?: string;
  mode?: QuizMode;
  instructions?: string;
  time_limit_seconds?: number;
  calculator_policy?: QuizSummary["calculator_policy"];
  reference_material?: string;
  total_marks?: number;
  questions: BackendQuestion[];
  answers?: AnswerMap | null;
};

function toAttempt(attempt: BackendAttempt): QuizAttempt {
  return {
    id: attempt.id,
    quiz_id: attempt.quiz_id,
    started_at: attempt.started_at,
    expires_at: attempt.expires_at ?? null,
    server_now: attempt.server_now,
    status: "in_progress",
    draft_revision: attempt.draft_revision ?? 0,
    quiz_title: attempt.quiz_title,
    mode: attempt.mode === "practice" ? "practice" : "exam",
    instructions: attempt.instructions || undefined,
    time_limit_seconds: attempt.time_limit_seconds,
    calculator_policy: attempt.calculator_policy ?? "none",
    reference_material: attempt.reference_material || undefined,
    total_marks: attempt.total_marks,
    questions: (attempt.questions ?? []).map(toQuestion),
    answers: attempt.answers ?? {},
  };
}

export function listCourseQuizzes(courseId: string) {
  return listMyQuizzes().then((result) => ({
    data: result.data.filter((quiz) => quiz.course_id === courseId),
  }));
}

export function listMyQuizzes(params: { status?: string; page?: number; limit?: number } = {}) {
  return apiRequest<{ quizzes: BackendQuizSummary[] }>("/v1/student/assessments", {
    auth: true,
    query: params,
  }).then((result) => {
    const quizzes = result.quizzes ?? [];
    return {
      data: quizzes.map(toQuizSummary),
      meta: {
        page: params.page ?? 1,
        limit: params.limit ?? quizzes.length,
        total: quizzes.length,
        total_pages: 1,
      },
    };
  });
}

export function getQuiz(quizId: string): Promise<Quiz> {
  return apiRequest<BackendQuizSummary>(`/v1/student/assessments/${encodeURIComponent(quizId)}`, {
    auth: true,
  }).then(toQuizSummary);
}

export function startQuizAttempt(quizId: string): Promise<QuizAttempt> {
  return apiRequest<BackendAttempt>(`/v1/quizzes/${encodeURIComponent(quizId)}/attempts`, {
    method: "POST",
    auth: true,
  }).then((attempt) => {
    rememberAttempt(attempt.id, quizId);
    return toAttempt(attempt);
  });
}

/**
 * Resume an attempt referenced in the URL. An in-progress attempt is resumed
 * through the start endpoint (which returns the existing attempt with its
 * saved answers); a finished attempt is returned with status submitted so the
 * caller can redirect to its result.
 */
export async function resumeQuizAttempt(quizId: string, attemptId: string): Promise<QuizAttempt> {
  const status = await apiRequest<BackendAttemptSummary>(
    `/v1/student/assessments/attempts/${encodeURIComponent(attemptId)}`,
    { auth: true },
  );
  if (status.status === "in_progress") {
    return startQuizAttempt(quizId);
  }
  return {
    id: status.id,
    quiz_id: status.quiz_id ?? quizId,
    started_at: status.started_at,
    submitted_at: status.submitted_at,
    status: "submitted",
    draft_revision: 0,
    questions: [],
    answers: {},
  };
}

export type SaveResult = { saved: boolean; draft_revision: number; server_now?: string };

/** Autosave. `revision` must increase with every change; the server keeps the newest. */
export async function saveQuizAnswers(
  attemptId: string,
  answers: AnswerMap,
  revision: number,
  options: { keepalive?: boolean } = {},
): Promise<SaveResult> {
  const quizId = await resolveAttemptQuizId(attemptId);
  const result = await apiRequest<{
    saved?: boolean;
    draft_revision?: number;
    server_now?: string;
  }>(`/v1/quizzes/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}`, {
    method: "PATCH",
    auth: true,
    body: { answers, revision },
    keepalive: options.keepalive,
  });
  return {
    saved: result.saved ?? true,
    draft_revision: result.draft_revision ?? revision,
    server_now: result.server_now,
  };
}

export type ReviewOption = {
  id: string;
  body: string;
  content_type?: string;
  image_url?: string;
  is_correct: boolean;
  position: number;
};

export type QuestionReview = {
  question_id: string;
  position: number;
  type: string;
  body: string;
  content_type?: string;
  image_url?: string;
  marks: number;
  options?: ReviewOption[];
  unit?: string;
  selected_option_ids: string[];
  text_answer?: string;
  answer_image_url?: string;
  answered: boolean;
  awarded?: number;
  status?: "correct" | "partial" | "incorrect" | "unanswered" | "pending" | "graded";
  correct_option_ids?: string[];
  accepted_answers?: string[];
  numeric_answer?: number;
  tolerance?: number;
  tolerance_type?: "absolute" | "percent";
  explanation?: string;
  feedback?: string;
  pending?: boolean;
};

export type QuizResult = {
  id: string;
  quiz_id: string;
  quiz_title?: string;
  status: string;
  grading_status: "auto" | "pending" | "graded";
  started_at?: string;
  submitted_at?: string | null;
  score_percent: number | null;
  score_points: number | null;
  max_points: number | null;
  passed: boolean | null;
  passing_score_percent?: number;
  time_taken_seconds?: number | null;
  points_awarded: number;
  answers_released: boolean;
  review_available: boolean;
  review: QuestionReview[];
};

type BackendResult = BackendAttemptSummary & {
  quiz_title?: string;
  passing_score_percent?: number;
  answers_released?: boolean;
  review_available?: boolean;
  review?: QuestionReview[];
};

function toResult(r: BackendResult): QuizResult {
  return {
    id: r.id,
    quiz_id: r.quiz_id,
    quiz_title: r.quiz_title,
    status: r.status,
    grading_status: r.grading_status ?? "auto",
    started_at: r.started_at,
    submitted_at: r.submitted_at,
    score_percent: r.score_percent ?? null,
    score_points: r.score_points ?? null,
    max_points: r.max_points ?? null,
    passed: r.passed ?? null,
    passing_score_percent: r.passing_score_percent,
    time_taken_seconds: r.time_taken_seconds ?? null,
    points_awarded: r.points_awarded ?? 0,
    answers_released: Boolean(r.answers_released),
    review_available: Boolean(r.review_available),
    review: (r.review ?? []).slice().sort((a, b) => a.position - b.position),
  };
}

export type SubmitResult = {
  attempt_id: string;
  quiz_id: string;
  status: string;
  grading_status: "auto" | "pending" | "graded";
  score_percent: number;
  score_points: number;
  max_points: number;
  passed: boolean;
};

/**
 * Submit is idempotent on the server: retrying after a network failure (or a
 * double click) returns the already-recorded result.
 */
export async function submitQuizAttempt(
  attemptId: string,
  answers: AnswerMap,
): Promise<SubmitResult> {
  const quizId = await resolveAttemptQuizId(attemptId);
  return apiRequest<SubmitResult>(
    `/v1/quizzes/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}/submit`,
    {
      method: "POST",
      auth: true,
      body: { answers: answersToCommands(answers) },
    },
  );
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Draft answer map -> submit payload. A bare UUID string is a single-choice option ID. */
export function answersToCommands(answers: AnswerMap) {
  return Object.entries(answers).map(([question_id, value]) => {
    if (Array.isArray(value)) return { question_id, selected_option_ids: value };
    if (typeof value === "string") {
      return UUID_RE.test(value)
        ? { question_id, selected_option_ids: [value] }
        : { question_id, selected_option_ids: [], text_answer: value };
    }
    return {
      question_id,
      selected_option_ids: [],
      text_answer: value?.text,
      image_url: value?.image_url,
    };
  });
}

/** Result of a submitted attempt (with review per the release policy). */
export async function getQuizAttempt(attemptId: string): Promise<QuizResult> {
  const quizId = await resolveAttemptQuizId(attemptId);
  return getQuizResult(quizId, attemptId);
}

export function getQuizResult(quizId: string, attemptId: string): Promise<QuizResult> {
  return apiRequest<BackendResult>(
    `/v1/student/assessments/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}`,
    { auth: true },
  ).then(toResult);
}

export type CheckResult = {
  question_id: string;
  status: string;
  correct: boolean;
  correct_option_ids?: string[];
  accepted_answers?: string[];
  numeric_answer?: number;
  explanation?: string;
};

/** Practice mode only: instant feedback on one answer. */
export async function checkQuizAnswer(attemptId: string, questionId: string, answer: AnswerValue) {
  const quizId = await resolveAttemptQuizId(attemptId);
  return apiRequest<CheckResult>(
    `/v1/quizzes/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}/check`,
    { method: "POST", auth: true, body: { question_id: questionId, answer } },
  );
}

/** Option answers are stored as one ID (single choice) or a list (multi). */
export function isAnswered(value: AnswerValue | undefined): boolean {
  if (value == null) return false;
  if (Array.isArray(value)) return value.length > 0;
  if (typeof value === "string") return value.trim().length > 0;
  return Boolean(value.text?.trim() || value.image_url);
}

/** Attach an image (JPEG/PNG/WebP, ≤ 5 MB) to a written answer. Returns a stable URL. */
export async function uploadAnswerImage(attemptId: string, file: File): Promise<string> {
  const quizId = await resolveAttemptQuizId(attemptId);
  const form = new FormData();
  form.append("file", file);
  const result = await apiRequest<{ file_url: string }>(
    `/v1/quizzes/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}/files`,
    { method: "POST", auth: true, body: form },
  );
  return resolveAssetUrl(result.file_url);
}

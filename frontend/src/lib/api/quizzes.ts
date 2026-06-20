import { apiRequest } from "./client";

export type QuestionType = "single_choice" | "multi_select" | "true_false" | "short_answer";

export type QuizQuestion = {
  id: string;
  type: QuestionType;
  prompt: string;
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  points: number;
  is_required?: boolean;
  options?: {
    id: string;
    text: string;
    content_type?: "text" | "image" | "text_image";
    image_url?: string;
    is_correct?: boolean;
  }[];
  // present only on review/result
  correct_option_ids?: string[];
  correct_text?: string;
  explanation?: string;
};

export type QuizSummary = {
  id: string;
  course_id: string;
  course_title?: string;
  title: string;
  description?: string;
  time_limit_minutes?: number | null;
  total_questions: number;
  total_points: number;
  passing_score: number;
  is_free?: boolean;
  is_published?: boolean;
  is_locked?: boolean;
  attempts_allowed?: number | null;
  attempts_used?: number;
  best_score?: number | null;
  status?: "not_started" | "in_progress" | "completed" | "passed" | "failed";
  available_from?: string | null;
  due_at?: string | null;
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
  latest_attempt?: {
    id: string;
    status: string;
    score_percent?: number | null;
    passed?: boolean | null;
  } | null;
};

const attemptQuizKey = (attemptId: string) => `inspire:quiz-attempt:${attemptId}`;

function rememberAttempt(attemptId: string, quizId: string) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(attemptQuizKey(attemptId), quizId);
  }
}

function readAttemptQuizId(attemptId: string) {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(attemptQuizKey(attemptId));
}

async function resolveAttemptQuizId(attemptId: string) {
  const localQuizId = readAttemptQuizId(attemptId);
  if (localQuizId) return localQuizId;
  const attempt = await apiRequest<{ quiz_id: string }>(
    `/v1/student/assessments/attempts/${encodeURIComponent(attemptId)}`,
    { auth: true },
  );
  rememberAttempt(attemptId, attempt.quiz_id);
  return attempt.quiz_id;
}

function toQuizSummary(q: BackendQuizSummary): QuizSummary {
  return {
    id: q.id,
    course_id: q.course_id,
    title: q.title,
    time_limit_minutes: q.time_limit_seconds ? Math.ceil(q.time_limit_seconds / 60) : null,
    total_questions: 0,
    total_points: 0,
    passing_score: q.passing_score_percent,
    is_free: q.is_free ?? true,
    is_published: q.is_published ?? true,
    is_locked: q.is_locked ?? false,
    attempts_allowed: q.max_attempts || null,
    attempts_used: q.attempts_used ?? 0,
    status:
      q.latest_attempt?.status === "submitted"
        ? q.latest_attempt.passed
          ? "passed"
          : "failed"
        : q.latest_attempt?.status === "in_progress"
          ? "in_progress"
          : "not_started",
  };
}

function toQuestion(q: {
  id: string;
  body: string;
  type: string;
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  marks?: number;
  is_required?: boolean;
  options?: Array<{
    id: string;
    body: string;
    content_type?: "text" | "image" | "text_image";
    image_url?: string;
    is_correct?: boolean;
  }>;
  correct_option_ids?: string[];
  explanation?: string;
}): QuizQuestion {
  return {
    id: q.id,
    type:
      q.type === "multiple" ? "multi_select" : q.type === "single" ? "single_choice" : "true_false",
    prompt: q.body,
    content_type: q.content_type ?? (q.image_url ? (q.body ? "text_image" : "image") : "text"),
    image_url: q.image_url,
    points: q.marks ?? 1,
    is_required: q.is_required ?? true,
    options: q.options?.map((option) => ({
      id: option.id,
      text: option.body,
      content_type:
        option.content_type ?? (option.image_url ? (option.body ? "text_image" : "image") : "text"),
      image_url: option.image_url,
      is_correct: option.is_correct,
    })),
    correct_option_ids: q.correct_option_ids,
    explanation: q.explanation,
  };
}

export type Quiz = QuizSummary & {
  instructions?: string;
};

export type QuizAttempt = {
  id: string;
  quiz_id: string;
  started_at: string;
  expires_at?: string | null;
  submitted_at?: string | null;
  status: "in_progress" | "submitted" | "graded";
  current_question_index?: number;
  questions: QuizQuestion[];
  answers: Record<string, string[] | string>; // questionId → option ids or text
};

export type QuizResult = QuizAttempt & {
  score: number;
  max_score: number;
  percentage: number;
  passed: boolean;
  feedback?: string;
  per_question: { question_id: string; awarded: number; max: number; correct: boolean }[];
};

export function listCourseQuizzes(courseId: string) {
  return listMyQuizzes().then((result) => ({
    data: result.data.filter((quiz) => quiz.course_id === courseId),
  }));
}

export function listMyQuizzes(params: { status?: string; page?: number; limit?: number } = {}) {
  return apiRequest<{ quizzes: BackendQuizSummary[] }>("/v1/student/assessments", {
    auth: true,
    query: params,
  }).then((result) => ({
    data: result.quizzes.map(toQuizSummary),
    meta: {
      page: params.page ?? 1,
      limit: params.limit ?? result.quizzes.length,
      total: result.quizzes.length,
      total_pages: 1,
    },
  }));
}

export function getQuiz(quizId: string) {
  return apiRequest<BackendQuizSummary & { show_answers_after_submission?: boolean }>(
    `/v1/student/assessments/${encodeURIComponent(quizId)}`,
    { auth: true },
  ).then((q) => ({
    ...toQuizSummary(q),
    instructions: "Answer all questions before submitting.",
  }));
}

export function startQuizAttempt(quizId: string) {
  return apiRequest<{
    id: string;
    quiz_id: string;
    started_at: string;
    expires_at?: string | null;
    questions: Array<{
      id: string;
      body: string;
      type: string;
      content_type?: "text" | "image" | "text_image";
      image_url?: string;
      marks?: number;
      is_required?: boolean;
      options?: Array<{
        id: string;
        body: string;
        content_type?: "text" | "image" | "text_image";
        image_url?: string;
      }>;
    }>;
  }>(`/v1/quizzes/${encodeURIComponent(quizId)}/attempts`, {
    method: "POST",
    auth: true,
  }).then((attempt) => {
    rememberAttempt(attempt.id, quizId);
    return {
      id: attempt.id,
      quiz_id: attempt.quiz_id,
      started_at: attempt.started_at,
      expires_at: attempt.expires_at,
      status: "in_progress",
      questions: attempt.questions.map(toQuestion),
      answers: (attempt as { answers?: QuizAttempt["answers"] }).answers ?? {},
    };
  });
}

export async function getQuizAttempt(attemptId: string) {
  const quizId = await resolveAttemptQuizId(attemptId);
  return apiRequest<{
    id: string;
    quiz_id?: string;
    status: string;
    started_at: string;
    submitted_at?: string;
    score_percent?: number;
    passed?: boolean;
    points_awarded?: number;
  }>(
    `/v1/student/assessments/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}`,
    {
      auth: true,
    },
  ).then((result) => ({
    id: result.id,
    quiz_id: result.quiz_id ?? quizId,
    started_at: result.started_at,
    submitted_at: result.submitted_at,
    status: result.status === "submitted" ? "graded" : "in_progress",
    questions: [],
    answers: {},
    score: result.score_percent ?? 0,
    max_score: 100,
    percentage: result.score_percent ?? 0,
    passed: Boolean(result.passed),
    feedback: result.passed ? "Passed" : "Review the material and try again.",
    per_question: [],
  }));
}

export async function saveQuizAnswers(attemptId: string, answers: QuizAttempt["answers"]) {
  const quizId = await resolveAttemptQuizId(attemptId);
  return apiRequest<{ answers?: QuizAttempt["answers"] }>(
    `/v1/quizzes/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}`,
    {
      method: "PATCH",
      auth: true,
      body: { answers },
    },
  ).then(() => ({ saved_at: new Date().toISOString() }));
}

export async function submitQuizAttempt(attemptId: string, answers: QuizAttempt["answers"]) {
  const quizId = await resolveAttemptQuizId(attemptId);
  return apiRequest<{
    score_percent: number;
    passed: boolean;
    points_awarded: number;
    question_results?: Array<{
      question_id: string;
      is_correct: boolean;
      selected_options: string[];
      correct_options: string[];
      explanation?: string;
    }>;
  }>(`/v1/quizzes/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}/submit`, {
    method: "POST",
    auth: true,
    body: {
      answers: Object.entries(answers).map(([question_id, selected_option_ids]) => ({
        question_id,
        selected_option_ids: Array.isArray(selected_option_ids)
          ? selected_option_ids
          : [selected_option_ids],
      })),
    },
  }).then((result) => ({
    id: attemptId,
    quiz_id: quizId,
    started_at: new Date().toISOString(),
    submitted_at: new Date().toISOString(),
    status: "graded",
    questions: [],
    answers,
    score: result.score_percent,
    max_score: 100,
    percentage: result.score_percent,
    passed: result.passed,
    feedback: result.passed ? "Passed" : "Review the material and try again.",
    per_question:
      result.question_results?.map((q) => ({
        question_id: q.question_id,
        awarded: q.is_correct ? 1 : 0,
        max: 1,
        correct: q.is_correct,
      })) ?? [],
  }));
}

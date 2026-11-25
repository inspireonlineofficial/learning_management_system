import { apiRequest } from "./client";
import { listMyTaughtCourses } from "./teacher";
import { toFrontendType, type QuestionType, type QuizMode, type QuizQuestion } from "./quizzes";

// Teacher-side assessment API: authoring, monitoring, grading, analytics.

export type CalculatorPolicy = "none" | "basic" | "scientific" | "graphing";
export type QuizLifecycle = "draft" | "scheduled" | "open" | "closed";

export type TeacherQuizSettings = {
  course_id?: string;
  lesson_id?: string | null;
  title: string;
  description?: string;
  instructions?: string;
  mode?: QuizMode;
  /** 0 / null = untimed. */
  time_limit_minutes?: number | null;
  passing_score?: number;
  /** null = unlimited. */
  attempts_allowed?: number | null;
  is_free?: boolean;
  is_published?: boolean;
  shuffle_questions?: boolean;
  shuffle_options?: boolean;
  partial_credit?: boolean;
  show_answers_after_submission?: boolean;
  available_from?: string | null;
  available_until?: string | null;
  calculator_policy?: CalculatorPolicy;
  reference_material?: string;
};

/** Kept for existing callers (course editor, quiz-builder/new). */
export type TeacherQuizInput = TeacherQuizSettings;

export type TeacherQuestion = QuizQuestion & {
  hint?: string;
  correct_option_ids: string[];
  /** Short answer: accepted answers. */
  accepted_answers: string[];
  case_sensitive?: boolean;
  numeric_answer?: number | null;
  tolerance?: number;
  tolerance_type?: "absolute" | "percent";
};

export type TeacherQuiz = Required<Pick<TeacherQuizSettings, "title">> & {
  id: string;
  course_id: string;
  course_title?: string;
  lesson_id?: string | null;
  description: string;
  instructions: string;
  mode: QuizMode;
  time_limit_minutes: number | null;
  passing_score: number;
  attempts_allowed: number | null;
  is_free: boolean;
  is_published: boolean;
  shuffle_questions: boolean;
  shuffle_options: boolean;
  partial_credit: boolean;
  show_answers_after_submission: boolean;
  available_from: string | null;
  available_until: string | null;
  calculator_policy: CalculatorPolicy;
  reference_material: string;
  lifecycle: QuizLifecycle;
  total_questions: number;
  total_points: number;
  attempt_count: number;
  submitted_count: number;
  in_progress_count: number;
  pending_grading_count: number;
  questions: TeacherQuestion[];
};

/** Fields a question editor fills in. */
export type TeacherQuestionInput = {
  id?: string;
  type: QuestionType;
  prompt: string;
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  points?: number;
  is_required?: boolean;
  options?: {
    id?: string;
    text: string;
    content_type?: "text" | "image" | "text_image";
    image_url?: string;
    is_correct?: boolean;
  }[];
  /** Short answer: one accepted answer per line. */
  correct_text?: string;
  case_sensitive?: boolean;
  numeric_answer?: number | null;
  tolerance?: number;
  tolerance_type?: "absolute" | "percent";
  unit?: string;
  max_words?: number;
  explanation?: string;
  hint?: string;
};

type BackendOption = {
  id: string;
  body: string;
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  is_correct?: boolean;
  position: number;
};

type BackendTeacherQuestion = {
  id: string;
  body: string;
  type: string;
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  marks?: number;
  is_required?: boolean;
  position: number;
  explanation?: string;
  hint?: string;
  settings?: {
    numeric_answer?: number;
    tolerance?: number;
    tolerance_type?: "absolute" | "percent";
    unit?: string;
    max_words?: number;
    case_sensitive?: boolean;
  };
  correct_option_ids?: string[];
  options?: BackendOption[];
};

type BackendTeacherQuiz = {
  id: string;
  course_id: string;
  lesson_id?: string | null;
  title: string;
  description?: string;
  instructions?: string;
  mode?: QuizMode;
  time_limit_seconds: number;
  max_attempts: number;
  passing_score_percent: number;
  is_free?: boolean;
  is_published?: boolean;
  shuffle_questions?: boolean;
  shuffle_options?: boolean;
  partial_credit?: boolean;
  show_answers_after_submission?: boolean;
  available_from?: string | null;
  available_until?: string | null;
  calculator_policy?: CalculatorPolicy;
  reference_material?: string;
  lifecycle?: QuizLifecycle;
  question_count?: number;
  total_marks?: number;
  attempt_count?: number;
  submitted_count?: number;
  in_progress_count?: number;
  pending_grading_count?: number;
  questions?: BackendTeacherQuestion[];
};

function toQuestion(q: BackendTeacherQuestion): TeacherQuestion {
  const type = toFrontendType(q.type);
  const options = (q.options ?? []).slice().sort((a, b) => a.position - b.position);
  return {
    id: q.id,
    type,
    prompt: q.body,
    position: q.position,
    content_type: q.content_type ?? (q.image_url ? (q.body ? "text_image" : "image") : "text"),
    image_url: q.image_url || undefined,
    points: q.marks ?? 1,
    is_required: q.is_required ?? true,
    explanation: q.explanation || undefined,
    hint: q.hint || undefined,
    options:
      type === "short_answer"
        ? []
        : options.map((o) => ({
            id: o.id,
            text: o.body,
            content_type: o.content_type,
            image_url: o.image_url || undefined,
            is_correct: o.is_correct,
          })),
    correct_option_ids:
      q.correct_option_ids ?? options.filter((o) => o.is_correct).map((o) => o.id),
    accepted_answers: type === "short_answer" ? options.map((o) => o.body) : [],
    correct_text: type === "short_answer" ? options.map((o) => o.body).join("\n") : undefined,
    case_sensitive: q.settings?.case_sensitive,
    numeric_answer: q.settings?.numeric_answer ?? null,
    tolerance: q.settings?.tolerance ?? 0,
    tolerance_type: q.settings?.tolerance_type ?? "absolute",
    unit: q.settings?.unit || undefined,
    max_words: q.settings?.max_words || undefined,
  };
}

function toQuiz(quiz: BackendTeacherQuiz, courseTitle?: string): TeacherQuiz {
  const questions = (quiz.questions ?? [])
    .map(toQuestion)
    .sort((a, b) => (a.position ?? 0) - (b.position ?? 0));
  return {
    id: quiz.id,
    course_id: quiz.course_id,
    course_title: courseTitle,
    lesson_id: quiz.lesson_id ?? null,
    title: quiz.title,
    description: quiz.description ?? "",
    instructions: quiz.instructions ?? "",
    mode: quiz.mode === "practice" ? "practice" : "exam",
    time_limit_minutes: quiz.time_limit_seconds ? Math.ceil(quiz.time_limit_seconds / 60) : null,
    passing_score: quiz.passing_score_percent,
    attempts_allowed: quiz.max_attempts || null,
    is_free: quiz.is_free ?? true,
    is_published: quiz.is_published ?? false,
    shuffle_questions: Boolean(quiz.shuffle_questions),
    shuffle_options: Boolean(quiz.shuffle_options),
    partial_credit: Boolean(quiz.partial_credit),
    show_answers_after_submission: Boolean(quiz.show_answers_after_submission),
    available_from: quiz.available_from ?? null,
    available_until: quiz.available_until ?? null,
    calculator_policy: quiz.calculator_policy ?? "none",
    reference_material: quiz.reference_material ?? "",
    lifecycle: quiz.lifecycle ?? (quiz.is_published ? "open" : "draft"),
    total_questions: quiz.question_count ?? questions.length,
    total_points: quiz.total_marks ?? questions.reduce((sum, q) => sum + q.points, 0),
    attempt_count: quiz.attempt_count ?? 0,
    submitted_count: quiz.submitted_count ?? 0,
    in_progress_count: quiz.in_progress_count ?? 0,
    pending_grading_count: quiz.pending_grading_count ?? 0,
    questions,
  };
}

const backendType = (type: QuestionType) =>
  type === "multi_select" ? "multiple" : type === "single_choice" ? "single" : type;

/** Question payload accepted by both the quiz and the single-question endpoints. */
export function questionPayload(input: TeacherQuestionInput, position?: number) {
  const settings =
    input.type === "numeric"
      ? {
          numeric_answer: input.numeric_answer ?? undefined,
          tolerance: input.tolerance ?? 0,
          tolerance_type: input.tolerance_type ?? "absolute",
          unit: input.unit ?? "",
        }
      : input.type === "long_answer"
        ? { max_words: input.max_words ?? 0 }
        : input.type === "short_answer"
          ? { case_sensitive: Boolean(input.case_sensitive) }
          : undefined;
  const usesOptions = ["single_choice", "multi_select", "true_false"].includes(input.type);
  return {
    id: input.id,
    type: backendType(input.type),
    body: input.prompt,
    content_type: input.content_type,
    image_url: input.image_url ?? "",
    marks: input.points ?? 1,
    is_required: input.is_required ?? true,
    position,
    explanation: input.explanation ?? "",
    hint: input.hint ?? "",
    settings,
    correct_text: input.type === "short_answer" ? (input.correct_text ?? "") : undefined,
    options: usesOptions
      ? (input.options ?? []).map((o, i) => ({
          id: o.id,
          body: o.text,
          content_type: o.content_type,
          image_url: o.image_url ?? "",
          is_correct: Boolean(o.is_correct),
          position: i + 1,
        }))
      : [],
  };
}

/** Convert a loaded question back into editor input (keeps IDs). */
export function questionToInput(q: TeacherQuestion): TeacherQuestionInput {
  return {
    id: q.id,
    type: q.type,
    prompt: q.prompt,
    content_type: q.content_type,
    image_url: q.image_url,
    points: q.points,
    is_required: q.is_required,
    options: (q.options ?? []).map((o) => ({
      id: o.id,
      text: o.text,
      content_type: o.content_type,
      image_url: o.image_url,
      is_correct: q.correct_option_ids.includes(o.id) || Boolean(o.is_correct),
    })),
    correct_text: q.accepted_answers.join("\n"),
    case_sensitive: q.case_sensitive,
    numeric_answer: q.numeric_answer,
    tolerance: q.tolerance,
    tolerance_type: q.tolerance_type,
    unit: q.unit,
    max_words: q.max_words,
    explanation: q.explanation,
    hint: q.hint,
  };
}

function settingsPayload(s: TeacherQuizSettings) {
  return {
    lesson_id: s.lesson_id ?? undefined,
    title: s.title,
    description: s.description ?? "",
    instructions: s.instructions ?? "",
    mode: s.mode ?? "exam",
    time_limit_seconds: s.time_limit_minutes ? Math.round(s.time_limit_minutes * 60) : 0,
    max_attempts: s.attempts_allowed ?? 0,
    passing_score_percent: s.passing_score ?? 60,
    is_free: s.is_free ?? true,
    is_published: s.is_published ?? false,
    shuffle_questions: Boolean(s.shuffle_questions),
    shuffle_options: Boolean(s.shuffle_options),
    partial_credit: Boolean(s.partial_credit),
    show_answers_after_submission: Boolean(s.show_answers_after_submission),
    available_from: s.available_from || null,
    available_until: s.available_until || null,
    calculator_policy: s.calculator_policy ?? "none",
    reference_material: s.reference_material ?? "",
  };
}

export function listTeacherQuizzes(
  params: { course_id?: string; page?: number; limit?: number } = {},
) {
  const loadForCourse = (courseId: string, title?: string) =>
    apiRequest<BackendTeacherQuiz[]>(
      `/v1/teacher/courses/${encodeURIComponent(courseId)}/quizzes`,
      {
        auth: true,
      },
    ).then((items) => (items ?? []).map((q) => toQuiz(q, title)));

  const source = params.course_id
    ? loadForCourse(params.course_id)
    : listMyTaughtCourses({ limit: 100 }).then((courses) =>
        Promise.all(courses.data.map((course) => loadForCourse(course.id, course.title))).then(
          (groups) => groups.flat(),
        ),
      );
  return source.then((items) => ({ data: items, meta: { total: items.length } }));
}

export function getTeacherQuiz(quizId: string): Promise<TeacherQuiz> {
  return apiRequest<BackendTeacherQuiz>(`/v1/teacher/quizzes/${encodeURIComponent(quizId)}`, {
    auth: true,
  }).then((q) => toQuiz(q));
}

/** Create a quiz. Without questions it is always saved as a draft. */
export function createQuiz(input: TeacherQuizSettings & { questions?: TeacherQuestionInput[] }) {
  if (!input.course_id) return Promise.reject(new Error("Select a course before creating a quiz."));
  const questions = input.questions ?? [];
  return apiRequest<BackendTeacherQuiz>(
    `/v1/teacher/courses/${encodeURIComponent(input.course_id)}/quizzes`,
    {
      method: "POST",
      auth: true,
      body: {
        ...settingsPayload({
          ...input,
          is_published: questions.length > 0 && (input.is_published ?? false),
        }),
        questions: questions.map((q, i) => questionPayload(q, i + 1)),
      },
    },
  ).then((q) => toQuiz(q));
}

/**
 * Save quiz settings. The current questions are sent with their IDs, so the
 * server updates them in place: in-progress attempts keep their answers and
 * submitted attempts keep their snapshot.
 */
export function updateQuiz(
  quizId: string,
  settings: TeacherQuizSettings,
  questions: TeacherQuestion[],
): Promise<TeacherQuiz> {
  return apiRequest<BackendTeacherQuiz>(`/v1/teacher/quizzes/${encodeURIComponent(quizId)}`, {
    method: "PATCH",
    auth: true,
    body: {
      ...settingsPayload(settings),
      questions: questions.map((q, i) => questionPayload(questionToInput(q), i + 1)),
    },
  }).then((q) => toQuiz(q));
}

export function deleteQuiz(quizId: string) {
  return apiRequest<{ ok: true }>(`/v1/teacher/quizzes/${encodeURIComponent(quizId)}`, {
    method: "DELETE",
    auth: true,
  });
}

export function duplicateQuiz(quizId: string) {
  return apiRequest<BackendTeacherQuiz>(
    `/v1/teacher/quizzes/${encodeURIComponent(quizId)}/duplicate`,
    {
      method: "POST",
      auth: true,
    },
  ).then((q) => toQuiz(q));
}

export function createQuestion(quizId: string, input: TeacherQuestionInput) {
  return apiRequest<BackendTeacherQuestion>(
    `/v1/teacher/quizzes/${encodeURIComponent(quizId)}/questions`,
    {
      method: "POST",
      auth: true,
      body: questionPayload(input),
    },
  ).then(toQuestion);
}

export function updateQuestion(questionId: string, input: TeacherQuestionInput, position?: number) {
  return apiRequest<BackendTeacherQuestion>(
    `/v1/teacher/questions/${encodeURIComponent(questionId)}`,
    {
      method: "PATCH",
      auth: true,
      body: questionPayload(input, position),
    },
  ).then(toQuestion);
}

export function deleteQuestion(questionId: string) {
  return apiRequest<{ ok: true }>(`/v1/teacher/questions/${encodeURIComponent(questionId)}`, {
    method: "DELETE",
    auth: true,
  });
}

// ---------------------------------------------------------------------------
// Monitoring, grading, analytics

export type AttemptRow = {
  attempt_id: string;
  student_id: string;
  student_name: string;
  student_email: string;
  attempt_number: number;
  status: "in_progress" | "submitted" | "auto_submitted";
  grading_status: "auto" | "pending" | "graded";
  started_at: string;
  submitted_at?: string | null;
  time_taken_seconds?: number | null;
  score_percent?: number | null;
  score_points?: number | null;
  max_points?: number | null;
  passed?: boolean | null;
};

export type AttemptSummary = {
  enrolled_students: number;
  students_attempted: number;
  students_not_started: number;
  total_attempts: number;
  in_progress: number;
  submitted: number;
  pending_grading: number;
  average_score?: number | null;
  highest_score?: number | null;
  lowest_score?: number | null;
  pass_rate?: number | null;
  graded_attempts: number;
};

export type AttemptList = {
  quiz: BackendTeacherQuiz;
  summary: AttemptSummary;
  attempts: AttemptRow[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export function listQuizAttempts(
  quizId: string,
  params: {
    status?: string;
    q?: string;
    sort?: string;
    order?: string;
    page?: number;
    limit?: number;
  } = {},
) {
  return apiRequest<AttemptList>(`/v1/teacher/quizzes/${encodeURIComponent(quizId)}/attempts`, {
    auth: true,
    query: params,
  });
}

export type QuestionOutcome = {
  question_id: string;
  position: number;
  type: string;
  body: string;
  image_url?: string;
  marks: number;
  options?: {
    id: string;
    body: string;
    image_url?: string;
    is_correct: boolean;
    position: number;
  }[];
  correct_option_ids: string[];
  accepted_answers?: string[];
  numeric_answer?: number;
  tolerance?: number;
  tolerance_type?: string;
  unit?: string;
  explanation?: string;
  selected_option_ids: string[];
  text_answer?: string;
  answer_image_url?: string;
  answered: boolean;
  awarded: number;
  status: "correct" | "partial" | "incorrect" | "unanswered" | "pending" | "graded";
  feedback?: string;
  graded_at?: string;
};

export type TeacherAttemptDetail = AttemptRow & {
  quiz_title: string;
  review_available: boolean;
  results: QuestionOutcome[];
  draft_answers?: Record<string, unknown>;
};

export function getTeacherAttempt(quizId: string, attemptId: string) {
  return apiRequest<TeacherAttemptDetail>(
    `/v1/teacher/quizzes/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}`,
    { auth: true },
  );
}

export function gradeAttempt(
  quizId: string,
  attemptId: string,
  grades: { question_id: string; awarded: number; feedback: string }[],
) {
  return apiRequest<TeacherAttemptDetail>(
    `/v1/teacher/quizzes/${encodeURIComponent(quizId)}/attempts/${encodeURIComponent(attemptId)}/grade`,
    { method: "POST", auth: true, body: { grades } },
  );
}

export type QuizAnalytics = {
  quiz_id: string;
  submitted_attempts: number;
  graded_attempts: number;
  small_sample: boolean;
  average_score?: number | null;
  median_score?: number | null;
  average_time_seconds?: number | null;
  score_distribution: { from: number; to: number; count: number }[];
  questions: {
    question_id: string;
    position: number;
    type: string;
    body: string;
    marks: number;
    responses: number;
    correct: number;
    partial: number;
    incorrect: number;
    unanswered: number;
    pending: number;
    average_awarded: number;
    accuracy_percent?: number | null;
    option_counts?: { option_id: string; body: string; is_correct: boolean; count: number }[];
    common_wrong_answers?: { answer: string; count: number }[];
  }[];
};

export function getQuizAnalytics(quizId: string) {
  return apiRequest<QuizAnalytics>(`/v1/teacher/quizzes/${encodeURIComponent(quizId)}/analytics`, {
    auth: true,
  });
}

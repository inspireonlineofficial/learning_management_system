import { apiRequest } from "./client";
import { listMyTaughtCourses } from "./teacher";
import type { QuestionType, Quiz, QuizQuestion } from "./quizzes";

export type TeacherQuizInput = {
  course_id?: string;
  lesson_id?: string;
  title: string;
  description?: string;
  instructions?: string;
  time_limit_minutes?: number | null;
  passing_score?: number;
  attempts_allowed?: number | null;
  is_free?: boolean;
  is_published?: boolean;
};

export type TeacherQuestionInput = {
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
  correct_text?: string;
  explanation?: string;
};

type BackendTeacherQuiz = {
  id: string;
  course_id: string;
  lesson_id?: string | null;
  title: string;
  time_limit_seconds: number;
  max_attempts: number;
  passing_score_percent: number;
  is_free?: boolean;
  is_published?: boolean;
  shuffle_questions?: boolean;
  show_answers_after_submission?: boolean;
  questions?: BackendTeacherQuestion[];
};

type BackendTeacherQuestion = {
  id: string;
  body: string;
  type: "single" | "multiple" | "true_false";
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  marks?: number;
  is_required?: boolean;
  position: number;
  explanation?: string;
  correct_option_ids?: string[];
  options?: Array<{
    id: string;
    body: string;
    content_type?: "text" | "image" | "text_image";
    image_url?: string;
    is_correct?: boolean;
    position: number;
  }>;
};

export function listTeacherQuizzes(
  params: { course_id?: string; page?: number; limit?: number } = {},
) {
  const loadForCourse = (courseId: string) =>
    apiRequest<BackendTeacherQuiz[]>(
      `/v1/teacher/courses/${encodeURIComponent(courseId)}/quizzes`,
      { auth: true },
    );

  const source = params.course_id
    ? loadForCourse(params.course_id)
    : listMyTaughtCourses({ limit: 100 }).then((courses) =>
        Promise.all(courses.data.map((course) => loadForCourse(course.id))).then((groups) =>
          groups.flat(),
        ),
      );

  return source.then((items) => ({
    data: items.map(toQuiz),
    meta: { total: items.length },
  }));
}

export function getTeacherQuiz(quizId: string) {
  return apiRequest<BackendTeacherQuiz>(`/v1/teacher/quizzes/${encodeURIComponent(quizId)}`, {
    auth: true,
  }).then(toQuiz);
}

export function createQuiz(input: TeacherQuizInput) {
  if (!input.course_id) return Promise.reject(new Error("Select a course before creating a quiz."));
  return apiRequest<Quiz>(`/v1/teacher/courses/${encodeURIComponent(input.course_id)}/quizzes`, {
    method: "POST",
    auth: true,
    body: {
      lesson_id: input.lesson_id,
      title: input.title,
      time_limit_seconds: (input.time_limit_minutes ?? 30) * 60,
      max_attempts: input.attempts_allowed ?? 1,
      passing_score_percent: input.passing_score ?? 60,
      is_free: input.is_free ?? true,
      is_published: input.is_published ?? true,
      shuffle_questions: false,
      show_answers_after_submission: true,
      questions: toQuizPayload(input).questions,
    },
  });
}

export function updateQuiz(quizId: string, input: Partial<TeacherQuizInput>) {
  return getTeacherQuiz(quizId).then((current) =>
    apiRequest<BackendTeacherQuiz>(`/v1/teacher/quizzes/${encodeURIComponent(quizId)}`, {
      method: "PATCH",
      auth: true,
      body: toQuizPayload({
        course_id: input.course_id ?? current.course_id,
        lesson_id: input.lesson_id ?? undefined,
        title: input.title ?? current.title,
        time_limit_minutes: input.time_limit_minutes ?? current.time_limit_minutes,
        passing_score: input.passing_score ?? current.passing_score,
        attempts_allowed: input.attempts_allowed ?? current.attempts_allowed,
        is_free: input.is_free ?? current.is_free,
        is_published: input.is_published ?? current.is_published,
        questions: current.questions ?? [],
      }),
    }).then(toQuiz),
  );
}

export function deleteQuiz(quizId: string) {
  return apiRequest<{ ok: true }>(`/v1/teacher/quizzes/${encodeURIComponent(quizId)}`, {
    method: "DELETE",
    auth: true,
  });
}

export function createQuestion(quizId: string, input: TeacherQuestionInput) {
  return apiRequest<BackendTeacherQuestion>(
    `/v1/teacher/quizzes/${encodeURIComponent(quizId)}/questions`,
    {
      method: "POST",
      auth: true,
      body: toQuestionPayload(input),
    },
  ).then(toQuestion);
}

export function updateQuestion(questionId: string, input: Partial<TeacherQuestionInput>) {
  return apiRequest<BackendTeacherQuestion>(
    `/v1/teacher/questions/${encodeURIComponent(questionId)}`,
    {
      method: "PATCH",
      auth: true,
      body: toQuestionPayload(input),
    },
  ).then(toQuestion);
}

export function deleteQuestion(questionId: string) {
  return apiRequest<{ ok: true }>(`/v1/teacher/questions/${encodeURIComponent(questionId)}`, {
    method: "DELETE",
    auth: true,
  });
}

function toQuiz(quiz: BackendTeacherQuiz): Quiz {
  const questions = quiz.questions?.map(toQuestion) ?? [];
  return {
    id: quiz.id,
    course_id: quiz.course_id,
    title: quiz.title,
    time_limit_minutes: quiz.time_limit_seconds ? Math.ceil(quiz.time_limit_seconds / 60) : null,
    total_questions: questions.length,
    total_points: questions.length,
    passing_score: quiz.passing_score_percent,
    is_free: quiz.is_free ?? true,
    is_published: quiz.is_published ?? true,
    attempts_allowed: quiz.max_attempts || null,
    questions,
  } as Quiz & { questions: QuizQuestion[] };
}

function toQuestion(question: BackendTeacherQuestion): QuizQuestion {
  const type =
    question.type === "multiple"
      ? "multi_select"
      : question.type === "single"
        ? "single_choice"
        : question.type === "short_answer"
          ? "short_answer"
          : "true_false";
  const options = question.options?.map((option) => ({
    id: option.id,
    text: option.body,
    content_type:
      option.content_type ?? (option.image_url ? (option.body ? "text_image" : "image") : "text"),
    image_url: option.image_url,
    is_correct: option.is_correct,
  }));

  return {
    id: question.id,
    type,
    prompt: question.body,
    content_type:
      question.content_type ??
      (question.image_url ? (question.body ? "text_image" : "image") : "text"),
    image_url: question.image_url,
    points: question.marks ?? 1,
    is_required: question.is_required ?? true,
    options,
    correct_option_ids: question.correct_option_ids,
    correct_text:
      type === "short_answer" ? options?.find((option) => option.is_correct)?.text : undefined,
    explanation: question.explanation,
  };
}

function toQuizPayload(input: TeacherQuizInput & { questions?: QuizQuestion[] }) {
  return {
    lesson_id: input.lesson_id,
    title: input.title,
    time_limit_seconds: Math.max(1, input.time_limit_minutes ?? 30) * 60,
    max_attempts: input.attempts_allowed ?? 1,
    passing_score_percent: input.passing_score ?? 60,
    is_free: input.is_free ?? true,
    is_published: input.is_published ?? true,
    shuffle_questions: false,
    show_answers_after_submission: true,
    questions:
      input.questions && input.questions.length > 0
        ? input.questions.map((question, index) => ({
            body: question.prompt,
            content_type: question.content_type,
            image_url: question.image_url,
            marks: question.points ?? 1,
            is_required: question.is_required ?? true,
            type:
              question.type === "multi_select"
                ? "multiple"
                : question.type === "single_choice"
                  ? "single"
                  : question.type === "short_answer"
                    ? "short_answer"
                    : "true_false",
            position: index + 1,
            explanation: question.explanation ?? "",
            options:
              question.type === "short_answer"
                ? [
                    {
                      body: question.correct_text ?? "",
                      content_type: "text",
                      is_correct: true,
                      position: 1,
                    },
                  ]
                : question.options && question.options.length > 0
                  ? question.options.map((option, optionIndex) => ({
                      body: option.text,
                      content_type: option.content_type,
                      image_url: option.image_url,
                      is_correct:
                        question.correct_option_ids && question.correct_option_ids.length > 0
                          ? question.correct_option_ids.includes(option.id)
                          : option.is_correct || optionIndex === 0,
                      position: optionIndex + 1,
                    }))
                  : [
                      { body: "True", is_correct: true, position: 1 },
                      { body: "False", is_correct: false, position: 2 },
                    ],
          }))
        : [
            {
              body: "Draft question",
              content_type: "text",
              marks: 1,
              is_required: true,
              type: "true_false",
              position: 1,
              explanation: "",
              options: [
                { body: "True", is_correct: true, position: 1 },
                { body: "False", is_correct: false, position: 2 },
              ],
            },
          ],
  };
}

function toQuestionPayload(input: Partial<TeacherQuestionInput>) {
  const options =
    input.type === "short_answer"
      ? [
          {
            text: input.correct_text ?? "",
            content_type: "text",
            is_correct: true,
            position: 1,
          },
        ]
      : input.options && input.options.length > 0
        ? input.options.map((option, index) => ({
            text: option.text,
            content_type: option.content_type,
            image_url: option.image_url,
            is_correct: option.is_correct ?? index === 0,
            position: index + 1,
          }))
        : [
            { text: "True", is_correct: true, position: 1 },
            { text: "False", is_correct: false, position: 2 },
          ];
  return {
    type: input.type ?? "true_false",
    prompt: input.prompt ?? "Draft question",
    content_type: input.content_type,
    image_url: input.image_url,
    marks: input.points ?? 1,
    is_required: input.is_required ?? true,
    position: 1,
    explanation: input.explanation ?? "",
    correct_text: input.correct_text,
    options,
  };
}

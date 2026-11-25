import type { QuizQuestion } from "@/lib/api/quizzes";

export const QUESTION_TYPE_LABEL: Record<QuizQuestion["type"], string> = {
  single_choice: "Single choice",
  multi_select: "Multiple select",
  true_false: "True / false",
  short_answer: "Short answer",
  numeric: "Numerical answer",
  long_answer: "Written answer",
};

export function countWords(text: string): number {
  const trimmed = text.trim();
  return trimmed ? trimmed.split(/\s+/).length : 0;
}

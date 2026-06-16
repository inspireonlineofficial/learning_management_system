import { listLiveSessions } from "./live";
import { listMyAssignments } from "./assignments";
import { listMyQuizzes } from "./quizzes";

export type CalendarEvent = {
  id: string;
  type: "live_session" | "assignment_due" | "quiz_due";
  title: string;
  starts_at: string;
  ends_at?: string;
  course_id?: string;
  course_title?: string;
  url?: string;
  is_today?: boolean;
};

export const getCalendar = async (range: { from: string; to: string }) => {
  const [sessions, assignments, quizzes] = await Promise.all([
    listLiveSessions({
      from: range.from,
      to: range.to,
      limit: 100,
    }),
    listMyAssignments({ limit: 100 }),
    listMyQuizzes({ limit: 100 }),
  ]);

  const from = new Date(range.from).getTime();
  const to = new Date(range.to).getTime();
  const inRange = (value?: string | null) => {
    if (!value) return false;
    const time = new Date(value).getTime();
    return Number.isFinite(time) && time >= from && time <= to;
  };

  return {
    items: [
      ...sessions.data.map((session) => ({
        id: session.id,
        type: "live_session" as const,
        title: session.title,
        starts_at: session.starts_at,
        ends_at: session.ends_at,
        course_id: session.course_id,
        is_today: Boolean((session as { is_today?: boolean }).is_today),
        url: `/student/live-classes/${session.id}`,
      })),
      ...assignments.data
        .filter((assignment) => inRange(assignment.due_at))
        .map((assignment) => ({
          id: assignment.id,
          type: "assignment_due" as const,
          title: assignment.title,
          starts_at: assignment.due_at!,
          course_id: assignment.course_id,
          course_title: assignment.course_title,
          url: `/student/assignments/${assignment.id}`,
        })),
      ...quizzes.data
        .filter((quiz) => inRange(quiz.due_at))
        .map((quiz) => ({
          id: quiz.id,
          type: "quiz_due" as const,
          title: quiz.title,
          starts_at: quiz.due_at!,
          course_id: quiz.course_id,
          course_title: quiz.course_title,
          url: `/student/assessments/${quiz.id}`,
        })),
    ].sort((a, b) => new Date(a.starts_at).getTime() - new Date(b.starts_at).getTime()),
  };
};

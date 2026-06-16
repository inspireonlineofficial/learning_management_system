import { listLiveSessions } from "./live";

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
  const sessions = await listLiveSessions({
    from: range.from,
    to: range.to,
    limit: 100,
  });

  return {
    items: sessions.data.map((session) => ({
      id: session.id,
      type: "live_session" as const,
      title: session.title,
      starts_at: session.starts_at,
      ends_at: session.ends_at,
      course_id: session.course_id,
      is_today: Boolean((session as { is_today?: boolean }).is_today),
      url: `/student/live-classes/${session.id}`,
    })),
  };
};

import { apiRequest } from "./client";

export type LiveSessionStatus = "scheduled" | "live" | "ended" | "cancelled";

export type LiveSessionSummary = {
  id: string;
  course_id: string;
  course_title?: string;
  title: string;
  description?: string | null;
  host_name?: string;
  starts_at: string;
  ends_at: string;
  duration_minutes?: number;
  status: LiveSessionStatus;
  is_registered?: boolean;
  attendees_count?: number;
  capacity?: number | null;
  cover_url?: string | null;
};

export type LiveSessionList = {
  data: LiveSessionSummary[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export type LiveSessionDetail = LiveSessionSummary & {
  join_url?: string | null;
  recording_url?: string | null;
  agenda?: string | null;
  materials?: { id: string; title: string; url: string }[];
};

export function listLiveSessions(
  params: {
    scope?: "upcoming" | "past" | "live";
    page?: number;
    limit?: number;
    from?: string;
    to?: string;
  } = {},
) {
  return apiRequest<{ sessions: BackendLiveSession[] }>("/v1/student/live-sessions", {
    auth: true,
    query: params.from || params.to ? { from: params.from, to: params.to } : undefined,
  }).then((result) => {
    const data = (result.sessions ?? []).map(normalizeLiveSession);
    const filtered =
      params.scope === "live"
        ? data.filter((session) => session.status === "live")
        : params.scope === "past"
          ? data.filter((session) => session.status === "ended" || session.status === "cancelled")
          : data;
    return {
      data: filtered,
      meta: {
        page: params.page ?? 1,
        limit: params.limit ?? (filtered.length > 0 ? filtered.length : 20),
        total: filtered.length,
        total_pages: 1,
      },
    };
  });
}

export function getLiveSession(sessionId: string) {
  return apiRequest<BackendLiveSession>(
    `/v1/student/live-sessions/${encodeURIComponent(sessionId)}`,
    { auth: true },
  ).then(normalizeLiveSession);
}

export function joinSession(sessionId: string) {
  return apiRequest<{ session_id: string; room_token: string }>(
    `/v1/live-sessions/${encodeURIComponent(sessionId)}/join`,
    { method: "POST", auth: true },
  ).then((result) => ({
    room_token: result.room_token,
    join_url: result.room_token,
    session_id: result.session_id,
  }));
}

// ---------- Teacher endpoints ----------

export type TeacherLiveInput = {
  course_id: string;
  title: string;
  scheduled_at: string;
  duration_minutes: number;
  record_session?: boolean;
};

export function listTeacherLiveSessions(
  params: { scope?: "upcoming" | "past" | "live"; page?: number; limit?: number } = {},
) {
  return apiRequest<{ sessions: BackendLiveSession[] }>("/v1/teacher/live-sessions", {
    auth: true,
    query: params,
  }).then((result) => {
    const data = (result.sessions ?? []).map(normalizeLiveSession);
    const filtered =
      params.scope === "live"
        ? data.filter((session) => session.status === "live")
        : params.scope === "past"
          ? data.filter((session) => session.status === "ended" || session.status === "cancelled")
          : data;
    return {
      data: filtered,
      meta: {
        page: params.page ?? 1,
        limit: params.limit ?? (filtered.length > 0 ? filtered.length : 20),
        total: filtered.length,
        total_pages: 1,
      },
    };
  });
}

export function getTeacherLiveSession(sessionId: string) {
  return apiRequest<BackendLiveSession>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}`,
    { auth: true },
  ).then(normalizeLiveSession);
}

export function createLiveSession(input: TeacherLiveInput) {
  return apiRequest<BackendLiveSession>("/v1/teacher/live-sessions", {
    method: "POST",
    auth: true,
    body: input,
  }).then(normalizeLiveSession);
}

export function updateLiveSession(sessionId: string, input: Partial<TeacherLiveInput>) {
  return apiRequest<BackendLiveSession>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}`,
    { method: "PATCH", auth: true, body: input },
  ).then(normalizeLiveSession);
}

export function cancelLiveSession(sessionId: string) {
  return apiRequest<BackendLiveSession>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}`,
    {
      method: "PATCH",
      auth: true,
      body: { status: "cancelled" },
    },
  ).then(() => ({ ok: true }));
}

export function startLiveSession(sessionId: string) {
  return apiRequest<{ session: BackendLiveSession; room_token: string }>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/start`,
    { method: "POST", auth: true },
  ).then((result) => ({
    join_url: result.room_token,
    room_token: result.room_token,
    session: normalizeLiveSession(result.session),
  }));
}

export function endLiveSession(sessionId: string) {
  return apiRequest<BackendLiveSession>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/end`,
    { method: "POST", auth: true },
  ).then(normalizeLiveSession);
}

export function hostJoinSession(sessionId: string) {
  return apiRequest<{ session: BackendLiveSession; room_token: string }>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/start`,
    { method: "POST", auth: true },
  ).then((result) => ({
    join_url: result.room_token,
    room_token: result.room_token,
    session: normalizeLiveSession(result.session),
  }));
}

// ---------- Live room chat & participants ----------

export type LiveChatMessage = {
  id: string;
  author_id: string;
  author_name: string;
  author_role: "host" | "student";
  text: string;
  created_at: string;
};

export type LiveParticipant = {
  id: string;
  full_name: string;
  role: "host" | "student";
  status: "joined" | "left";
  hand_raised?: boolean;
  joined_at?: string;
};

export function listLiveChat(sessionId: string, params: { since?: string } = {}) {
  return apiRequest<{ data: LiveChatMessage[] }>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/chat`,
    { auth: true, query: params },
  );
}

export function postLiveChat(sessionId: string, text: string) {
  return apiRequest<LiveChatMessage>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/chat`,
    { method: "POST", auth: true, body: { text } },
  );
}

export function listLiveParticipants(sessionId: string) {
  return apiRequest<{ data: LiveParticipant[] }>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/participants`,
    { auth: true },
  );
}

export function muteParticipant(sessionId: string, participantId: string) {
  return apiRequest<{ ok: true }>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/participants/${encodeURIComponent(participantId)}/mute`,
    { method: "POST", auth: true },
  );
}

export function removeParticipant(sessionId: string, participantId: string) {
  return apiRequest<{ ok: true }>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/participants/${encodeURIComponent(participantId)}`,
    { method: "DELETE", auth: true },
  );
}

export function lowerHand(sessionId: string, participantId: string) {
  return apiRequest<{ ok: true }>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/participants/${encodeURIComponent(participantId)}/lower-hand`,
    { method: "POST", auth: true },
  );
}

export function setRecording(sessionId: string, recording: boolean) {
  return apiRequest<{ ok: true; recording: boolean }>(
    `/v1/teacher/live-sessions/${encodeURIComponent(sessionId)}/recording`,
    { method: "POST", auth: true, body: { recording } },
  );
}

type BackendLiveSession = {
  id: string;
  course_id: string;
  teacher_id: string;
  title: string;
  scheduled_at: string;
  duration_minutes: number;
  status: LiveSessionStatus;
  record_session: boolean;
  attendee_count: number;
  is_today: boolean;
  created_at: string;
  updated_at: string;
};

function normalizeLiveSession(session: BackendLiveSession): LiveSessionSummary {
  const starts = new Date(session.scheduled_at);
  const ends = new Date(starts.getTime() + session.duration_minutes * 60_000);
  return {
    id: session.id,
    course_id: session.course_id,
    title: session.title,
    starts_at: starts.toISOString(),
    ends_at: ends.toISOString(),
    duration_minutes: session.duration_minutes,
    status: session.status,
    attendees_count: session.attendee_count,
    is_registered: false,
  };
}

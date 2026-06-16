import { apiRequest } from "./client";

export type Badge = {
  id: string;
  name: string;
  description?: string;
  icon?: string | null;
  tier?: "bronze" | "silver" | "gold" | "platinum";
  earned_at?: string | null;
  progress?: { current: number; target: number } | null;
};

export type Achievement = {
  id: string;
  title: string;
  description?: string;
  points: number;
  earned_at: string;
  icon?: string | null;
};

export type LevelInfo = {
  level: number;
  title?: string;
  points: number;
  points_to_next: number;
  level_progress_percent: number;
};

export type GamificationOverview = {
  level: LevelInfo;
  total_points: number;
  rank?: number | null;
  streak_days: number;
  longest_streak: number;
  badges_earned: number;
  badges_total: number;
  recent_achievements: Achievement[];
};

export type LeaderboardEntry = {
  rank: number;
  user_id: string;
  full_name: string;
  avatar_url?: string | null;
  points: number;
  level: number;
  is_me?: boolean;
};

export type LeaderboardScope = "weekly" | "monthly" | "all_time";

export function getGamificationOverview() {
  return apiRequest<{
    total?: number;
    total_points?: number;
    global_rank?: number;
    rank?: number;
    streak_days?: number;
    longest_streak_days?: number;
    milestones?: Array<{ id: string; label: string; threshold: number; achieved_at?: string }>;
    recent_events?: Array<{ id: string; reason: string; points: number; created_at: string }>;
  }>("/v1/student/points", { auth: true }).then((points) => ({
    level: (() => {
      const total = points.total_points ?? points.total ?? 0;
      return {
        level: Math.max(1, Math.floor(total / 100) + 1),
        title: "Scholar",
        points: total,
        points_to_next: 100 - (total % 100),
        level_progress_percent: total % 100,
      };
    })(),
    total_points: points.total_points ?? points.total ?? 0,
    rank: points.global_rank ?? points.rank,
    streak_days: points.streak_days ?? 0,
    longest_streak: points.longest_streak_days ?? points.streak_days ?? 0,
    badges_earned: (points.milestones ?? []).filter((m) => m.achieved_at).length,
    badges_total: points.milestones?.length ?? 0,
    recent_achievements: (points.recent_events ?? []).map((event) => ({
      id: event.id,
      title: event.reason,
      points: event.points,
      earned_at: event.created_at,
    })),
  }));
}

export function listBadges(params: { earned?: boolean } = {}) {
  return apiRequest<{
    milestones?: Array<{ id: string; label: string; threshold: number; achieved_at?: string }>;
  }>("/v1/student/points", {
    auth: true,
  }).then((points) => {
    const data = (points.milestones ?? []).map((m) => ({
      id: m.id,
      name: m.label,
      points: m.threshold,
      tier: m.threshold >= 1000 ? "gold" : m.threshold >= 300 ? "silver" : "bronze",
      earned_at: m.achieved_at ?? null,
      progress: m.achieved_at ? null : { current: 0, target: m.threshold },
    })) satisfies Badge[];
    return {
      data:
        params.earned == null ? data : data.filter((b) => Boolean(b.earned_at) === params.earned),
    };
  });
}

export function listAchievements(params: { page?: number; limit?: number } = {}) {
  return apiRequest<{
    events?: Array<{
      id: string;
      type: string;
      source_title: string;
      points: number;
      earned_at: string;
    }>;
    meta?: { page: number; limit: number; total: number; total_pages: number };
  }>("/v1/student/points/history", { auth: true, query: params }).then((history) => ({
    data: (history.events ?? []).map((event) => ({
      id: event.id,
      title: event.source_title || event.type,
      description: event.type,
      points: event.points,
      earned_at: event.earned_at,
    })),
    meta: history.meta ?? {
      page: params.page ?? 1,
      limit: params.limit ?? 25,
      total: 0,
      total_pages: 1,
    },
  }));
}

export function getLeaderboard(params: { scope?: LeaderboardScope; limit?: number } = {}) {
  return apiRequest<{
    entries: Array<{
      rank: number;
      student_id?: string;
      user_id?: string;
      display_name?: string;
      full_name?: string;
      score?: number;
      points?: number;
    }>;
  }>("/v1/leaderboard", {
    auth: true,
    query: { period: params.scope === "weekly" ? "weekly" : "alltime", limit: params.limit },
  }).then((result) => ({
    data: result.entries.map((entry, index) => {
      const points = Math.round(entry.score ?? entry.points ?? 0);
      return {
        rank: entry.rank,
        user_id: entry.student_id ?? entry.user_id ?? `leaderboard-${index}`,
        full_name: entry.display_name ?? entry.full_name ?? "Student",
        points,
        level: Math.max(1, Math.floor(points / 100) + 1),
      };
    }),
  }));
}

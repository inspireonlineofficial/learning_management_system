import { apiRequest } from "./client";

export type PointsBreakdown = {
  total: number;
  streak_days: number;
  longest_streak_days?: number;
  this_week: number;
  this_month: number;
  daily: Array<{ date: string; points: number }>;
  by_source?: Array<{ source: string; points: number }>;
  milestones?: Array<{
    id: string;
    label: string;
    threshold: number;
    achieved_at?: string | null;
  }>;
  recent_events: Array<{ id: string; reason: string; points: number; created_at: string }>;
};
export const getPointsBreakdown = (period: "7d" | "30d" = "30d") =>
  apiRequest<
    Partial<PointsBreakdown> & {
      total_points?: number;
      global_rank?: number;
      history?: Array<{ id: string; reason: string; points: number; created_at: string }>;
    }
  >("/v1/student/points", { auth: true, query: { period } }).then((points) => ({
    total: points.total ?? points.total_points ?? 0,
    streak_days: points.streak_days ?? 0,
    longest_streak_days: points.longest_streak_days ?? points.streak_days ?? 0,
    this_week: points.this_week ?? 0,
    this_month: points.this_month ?? 0,
    daily: points.daily ?? [],
    by_source: points.by_source ?? [],
    milestones: points.milestones ?? [],
    recent_events: points.recent_events ?? points.history ?? [],
  }));

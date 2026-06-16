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
  apiRequest<PointsBreakdown>("/v1/student/points", { auth: true, query: { period } });

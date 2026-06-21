import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Award, Flame, Sparkles, Trophy } from "lucide-react";
import { useState } from "react";

import { AppShell, EmptyState, SectionHeading, StatCard } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import {
  getGamificationOverview,
  listAchievements,
  listBadges,
  type Badge,
} from "@/lib/api/gamification";

export const Route = createFileRoute("/_authenticated/student/achievements")({
  component: AchievementsPage,
});

function AchievementsPage() {
  const [filter, setFilter] = useState<"all" | "earned" | "locked">("all");

  const overview = useQuery({
    queryKey: ["gamification-overview"],
    queryFn: () => getGamificationOverview(),
  });

  const badges = useQuery({
    queryKey: ["badges", filter],
    queryFn: () => listBadges(filter === "all" ? {} : { earned: filter === "earned" }),
  });

  const achievements = useQuery({
    queryKey: ["achievements-feed"],
    queryFn: () => listAchievements({ limit: 25 }),
  });

  const o = overview.data;

  return (
    <AppShell eyebrow="Progress" title="Honours & milestones.">
      {overview.isError ? (
        <QueryErrorPanel
          error={overview.error}
          title="Couldn't load progress"
          onRetry={() => overview.refetch()}
          className="mb-8"
        />
      ) : (
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
          <StatCard
            label="Total points"
            value={overview.isLoading ? "—" : (o?.total_points.toLocaleString() ?? "0")}
            hint={o?.rank ? `Rank #${o.rank}` : undefined}
          />
          <StatCard
            label={`Level ${o?.level.level ?? "—"}`}
            value={o?.level.title ?? "Scholar"}
            hint={o?.level ? `${o.level.points_to_next} pts to next level` : undefined}
          />
          <StatCard
            label="Current streak"
            value={
              <span className="flex items-center gap-2">
                {o?.streak_days ?? 0}
                <Flame className="h-6 w-6 text-amber-500" />
              </span>
            }
            hint={o ? `Longest ${o.longest_streak} days` : undefined}
          />
          <StatCard label="Badges" value={`${o?.badges_earned ?? 0} / ${o?.badges_total ?? 0}`} />
        </div>
      )}

      {o?.level && (
        <div className="mb-10">
          <div className="flex justify-between text-xs text-brand/55 mb-2">
            <span>Level {o.level.level} progress</span>
            <span>{o.level.level_progress_percent}%</span>
          </div>
          <div className="h-2 bg-brand/10 overflow-hidden">
            <div
              className="h-full bg-accent transition-all"
              style={{ width: `${Math.min(100, o.level.level_progress_percent)}%` }}
            />
          </div>
        </div>
      )}

      <div className="mb-10">
        <SectionHeading
          title="Badges"
          action={
            <div className="flex gap-1">
              {(["all", "earned", "locked"] as const).map((f) => (
                <button
                  key={f}
                  onClick={() => setFilter(f)}
                  className={`px-3 py-1.5 text-xs font-medium capitalize ${
                    filter === f
                      ? "bg-brand text-white"
                      : "border border-brand/15 text-brand/70 hover:text-brand"
                  }`}
                >
                  {f}
                </button>
              ))}
            </div>
          }
        />
        {badges.isLoading ? (
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="h-32 border border-brand/10 bg-white/30 animate-pulse" />
            ))}
          </div>
        ) : !badges.data || badges.data.data.length === 0 ? (
          <EmptyState icon={Award} title="No badges to show" />
        ) : (
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {badges.data.data.map((b) => (
              <BadgeCard key={b.id} badge={b} />
            ))}
          </div>
        )}
      </div>

      <div>
        <SectionHeading title="Recent achievements" />
        {achievements.isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-16 border border-brand/10 bg-white/30 animate-pulse" />
            ))}
          </div>
        ) : !achievements.data || achievements.data.data.length === 0 ? (
          <EmptyState
            icon={Trophy}
            title="No achievements yet"
            description="Complete lessons and quizzes to earn them."
          />
        ) : (
          <ul className="divide-y divide-brand/10 border-y border-brand/10">
            {achievements.data.data.map((a) => (
              <li key={a.id} className="flex items-center gap-4 py-4 px-2">
                <div className="h-10 w-10 grid place-items-center bg-accent/10 text-accent">
                  <Sparkles className="h-4 w-4" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-brand">{a.title}</p>
                  {a.description && <p className="text-xs text-brand/55">{a.description}</p>}
                </div>
                <div className="text-right">
                  <p className="font-serif text-lg text-accent">+{a.points}</p>
                  <p className="text-[11px] text-brand/45">
                    {new Date(a.earned_at).toLocaleDateString()}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </AppShell>
  );
}

function BadgeCard({ badge: b }: { badge: Badge }) {
  const earned = !!b.earned_at;
  const tier = b.tier ?? "bronze";
  const tierClass = {
    bronze: "text-amber-700",
    silver: "text-slate-500",
    gold: "text-amber-500",
    platinum: "text-violet-600",
  }[tier];

  return (
    <div
      className={`border p-5 transition-all ${
        earned
          ? "border-brand/15 bg-white/50"
          : "border-dashed border-brand/15 bg-transparent opacity-70"
      }`}
    >
      <div className="flex items-start gap-3 mb-3">
        <div
          className={`h-12 w-12 grid place-items-center border ${
            earned ? `border-brand/15 ${tierClass}` : "border-brand/10 text-brand/30"
          }`}
        >
          <Award className="h-5 w-5" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="font-serif text-base leading-snug">{b.name}</p>
          <p className="text-[11px] uppercase tracking-wider text-brand/45 mt-0.5">{tier}</p>
        </div>
      </div>
      {b.description && <p className="text-xs text-brand/60 leading-relaxed">{b.description}</p>}
      {b.progress && !earned && (
        <div className="mt-3">
          <div className="flex justify-between text-[10px] text-brand/50 mb-1">
            <span>
              {b.progress.current} / {b.progress.target}
            </span>
            <span>{Math.round((b.progress.current / b.progress.target) * 100)}%</span>
          </div>
          <div className="h-1 bg-brand/10">
            <div
              className="h-full bg-accent"
              style={{
                width: `${Math.min(100, (b.progress.current / b.progress.target) * 100)}%`,
              }}
            />
          </div>
        </div>
      )}
      {earned && b.earned_at && (
        <p className="mt-3 text-[11px] text-brand/45">
          Earned {new Date(b.earned_at).toLocaleDateString()}
        </p>
      )}
    </div>
  );
}

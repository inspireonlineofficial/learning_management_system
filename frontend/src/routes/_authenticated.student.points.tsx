import { createFileRoute } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, Flame, Lock, Sparkles, Trophy } from "lucide-react";

import { AppShell, StatCard, SectionHeading } from "@/components/layout/app-shell";
import { getPointsBreakdown } from "@/lib/api/points";

export const Route = createFileRoute("/_authenticated/student/points")({
  component: Page,
});

function Page() {
  const [period, setPeriod] = useState<"7d" | "30d">("30d");
  const { data, isLoading } = useQuery({
    queryKey: ["points", period],
    queryFn: () => getPointsBreakdown(period),
  });

  const daily = data?.daily ?? [];
  const max = Math.max(1, ...daily.map((d) => d.points));
  const today = daily[daily.length - 1]?.points ?? 0;

  // Build a 35-cell streak grid (last 5 weeks) from daily data
  const streakGrid = useMemo(() => {
    const tail = daily.slice(-35);
    return Array.from({ length: 35 }, (_, i) => tail[i] ?? { date: "", points: 0 });
  }, [daily]);

  const milestones = data?.milestones ?? [];
  const bySource = data?.by_source ?? [];
  const totalSource = Math.max(
    1,
    bySource.reduce((s, x) => s + x.points, 0),
  );

  return (
    <AppShell eyebrow="Points" title="Points & streaks">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard label="Total" value={isLoading ? "—" : (data?.total.toLocaleString() ?? 0)} />
        <StatCard
          label="Streak"
          value={isLoading ? "—" : `${data?.streak_days ?? 0}d`}
          hint={data?.longest_streak_days ? `best ${data.longest_streak_days}d` : "days in a row"}
        />
        <StatCard label="This week" value={isLoading ? "—" : (data?.this_week ?? 0)} />
        <StatCard label="This month" value={isLoading ? "—" : (data?.this_month ?? 0)} />
      </div>

      <SectionHeading
        title="Activity"
        action={
          <div className="flex gap-1 text-xs">
            {(["7d", "30d"] as const).map((p) => (
              <button
                key={p}
                onClick={() => setPeriod(p)}
                className={`px-3 py-1 border ${period === p ? "border-brand bg-brand text-white" : "border-brand/15 text-brand/70"}`}
              >
                {p}
              </button>
            ))}
          </div>
        }
      />
      <div className="mt-6 border border-brand/10 bg-white/40 p-6">
        {daily.length > 0 ? (
          <>
            <div className="flex items-end gap-1 h-40">
              {daily.map((d) => (
                <div key={d.date} className="flex-1 flex flex-col justify-end gap-1">
                  <div
                    className="bg-accent/70 hover:bg-accent transition-colors"
                    style={{
                      height: `${(d.points / max) * 100}%`,
                      minHeight: d.points > 0 ? 2 : 0,
                    }}
                    title={`${d.date}: ${d.points} pts`}
                  />
                </div>
              ))}
            </div>
            <div className="flex justify-between text-[10px] text-brand/45 mt-2">
              <span>{daily[0]?.date}</span>
              <span>Today · {today} pts</span>
              <span>{daily[daily.length - 1]?.date}</span>
            </div>
          </>
        ) : (
          <p className="text-sm text-brand/55">No activity yet.</p>
        )}
      </div>

      <div className="grid lg:grid-cols-2 gap-6 mt-10">
        <div>
          <SectionHeading title="Streak calendar" />
          <div className="border border-brand/10 bg-white/40 p-5 mt-4">
            <div className="flex items-center gap-3 mb-4">
              <Flame className="h-5 w-5 text-accent" />
              <div>
                <p className="font-serif text-2xl">{data?.streak_days ?? 0} days</p>
                <p className="text-[11px] text-brand/55">
                  Keep going — earn at least 1 point each day.
                </p>
              </div>
            </div>
            <div className="grid grid-cols-7 gap-1.5">
              {streakGrid.map((d, i) => (
                <div
                  key={i}
                  title={d.date ? `${d.date}: ${d.points} pts` : ""}
                  className={`aspect-square ${
                    d.points > 0
                      ? d.points >= max * 0.66
                        ? "bg-accent"
                        : d.points >= max * 0.33
                          ? "bg-accent/70"
                          : "bg-accent/40"
                      : "bg-brand/5 border border-brand/10"
                  }`}
                />
              ))}
            </div>
            <div className="flex items-center justify-end gap-2 mt-3 text-[10px] text-brand/55">
              <span>Less</span>
              <div className="h-3 w-3 bg-brand/5 border border-brand/10" />
              <div className="h-3 w-3 bg-accent/40" />
              <div className="h-3 w-3 bg-accent/70" />
              <div className="h-3 w-3 bg-accent" />
              <span>More</span>
            </div>
          </div>
        </div>

        <div>
          <SectionHeading title="Milestones" />
          <ul className="space-y-2 mt-4">
            {milestones.length === 0 && !isLoading && (
              <li className="text-sm text-brand/55 border border-dashed border-brand/15 p-6 text-center">
                No milestones configured.
              </li>
            )}
            {milestones.map((m) => {
              const achieved = !!m.achieved_at;
              const pct = Math.min(100, Math.round(((data?.total ?? 0) / m.threshold) * 100));
              return (
                <li
                  key={m.id}
                  className={`border p-4 flex items-center gap-4 ${achieved ? "border-accent/40 bg-accent/[0.04]" : "border-brand/10 bg-white/40"}`}
                >
                  <div
                    className={`h-10 w-10 grid place-items-center ${achieved ? "bg-accent text-white" : "bg-brand/5 text-brand/40"}`}
                  >
                    {achieved ? <Check className="h-4 w-4" /> : <Lock className="h-4 w-4" />}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{m.label}</p>
                    <div className="h-1 bg-brand/10 mt-1.5 overflow-hidden">
                      <div
                        className={`h-full ${achieved ? "bg-accent" : "bg-brand/40"}`}
                        style={{ width: `${pct}%` }}
                      />
                    </div>
                    <p className="text-[11px] text-brand/55 mt-1">
                      {achieved
                        ? `Achieved ${new Date(m.achieved_at!).toLocaleDateString()}`
                        : `${(data?.total ?? 0).toLocaleString()} / ${m.threshold.toLocaleString()} pts`}
                    </p>
                  </div>
                  <Trophy className={`h-4 w-4 ${achieved ? "text-accent" : "text-brand/25"}`} />
                </li>
              );
            })}
          </ul>
        </div>
      </div>

      {bySource.length > 0 && (
        <>
          <SectionHeading title="Points by source" />
          <ul className="space-y-2 mt-4">
            {bySource.map((s) => {
              const pct = Math.round((s.points / totalSource) * 100);
              return (
                <li key={s.source} className="border border-brand/10 bg-white/40 p-4">
                  <div className="flex items-center justify-between text-sm mb-2">
                    <span className="capitalize">{s.source.replace(/_/g, " ")}</span>
                    <span className="text-brand/70">
                      {s.points.toLocaleString()}{" "}
                      <span className="text-brand/45 text-xs">({pct}%)</span>
                    </span>
                  </div>
                  <div className="h-1.5 bg-brand/10 overflow-hidden">
                    <div className="h-full bg-brand/60" style={{ width: `${pct}%` }} />
                  </div>
                </li>
              );
            })}
          </ul>
        </>
      )}

      <SectionHeading title="Recent rewards" />
      <ul className="space-y-2 mt-4">
        {data?.recent_events.map((e) => (
          <li key={e.id} className="border border-brand/10 bg-white/40 p-4 flex items-center gap-3">
            <Sparkles className="h-4 w-4 text-accent" />
            <p className="flex-1 text-sm">{e.reason}</p>
            <span className="font-serif text-lg">+{e.points}</span>
            <span className="text-xs text-brand/45">
              {new Date(e.created_at).toLocaleDateString()}
            </span>
          </li>
        )) ?? null}
      </ul>
    </AppShell>
  );
}

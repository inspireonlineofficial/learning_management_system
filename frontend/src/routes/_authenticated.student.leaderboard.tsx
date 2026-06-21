import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Crown, Trophy, User } from "lucide-react";
import { useState } from "react";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import {
  getLeaderboard,
  type LeaderboardEntry,
  type LeaderboardScope,
} from "@/lib/api/gamification";

export const Route = createFileRoute("/_authenticated/student/leaderboard")({
  component: LeaderboardPage,
});

function LeaderboardPage() {
  const [scope, setScope] = useState<LeaderboardScope>("weekly");

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["leaderboard", scope],
    queryFn: () => getLeaderboard({ scope, limit: 50 }),
  });

  return (
    <AppShell eyebrow="Standings" title="The leaderboard.">
      <div className="flex gap-2 mb-8">
        {(["weekly", "monthly", "all_time"] as LeaderboardScope[]).map((s) => (
          <button
            key={s}
            onClick={() => setScope(s)}
            className={`px-5 py-2 text-sm font-medium transition-colors ${
              scope === s
                ? "bg-brand text-white"
                : "border border-brand/15 text-brand/70 hover:text-brand hover:bg-brand/[0.03]"
            }`}
          >
            {s === "all_time" ? "All time" : s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
      </div>

      {isError ? (
        <QueryErrorPanel
          error={error}
          title="Couldn't load leaderboard"
          onRetry={() => refetch()}
        />
      ) : isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-14 border border-brand/10 bg-white/30 animate-pulse" />
          ))}
        </div>
      ) : !data || data.data.length === 0 ? (
        <EmptyState icon={Trophy} title="No standings yet" />
      ) : (
        <>
          {data.data.slice(0, 3).length > 0 && (
            <div className="grid sm:grid-cols-3 gap-3 mb-8">
              {data.data.slice(0, 3).map((e) => (
                <PodiumCard key={e.user_id} entry={e} />
              ))}
            </div>
          )}

          <ul className="divide-y divide-brand/10 border-y border-brand/10">
            {data.data.slice(3).map((e) => (
              <RankRow key={e.user_id} entry={e} />
            ))}
          </ul>

          {data.me && !data.data.some((e) => e.is_me) && (
            <div className="mt-8 border-2 border-accent/40 bg-accent/5">
              <RankRow entry={{ ...data.me, is_me: true }} />
            </div>
          )}
        </>
      )}
    </AppShell>
  );
}

function PodiumCard({ entry: e }: { entry: LeaderboardEntry }) {
  const color =
    e.rank === 1
      ? "text-amber-500 border-amber-300"
      : e.rank === 2
        ? "text-slate-500 border-slate-300"
        : "text-amber-700 border-amber-300";
  return (
    <div className={`border bg-white/50 p-6 text-center ${e.is_me ? "ring-2 ring-accent" : ""}`}>
      <div className={`inline-flex h-10 w-10 items-center justify-center border-2 ${color} mb-3`}>
        {e.rank === 1 ? (
          <Crown className="h-5 w-5" />
        ) : (
          <span className="font-serif">{e.rank}</span>
        )}
      </div>
      <Avatar entry={e} />
      <p className="mt-3 font-medium text-brand truncate">{e.full_name}</p>
      <p className="text-xs text-brand/50">Level {e.level}</p>
      <p className="mt-3 font-serif text-2xl text-accent">{e.points.toLocaleString()}</p>
    </div>
  );
}

function RankRow({ entry: e }: { entry: LeaderboardEntry }) {
  return (
    <li className={`flex items-center gap-4 py-4 px-4 ${e.is_me ? "bg-accent/5" : ""}`}>
      <span className="w-8 text-center font-serif text-lg text-brand/55">{e.rank}</span>
      <Avatar entry={e} small />
      <div className="flex-1 min-w-0">
        <p className="font-medium text-brand truncate">
          {e.full_name}
          {e.is_me && <span className="ml-2 text-[11px] text-accent">you</span>}
        </p>
        <p className="text-xs text-brand/50">Level {e.level}</p>
      </div>
      <p className="font-serif text-lg text-brand">{e.points.toLocaleString()}</p>
    </li>
  );
}

function Avatar({ entry: e, small }: { entry: LeaderboardEntry; small?: boolean }) {
  const size = small ? "h-9 w-9" : "h-14 w-14 mx-auto";
  if (e.avatar_url) {
    return <img src={e.avatar_url} alt="" className={`${size} object-cover`} />;
  }
  return (
    <div className={`${size} grid place-items-center bg-brand/10 text-brand`}>
      <User className={small ? "h-4 w-4" : "h-6 w-6"} />
    </div>
  );
}

import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { AlertTriangle, ArrowLeft, Search } from "lucide-react";
import { useEffect, useState } from "react";
import { z } from "zod";

import { RichContent } from "@/components/assessment/rich-content";
import { AppShell } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import {
  getQuizAnalytics,
  listQuizAttempts,
  type AttemptRow,
  type QuizAnalytics,
} from "@/lib/api/teacher-quizzes";

const searchSchema = z.object({
  tab: z.enum(["attempts", "analytics"]).optional(),
  status: z.enum(["", "in_progress", "submitted", "pending_grading", "graded"]).optional(),
  q: z.string().optional(),
  sort: z.enum(["started_at", "submitted_at", "score", "student"]).optional(),
  order: z.enum(["asc", "desc"]).optional(),
  page: z.number().int().positive().optional(),
});

export const Route = createFileRoute("/_authenticated/teacher/quiz-builder/$quizId/attempts/")({
  validateSearch: searchSchema,
  component: Page,
});

function formatDuration(seconds?: number | null) {
  if (seconds == null) return "—";
  const m = Math.floor(seconds / 60);
  return m >= 60 ? `${Math.floor(m / 60)} h ${m % 60} min` : `${m} min ${seconds % 60} s`;
}

function Page() {
  const { quizId } = Route.useParams();
  const search = Route.useSearch();
  const tab = search.tab ?? "attempts";
  return (
    <AppShell eyebrow="Assessment monitoring" title="Attempts and results">
      <Link
        to="/teacher/quiz-builder/$quizId/edit"
        params={{ quizId }}
        className="mb-6 inline-flex items-center gap-2 text-xs text-brand/70 hover:text-brand"
      >
        <ArrowLeft className="h-3.5 w-3.5" aria-hidden /> Back to the quiz builder
      </Link>
      <div role="tablist" aria-label="View" className="mb-6 flex gap-1 border-b border-brand/10">
        {(["attempts", "analytics"] as const).map((t) => (
          <Link
            key={t}
            role="tab"
            aria-selected={tab === t}
            to="/teacher/quiz-builder/$quizId/attempts"
            params={{ quizId }}
            search={(prev) => ({ ...prev, tab: t })}
            className={`-mb-px border-b-2 px-4 py-2 text-sm capitalize ${tab === t ? "border-brand font-medium" : "border-transparent text-brand/70 hover:text-brand"}`}
          >
            {t}
          </Link>
        ))}
      </div>
      {tab === "attempts" ? <AttemptsTab quizId={quizId} /> : <AnalyticsTab quizId={quizId} />}
    </AppShell>
  );
}

function AttemptsTab({ quizId }: { quizId: string }) {
  const search = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });
  const [query, setQuery] = useState(search.q ?? "");
  useEffect(() => {
    const t = setTimeout(() => {
      if ((search.q ?? "") !== query)
        navigate({ search: (p) => ({ ...p, q: query || undefined, page: 1 }), replace: true });
    }, 300);
    return () => clearTimeout(t);
  }, [query, search.q, navigate]);

  const params = {
    status: search.status || undefined,
    q: search.q || undefined,
    sort: search.sort,
    order: search.order,
    page: search.page ?? 1,
    limit: 25,
  };
  const { data, isLoading, isError, error, refetch, isFetching } = useQuery({
    queryKey: ["quiz-attempts", quizId, params],
    queryFn: () => listQuizAttempts(quizId, params),
    placeholderData: keepPreviousData,
    refetchInterval: 30_000,
  });

  if (isError)
    return (
      <QueryErrorPanel error={error} title="Couldn't load attempts" onRetry={() => refetch()} />
    );
  if (isLoading || !data)
    return <div className="h-40 animate-pulse border border-brand/10 bg-white/30" />;
  const s = data.summary;
  const sortBy = (sort: NonNullable<typeof search.sort>) =>
    navigate({
      search: (p) => ({
        ...p,
        sort,
        order: p.sort === sort && p.order !== "asc" ? "asc" : "desc",
        page: 1,
      }),
    });

  return (
    <div>
      <h2 className="font-serif text-xl">{data.quiz.title}</h2>
      <dl className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <Stat label="Enrolled" value={s.enrolled_students} />
        <Stat label="Started" value={s.students_attempted} />
        <Stat label="Not started" value={s.students_not_started} />
        <Stat label="In progress" value={s.in_progress} />
        <Stat label="Submitted" value={s.submitted} />
        <Stat label="To mark" value={s.pending_grading} highlight={s.pending_grading > 0} />
        <Stat label="Average" value={s.average_score != null ? `${s.average_score}%` : "—"} />
        <Stat label="Highest" value={s.highest_score != null ? `${s.highest_score}%` : "—"} />
        <Stat label="Lowest" value={s.lowest_score != null ? `${s.lowest_score}%` : "—"} />
        <Stat label="Pass rate" value={s.pass_rate != null ? `${s.pass_rate}%` : "—"} />
      </dl>
      <p className="mt-2 text-xs text-brand/70">
        Scores use each student's latest fully graded attempt.
      </p>

      <div className="mt-6 flex flex-wrap items-center gap-3">
        <label className="relative">
          <span className="sr-only">Search students</span>
          <Search
            className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-brand/70"
            aria-hidden
          />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search name or email"
            className="w-64 border border-brand/15 bg-white py-2 pl-9 pr-3 text-sm"
          />
        </label>
        <label className="text-sm">
          <span className="sr-only">Status</span>
          <select
            value={search.status ?? ""}
            onChange={(e) =>
              navigate({
                search: (p) => ({
                  ...p,
                  status: (e.target.value || undefined) as typeof p.status,
                  page: 1,
                }),
              })
            }
            className="border border-brand/15 bg-white px-3 py-2 text-sm"
          >
            <option value="">All attempts</option>
            <option value="pending_grading">Needs marking</option>
            <option value="graded">Graded</option>
            <option value="submitted">Submitted</option>
            <option value="in_progress">In progress</option>
          </select>
        </label>
        {isFetching && <span className="text-xs text-brand/70">Updating…</span>}
      </div>

      <div className="mt-4 overflow-x-auto border border-brand/10 bg-white/60">
        <table className="w-full min-w-[52rem] text-sm">
          <thead className="bg-brand/[0.03] text-left text-xs text-brand/70">
            <tr>
              <SortTh
                label="Student"
                active={search.sort === "student"}
                order={search.order}
                onClick={() => sortBy("student")}
              />
              <th className="px-3 py-2 font-medium">#</th>
              <th className="px-3 py-2 font-medium">Status</th>
              <SortTh
                label="Started"
                active={(search.sort ?? "started_at") === "started_at"}
                order={search.order}
                onClick={() => sortBy("started_at")}
              />
              <SortTh
                label="Submitted"
                active={search.sort === "submitted_at"}
                order={search.order}
                onClick={() => sortBy("submitted_at")}
              />
              <th className="px-3 py-2 font-medium">Time</th>
              <SortTh
                label="Score"
                active={search.sort === "score"}
                order={search.order}
                onClick={() => sortBy("score")}
              />
              <th className="px-3 py-2" />
            </tr>
          </thead>
          <tbody className="divide-y divide-brand/10">
            {data.attempts.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-3 py-8 text-center text-brand/70">
                  No attempts match.
                </td>
              </tr>
            ) : (
              data.attempts.map((a) => <AttemptTr key={a.attempt_id} a={a} quizId={quizId} />)
            )}
          </tbody>
        </table>
      </div>
      {data.meta.total_pages > 1 && (
        <div className="mt-4 flex items-center justify-between text-sm">
          <button
            type="button"
            disabled={data.meta.page <= 1}
            onClick={() => navigate({ search: (p) => ({ ...p, page: (p.page ?? 1) - 1 }) })}
            className="border border-brand/15 px-3 py-1.5 disabled:opacity-40"
          >
            Previous
          </button>
          <span className="tabular-nums text-brand/70">
            Page {data.meta.page} of {data.meta.total_pages} · {data.meta.total} attempts
          </span>
          <button
            type="button"
            disabled={data.meta.page >= data.meta.total_pages}
            onClick={() => navigate({ search: (p) => ({ ...p, page: (p.page ?? 1) + 1 }) })}
            className="border border-brand/15 px-3 py-1.5 disabled:opacity-40"
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}

function SortTh({
  label,
  active,
  order,
  onClick,
}: {
  label: string;
  active: boolean;
  order?: string;
  onClick: () => void;
}) {
  return (
    <th
      className="px-3 py-2 font-medium"
      aria-sort={active ? (order === "asc" ? "ascending" : "descending") : "none"}
    >
      <button
        type="button"
        onClick={onClick}
        className="inline-flex items-center gap-1 hover:text-brand"
      >
        {label} {active ? (order === "asc" ? "▲" : "▼") : ""}
      </button>
    </th>
  );
}

function AttemptTr({ a, quizId }: { a: AttemptRow; quizId: string }) {
  const status =
    a.status === "in_progress"
      ? { text: "In progress", tone: "text-sky-800" }
      : a.grading_status === "pending"
        ? { text: "Needs marking", tone: "text-accent-ink font-medium" }
        : {
            text: a.status === "auto_submitted" ? "Auto-submitted" : "Submitted",
            tone: "text-brand/70",
          };
  return (
    <tr className="hover:bg-brand/[0.02]">
      <td className="px-3 py-2.5">
        <p>{a.student_name || "Student"}</p>
        <p className="text-xs text-brand/70">{a.student_email}</p>
      </td>
      <td className="px-3 py-2.5 tabular-nums">{a.attempt_number}</td>
      <td className={`px-3 py-2.5 ${status.tone}`}>{status.text}</td>
      <td className="px-3 py-2.5 text-xs">{new Date(a.started_at).toLocaleString()}</td>
      <td className="px-3 py-2.5 text-xs">
        {a.submitted_at ? new Date(a.submitted_at).toLocaleString() : "—"}
      </td>
      <td className="px-3 py-2.5 tabular-nums text-xs">{formatDuration(a.time_taken_seconds)}</td>
      <td className="px-3 py-2.5 tabular-nums">
        {a.score_percent != null ? (
          <span className={a.passed ? "text-emerald-700" : "text-brand"}>{a.score_percent}%</span>
        ) : a.score_points != null ? (
          <span className="text-brand/70">
            {a.score_points}/{a.max_points} so far
          </span>
        ) : (
          "—"
        )}
      </td>
      <td className="px-3 py-2.5 text-right">
        <Link
          to="/teacher/quiz-builder/$quizId/attempts/$attemptId"
          params={{ quizId, attemptId: a.attempt_id }}
          className="text-sm text-accent-ink underline-offset-4 hover:underline"
        >
          {a.grading_status === "pending" && a.status !== "in_progress" ? "Mark" : "View"}
        </Link>
      </td>
    </tr>
  );
}

function Stat({
  label,
  value,
  highlight = false,
}: {
  label: string;
  value: number | string;
  highlight?: boolean;
}) {
  return (
    <div
      className={`border px-3 py-2.5 ${highlight ? "border-accent/40 bg-accent/5" : "border-brand/10 bg-white/50"}`}
    >
      <dt className="text-xs text-brand/70">{label}</dt>
      <dd className="font-serif text-2xl tabular-nums">{value}</dd>
    </div>
  );
}

function AnalyticsTab({ quizId }: { quizId: string }) {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["quiz-analytics", quizId],
    queryFn: () => getQuizAnalytics(quizId),
  });
  if (isError)
    return (
      <QueryErrorPanel error={error} title="Couldn't load analytics" onRetry={() => refetch()} />
    );
  if (isLoading || !data)
    return <div className="h-40 animate-pulse border border-brand/10 bg-white/30" />;
  if (data.submitted_attempts === 0) {
    return (
      <p className="rounded-lg border border-borderui bg-white p-8 text-center text-sm text-slatebody">
        No submitted attempts yet. Analytics appear once students submit.
      </p>
    );
  }
  return <Analytics data={data} />;
}

function Analytics({ data }: { data: QuizAnalytics }) {
  const maxBucket = Math.max(1, ...data.score_distribution.map((b) => b.count));
  const hardest = data.questions
    .filter((q) => q.accuracy_percent != null)
    .slice()
    .sort((a, b) => (a.accuracy_percent ?? 0) - (b.accuracy_percent ?? 0))
    .slice(0, 3);
  return (
    <div className="space-y-8">
      {data.small_sample && (
        <p className="flex items-start gap-2 border-l-2 border-amber-500 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
          Only {data.graded_attempts} graded attempt{data.graded_attempts === 1 ? "" : "s"} so far —
          treat these figures as indicative, not reliable.
        </p>
      )}
      <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Stat label="Submitted" value={data.submitted_attempts} />
        <Stat label="Graded" value={data.graded_attempts} />
        <Stat label="Average" value={data.average_score != null ? `${data.average_score}%` : "—"} />
        <Stat label="Median" value={data.median_score != null ? `${data.median_score}%` : "—"} />
      </dl>

      <section>
        <h3 className="font-serif text-lg">Score distribution</h3>
        <p className="text-xs text-brand/70">Graded attempts per 10-point band.</p>
        <div
          className="mt-3 flex h-40 items-end gap-1.5"
          role="img"
          aria-label="Score distribution histogram"
        >
          {data.score_distribution.map((b) => (
            <div key={b.from} className="flex flex-1 flex-col items-center gap-1">
              <span className="text-[10px] tabular-nums text-brand/70">{b.count || ""}</span>
              <div
                className="w-full bg-brand/70"
                style={{ height: `${(b.count / maxBucket) * 100}%`, minHeight: b.count ? 4 : 0 }}
              />
              <span className="text-[10px] tabular-nums text-brand/70">{b.from}</span>
            </div>
          ))}
        </div>
      </section>

      {hardest.length > 0 && !data.small_sample && (
        <section>
          <h3 className="font-serif text-lg">Most missed</h3>
          <ul className="mt-2 list-disc pl-5 text-sm">
            {hardest.map((q) => (
              <li key={q.question_id}>
                Question {q.position}: {q.accuracy_percent}% correct
              </li>
            ))}
          </ul>
        </section>
      )}

      <section>
        <h3 className="font-serif text-lg">Questions</h3>
        <div className="mt-3 space-y-3">
          {data.questions.map((q) => (
            <details key={q.question_id} className="border border-brand/10 bg-white/60">
              <summary className="flex cursor-pointer flex-wrap items-center gap-3 px-4 py-3 text-sm">
                <span className="font-serif tabular-nums">Q{q.position}</span>
                <span className="min-w-0 flex-1 truncate">{q.body.split("\n")[0]}</span>
                <span className="tabular-nums text-brand/70">
                  {q.accuracy_percent != null
                    ? `${q.accuracy_percent}% correct`
                    : "awaiting marking"}
                </span>
                <span className="text-xs tabular-nums text-brand/70">
                  avg {q.average_awarded}/{q.marks}
                </span>
              </summary>
              <div className="border-t border-brand/10 px-4 py-3 text-sm">
                <RichContent text={q.body} className="text-brand/80" />
                <p className="mt-3 text-xs text-brand/70 tabular-nums">
                  {q.responses} responses · {q.correct} correct · {q.partial} partial ·{" "}
                  {q.incorrect} incorrect · {q.unanswered} unanswered
                  {q.pending ? ` · ${q.pending} awaiting marking` : ""}
                </p>
                {q.option_counts && q.option_counts.length > 0 && (
                  <ul className="mt-3 space-y-1.5">
                    {q.option_counts.map((o) => (
                      <li key={o.option_id} className="flex items-center gap-3">
                        <span className="w-28 shrink-0 text-xs tabular-nums text-brand/70">
                          {o.count} chose{o.is_correct ? " ✓" : ""}
                        </span>
                        <div className="h-2 flex-1 bg-brand/10">
                          <div
                            className={`h-2 ${o.is_correct ? "bg-emerald-600" : "bg-brand/50"}`}
                            style={{ width: `${q.responses ? (o.count / q.responses) * 100 : 0}%` }}
                          />
                        </div>
                        <span className="w-1/2 truncate text-xs">{o.body}</span>
                      </li>
                    ))}
                  </ul>
                )}
                {q.common_wrong_answers && q.common_wrong_answers.length > 0 && (
                  <p className="mt-3 text-xs text-brand/70">
                    Common wrong answers:{" "}
                    {q.common_wrong_answers.map((w) => `"${w.answer}" (${w.count})`).join(", ")}
                  </p>
                )}
              </div>
            </details>
          ))}
        </div>
      </section>
    </div>
  );
}

import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ClipboardList, FileText } from "lucide-react";
import { useState } from "react";

import { AppShell, EmptyState, SectionHeading } from "@/components/layout/app-shell";
import { listMyAssignments } from "@/lib/api/assignments";
import { listMyQuizzes } from "@/lib/api/quizzes";

export const Route = createFileRoute("/_authenticated/student/assessments/")({
  component: AssessmentsHub,
});

function AssessmentsHub() {
  const [tab, setTab] = useState<"quizzes" | "assignments">("quizzes");

  return (
    <AppShell eyebrow="Assessments" title="Quizzes & assignments.">
      <div className="flex gap-2 mb-8">
        <TabButton active={tab === "quizzes"} onClick={() => setTab("quizzes")}>
          Quizzes
        </TabButton>
        <TabButton active={tab === "assignments"} onClick={() => setTab("assignments")}>
          Assignments
        </TabButton>
      </div>

      {tab === "quizzes" ? <QuizzesPanel /> : <AssignmentsPanel />}
    </AppShell>
  );
}

function QuizzesPanel() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["my-quizzes"],
    queryFn: () => listMyQuizzes({ limit: 50 }),
  });

  if (isError) return <ErrorBox message={(error as Error)?.message} onRetry={() => refetch()} />;
  if (isLoading) return <ListSkeleton />;
  if (!data || data.data.length === 0)
    return (
      <EmptyState
        icon={ClipboardList}
        title="No quizzes yet"
        description="Quizzes appear here once your courses publish them."
      />
    );

  return (
    <>
      <SectionHeading title={`${data.meta.total} quiz${data.meta.total === 1 ? "" : "zes"}`} />
      <ul className="divide-y divide-brand/10 border-y border-brand/10">
        {data.data.map((q) => (
          <li key={q.id}>
            <Link
              to="/student/assessments/$quizId"
              params={{ quizId: q.id }}
              className="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-6 py-5 px-2 hover:bg-brand/[0.02] transition-colors"
            >
              <div className="flex-1 min-w-0">
                {q.course_title && <p className="eyebrow text-brand/40 mb-1">{q.course_title}</p>}
                <p className="font-serif text-lg leading-snug">{q.title}</p>
                <p className="mt-1 text-xs text-brand/50">
                  {q.total_questions} questions · {q.total_points} pts · pass {q.passing_score}%
                  {q.due_at && ` · due ${new Date(q.due_at).toLocaleDateString()}`}
                </p>
              </div>
              <StatusPill status={q.status ?? "not_started"} score={q.best_score ?? undefined} />
            </Link>
          </li>
        ))}
      </ul>
    </>
  );
}

function AssignmentsPanel() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["my-assignments"],
    queryFn: () => listMyAssignments({ limit: 50 }),
  });

  if (isError) return <ErrorBox message={(error as Error)?.message} onRetry={() => refetch()} />;
  if (isLoading) return <ListSkeleton />;
  if (!data || data.data.length === 0)
    return (
      <EmptyState
        icon={FileText}
        title="No assignments yet"
        description="Assignments appear here once your courses publish them."
      />
    );

  return (
    <>
      <SectionHeading title={`${data.meta.total} assignment${data.meta.total === 1 ? "" : "s"}`} />
      <ul className="divide-y divide-brand/10 border-y border-brand/10">
        {data.data.map((a) => (
          <li key={a.id}>
            <Link
              to="/student/assignments/$assignmentId"
              params={{ assignmentId: a.id }}
              className="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-6 py-5 px-2 hover:bg-brand/[0.02] transition-colors"
            >
              <div className="flex-1 min-w-0">
                {a.course_title && <p className="eyebrow text-brand/40 mb-1">{a.course_title}</p>}
                <p className="font-serif text-lg leading-snug">{a.title}</p>
                <p className="mt-1 text-xs text-brand/50">
                  {a.total_points} pts
                  {a.due_at && ` · due ${new Date(a.due_at).toLocaleString()}`}
                  {typeof a.grade === "number" && ` · grade ${a.grade}/${a.total_points}`}
                </p>
              </div>
              <StatusPill status={a.status} />
            </Link>
          </li>
        ))}
      </ul>
    </>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`px-5 py-2 text-sm font-medium transition-colors ${
        active
          ? "bg-brand text-white"
          : "border border-brand/15 text-brand/70 hover:text-brand hover:bg-brand/[0.03]"
      }`}
    >
      {children}
    </button>
  );
}

function StatusPill({ status, score }: { status: string; score?: number }) {
  const map: Record<string, { label: string; cls: string }> = {
    not_started: { label: "Not started", cls: "text-brand/55 border-brand/15" },
    in_progress: { label: "In progress", cls: "text-accent border-accent/40" },
    completed: { label: "Completed", cls: "text-brand border-brand/30" },
    passed: { label: "Passed", cls: "text-emerald-700 border-emerald-300 bg-emerald-50" },
    failed: { label: "Failed", cls: "text-destructive border-destructive/30 bg-destructive/5" },
    submitted: { label: "Submitted", cls: "text-brand border-brand/30" },
    graded: { label: "Graded", cls: "text-emerald-700 border-emerald-300 bg-emerald-50" },
    revision_requested: { label: "Revise", cls: "text-amber-700 border-amber-300 bg-amber-50" },
    late: { label: "Late", cls: "text-amber-700 border-amber-300 bg-amber-50" },
    missed: { label: "Missed", cls: "text-destructive border-destructive/30 bg-destructive/5" },
  };
  const m = map[status] ?? map.not_started;
  return (
    <span
      className={`inline-flex items-center px-3 py-1.5 text-[11px] font-medium border ${m.cls}`}
    >
      {m.label}
      {typeof score === "number" && ` · ${score}%`}
    </span>
  );
}

function ListSkeleton() {
  return (
    <div className="space-y-3">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="h-20 border border-brand/10 bg-white/30 animate-pulse" />
      ))}
    </div>
  );
}

function ErrorBox({ message, onRetry }: { message?: string; onRetry: () => void }) {
  return (
    <div className="border border-destructive/20 bg-destructive/5 p-6 text-sm">
      <p className="font-medium text-destructive">Couldn't load</p>
      <p className="mt-1 text-brand/60">{message}</p>
      <button onClick={onRetry} className="mt-3 px-4 py-2 bg-brand text-white text-xs">
        Try again
      </button>
    </div>
  );
}

import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeft, RotateCcw } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { AppShell } from "@/components/layout/app-shell";
import { getSubmission, gradeSubmission } from "@/lib/api/submissions";

export const Route = createFileRoute(
  "/_authenticated/teacher/assignments/$assignmentId/submissions/$submissionId",
)({
  component: Page,
});

function Page() {
  const { assignmentId, submissionId } = Route.useParams();
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["submission", submissionId],
    queryFn: () => getSubmission(assignmentId, submissionId),
  });
  const [score, setScore] = useState<number>(0);
  const [feedback, setFeedback] = useState("");

  useEffect(() => {
    if (data) {
      if (data.score != null) setScore(data.score);
      if ((data as any).feedback) setFeedback((data as any).feedback);
    }
  }, [data]);

  const grade = useMutation({
    mutationFn: () => gradeSubmission(assignmentId, submissionId, { score, feedback }),
    onSuccess: () => {
      toast.success("Saved grade");
      qc.invalidateQueries({ queryKey: ["submission", submissionId] });
      qc.invalidateQueries({ queryKey: ["submissions", assignmentId] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const requestRevision = useMutation({
    mutationFn: () =>
      gradeSubmission(assignmentId, submissionId, {
        score,
        feedback: feedback || "Revision requested.",
      }),
    onSuccess: () => {
      toast.success("Returned for revision");
      qc.invalidateQueries({ queryKey: ["submission", submissionId] });
      qc.invalidateQueries({ queryKey: ["submissions", assignmentId] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <AppShell eyebrow="Submission" title={data?.student.full_name ?? "Submission"}>
      <Link
        to="/teacher/assignments/$assignmentId/submissions"
        params={{ assignmentId }}
        className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-6"
      >
        <ArrowLeft className="h-3.5 w-3.5" /> All submissions
      </Link>

      {isLoading && <div className="h-40 border border-brand/10 bg-white/30 animate-pulse" />}

      {data && (
        <div className="grid lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 space-y-6">
            <div className="flex flex-wrap gap-4 text-xs text-brand/60">
              <span>Submitted {new Date(data.submitted_at).toLocaleString()}</span>
              <span className="eyebrow text-brand/55">{data.status}</span>
            </div>

            <div>
              <h3 className="font-serif text-lg mb-3">Response</h3>
              <div className="border border-brand/10 bg-white/40 p-5 text-sm whitespace-pre-wrap min-h-[160px]">
                {data.text || <span className="text-brand/45">No text response</span>}
              </div>
            </div>

            {data.files && data.files.length > 0 && (
              <div>
                <h4 className="font-serif text-base mb-2">Attachments</h4>
                <ul className="space-y-2">
                  {data.files.map((f) => (
                    <li key={f.url}>
                      <a
                        href={f.url}
                        target="_blank"
                        rel="noreferrer"
                        className="text-accent hover:underline text-sm"
                      >
                        {f.name}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>

          <div>
            <h3 className="font-serif text-lg mb-3">Grade</h3>
            <div className="border border-brand/10 bg-white/40 p-5 space-y-3">
              <label className="block">
                <span className="text-xs eyebrow text-brand/45">Score</span>
                <input
                  type="number"
                  value={score}
                  onChange={(e) => setScore(Number(e.target.value))}
                  className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
                />
              </label>
              <label className="block">
                <span className="text-xs eyebrow text-brand/45">Feedback</span>
                <textarea
                  rows={6}
                  value={feedback}
                  onChange={(e) => setFeedback(e.target.value)}
                  placeholder="Comments for the student…"
                  className="mt-1 w-full border border-brand/15 bg-white p-3 text-sm"
                />
              </label>
              <button
                onClick={() => grade.mutate()}
                disabled={grade.isPending}
                className="bg-brand text-white px-6 py-2 text-sm w-full disabled:opacity-50"
              >
                {grade.isPending ? "Saving…" : "Save grade"}
              </button>
              <button
                onClick={() => {
                  if (!feedback.trim()) {
                    toast.error("Add feedback explaining what to revise.");
                    return;
                  }
                  requestRevision.mutate();
                }}
                disabled={requestRevision.isPending}
                className="inline-flex w-full items-center justify-center gap-2 border border-brand/15 px-6 py-2 text-sm hover:bg-brand/[0.03] disabled:opacity-50"
              >
                <RotateCcw className="h-3.5 w-3.5" />
                {requestRevision.isPending ? "Returning…" : "Return for revision"}
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}

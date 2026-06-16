import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Calendar, FileText, Paperclip, X } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import {
  getAssignment,
  submitAssignment,
  type AssignmentAttachment,
  type SubmissionPayload,
} from "@/lib/api/assignments";

export const Route = createFileRoute("/_authenticated/student/assignments/$assignmentId")({
  component: AssignmentDetail,
});

type Draft = {
  text: string;
  attachments: AssignmentAttachment[];
};

function AssignmentDetail() {
  const { assignmentId } = Route.useParams();
  const qc = useQueryClient();

  const {
    data: assignment,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["assignment", assignmentId],
    queryFn: () => getAssignment(assignmentId),
  });

  const [draft, setDraft] = useState<Draft>({ text: "", attachments: [] });

  useEffect(() => {
    if (assignment?.submission) {
      setDraft({
        text: assignment.submission.text ?? "",
        attachments: assignment.submission.attachments ?? [],
      });
    }
  }, [assignment?.submission]);

  const submitMutation = useMutation({
    mutationFn: (payload: SubmissionPayload) => submitAssignment(assignmentId, payload),
    onSuccess: () => {
      toast.success("Submission received.");
      qc.invalidateQueries({ queryKey: ["assignment", assignmentId] });
      qc.invalidateQueries({ queryKey: ["my-assignments"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not submit"),
  });

  if (isLoading) {
    return (
      <AppShell eyebrow="Assignment" title="Loading…">
        <div className="h-40 bg-white/30 border border-brand/10 animate-pulse" />
      </AppShell>
    );
  }
  if (isError || !assignment) {
    return (
      <AppShell eyebrow="Assignment" title="Couldn't load this assignment">
        <p className="text-sm text-brand/55">{(error as Error)?.message}</p>
        <button onClick={() => refetch()} className="mt-4 px-4 py-2 bg-brand text-white text-xs">
          Try again
        </button>
      </AppShell>
    );
  }

  const submission = assignment.submission;
  const canSubmit =
    !submission ||
    submission.revision_requested ||
    (assignment.allow_resubmission && submission.status !== "graded");
  const isOverdue =
    assignment.due_at && !submission && new Date(assignment.due_at).getTime() < Date.now();

  const handleSubmit = () => {
    if (!draft.text.trim() && draft.attachments.length === 0) {
      toast.error("Add written work or at least one attachment.");
      return;
    }
    submitMutation.mutate({
      text: draft.text.trim() || undefined,
      attachments: draft.attachments.map((a) => ({
        filename: a.filename,
        url: a.url,
        size_bytes: a.size_bytes,
        mime_type: a.mime_type,
      })),
    });
  };

  return (
    <AppShell eyebrow={assignment.course_title ?? "Assignment"} title={assignment.title}>
      <div className="grid lg:grid-cols-[1fr_320px] gap-10">
        <article>
          {/* Meta strip */}
          <div className="flex flex-wrap gap-4 text-xs text-brand/60 mb-8">
            <span className="inline-flex items-center gap-1.5">
              <FileText className="h-3.5 w-3.5" />
              {assignment.total_points} points
            </span>
            {assignment.due_at && (
              <span
                className={`inline-flex items-center gap-1.5 ${
                  isOverdue ? "text-destructive font-medium" : ""
                }`}
              >
                <Calendar className="h-3.5 w-3.5" />
                Due {new Date(assignment.due_at).toLocaleString()}
                {isOverdue && " · overdue"}
              </span>
            )}
            {assignment.late_penalty_percent ? (
              <span className="text-amber-700">
                Late penalty: {assignment.late_penalty_percent}%
              </span>
            ) : null}
          </div>

          {/* Brief */}
          <section className="mb-10">
            <h2 className="font-serif text-2xl mb-3">Brief</h2>
            {assignment.instructions_html ? (
              <div
                className="prose prose-sm max-w-none text-brand/80 leading-relaxed"
                dangerouslySetInnerHTML={{ __html: assignment.instructions_html }}
              />
            ) : (
              <p className="text-brand/75 leading-relaxed whitespace-pre-line">
                {assignment.brief}
              </p>
            )}
          </section>

          {/* Rubric */}
          {assignment.rubric && assignment.rubric.criteria.length > 0 && (
            <section className="mb-10">
              <h2 className="font-serif text-2xl mb-4">Rubric</h2>
              <div className="border border-brand/10 divide-y divide-brand/10">
                {assignment.rubric.criteria.map((c) => (
                  <div
                    key={c.id}
                    className="p-4 flex items-start justify-between gap-4 bg-white/40"
                  >
                    <div className="min-w-0">
                      <p className="font-serif text-base">{c.title}</p>
                      {c.description && (
                        <p className="text-xs text-brand/55 mt-1">{c.description}</p>
                      )}
                    </div>
                    <p className="text-sm font-medium text-brand/70 whitespace-nowrap">
                      {c.points} pts
                    </p>
                  </div>
                ))}
              </div>
            </section>
          )}

          {/* Resources */}
          {assignment.resources && assignment.resources.length > 0 && (
            <section className="mb-10">
              <h2 className="font-serif text-2xl mb-3">Resources</h2>
              <ul className="space-y-2">
                {assignment.resources.map((r) => (
                  <li key={r.id}>
                    <a
                      href={r.url}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-2 text-sm text-accent hover:underline underline-offset-4"
                    >
                      <Paperclip className="h-3.5 w-3.5" />
                      {r.filename}
                    </a>
                  </li>
                ))}
              </ul>
            </section>
          )}

          {/* Existing graded submission */}
          {submission && submission.status === "graded" && (
            <section className="mb-10 border border-emerald-200 bg-emerald-50 p-6">
              <p className="eyebrow text-emerald-700">Graded</p>
              <p className="mt-2 font-serif text-3xl">
                {submission.grade} / {assignment.total_points}
              </p>
              {submission.feedback && (
                <p className="mt-3 text-sm text-emerald-900 whitespace-pre-line">
                  {submission.feedback}
                </p>
              )}
              {submission.graded_by?.full_name && (
                <p className="mt-3 text-xs text-emerald-700/80">
                  Graded by {submission.graded_by.full_name}
                </p>
              )}
            </section>
          )}

          {submission?.revision_requested && (
            <section className="mb-10 border border-amber-200 bg-amber-50 p-6">
              <p className="eyebrow text-amber-700">Revision requested</p>
              {submission.feedback && (
                <p className="mt-2 text-sm text-amber-900 whitespace-pre-line">
                  {submission.feedback}
                </p>
              )}
              {submission.resubmission_deadline && (
                <p className="mt-3 text-xs text-amber-700/80">
                  Resubmit by {new Date(submission.resubmission_deadline).toLocaleString()}
                </p>
              )}
            </section>
          )}

          {/* Submission form */}
          {canSubmit && (
            <section>
              <h2 className="font-serif text-2xl mb-4">
                {submission ? "Revise & resubmit" : "Your submission"}
              </h2>
              <textarea
                value={draft.text}
                onChange={(e) => setDraft((d) => ({ ...d, text: e.target.value }))}
                rows={10}
                maxLength={20000}
                placeholder="Write your response…"
                className="w-full p-4 bg-white border border-brand/15 focus:border-brand/40 focus:outline-none text-sm leading-relaxed font-sans"
              />
              <p className="mt-1 text-xs text-brand/45 text-right">
                {draft.text.length.toLocaleString()} / 20,000
              </p>

              <AttachmentField
                attachments={draft.attachments}
                onAdd={(att) => setDraft((d) => ({ ...d, attachments: [...d.attachments, att] }))}
                onRemove={(idx) =>
                  setDraft((d) => ({
                    ...d,
                    attachments: d.attachments.filter((_, i) => i !== idx),
                  }))
                }
              />

              <div className="mt-6 flex items-center gap-4">
                <button
                  onClick={handleSubmit}
                  disabled={submitMutation.isPending}
                  className="bg-brand text-white px-6 py-3 text-sm font-medium hover:bg-brand/90 disabled:opacity-60"
                >
                  {submitMutation.isPending
                    ? "Submitting…"
                    : submission
                      ? "Resubmit"
                      : "Submit assignment"}
                </button>
                <Link
                  to="/student/assessments"
                  className="text-xs text-brand/55 hover:text-brand underline underline-offset-4"
                >
                  Save & exit
                </Link>
              </div>
            </section>
          )}

          {/* Read-only submitted state */}
          {submission && !canSubmit && submission.status !== "graded" && (
            <section className="border border-brand/10 bg-white/50 p-6">
              <p className="eyebrow text-brand/55">Submitted</p>
              <p className="mt-2 text-sm text-brand/65">
                Submitted on {new Date(submission.submitted_at).toLocaleString()}. Awaiting
                feedback.
              </p>
            </section>
          )}
        </article>

        {/* Sidebar */}
        <aside className="lg:sticky lg:top-8 lg:self-start">
          <div className="border border-brand/10 bg-white/50 p-5">
            <p className="eyebrow text-brand/40 mb-3">Status</p>
            <p className="font-serif text-xl capitalize">
              {assignment.status.replaceAll("_", " ")}
            </p>
            {submission && (
              <div className="mt-5 pt-5 border-t border-brand/10 space-y-2 text-xs text-brand/60">
                <p>Submitted {new Date(submission.submitted_at).toLocaleDateString()}</p>
                {submission.attachments.length > 0 && (
                  <p>{submission.attachments.length} attached file(s)</p>
                )}
              </div>
            )}
          </div>
        </aside>
      </div>
    </AppShell>
  );
}

function AttachmentField({
  attachments,
  onAdd,
  onRemove,
}: {
  attachments: AssignmentAttachment[];
  onAdd: (a: AssignmentAttachment) => void;
  onRemove: (idx: number) => void;
}) {
  const [filename, setFilename] = useState("");
  const [url, setUrl] = useState("");

  const add = () => {
    if (!filename.trim() || !url.trim()) {
      toast.error("Add a filename and a link to the file.");
      return;
    }
    try {
      // basic URL sanity check
      new URL(url);
    } catch {
      toast.error("Attachment URL must be a valid link.");
      return;
    }
    onAdd({
      id: crypto.randomUUID(),
      filename: filename.trim().slice(0, 200),
      url: url.trim(),
    });
    setFilename("");
    setUrl("");
  };

  return (
    <div className="mt-6">
      <p className="eyebrow text-brand/45 mb-3">Attachments</p>
      {attachments.length > 0 && (
        <ul className="space-y-2 mb-4">
          {attachments.map((a, i) => (
            <li
              key={a.id}
              className="flex items-center justify-between border border-brand/10 bg-white px-4 py-2.5 text-sm"
            >
              <a
                href={a.url}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-2 text-accent hover:underline underline-offset-4 truncate"
              >
                <Paperclip className="h-3.5 w-3.5 flex-shrink-0" />
                <span className="truncate">{a.filename}</span>
              </a>
              <button
                onClick={() => onRemove(i)}
                className="text-brand/40 hover:text-destructive ml-2"
                aria-label="Remove attachment"
              >
                <X className="h-4 w-4" />
              </button>
            </li>
          ))}
        </ul>
      )}
      <div className="grid sm:grid-cols-[1fr_2fr_auto] gap-2">
        <input
          value={filename}
          onChange={(e) => setFilename(e.target.value)}
          placeholder="File name"
          maxLength={200}
          className="p-3 bg-white border border-brand/15 focus:border-brand/40 focus:outline-none text-sm"
        />
        <input
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://…"
          maxLength={2000}
          className="p-3 bg-white border border-brand/15 focus:border-brand/40 focus:outline-none text-sm"
        />
        <button
          type="button"
          onClick={add}
          className="px-5 py-3 border border-brand/15 text-sm font-medium hover:bg-brand/[0.03]"
        >
          Add link
        </button>
      </div>
      <p className="mt-2 text-[11px] text-brand/45">
        Paste a shareable link (Drive, Dropbox, etc.). Native file upload arrives with the storage
        slice.
      </p>
    </div>
  );
}

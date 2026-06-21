import { useEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import {
  getTeacherAssignment,
  updateTeacherAssignment,
  type TeacherAssignmentInput,
} from "@/lib/api/assignments";

export const Route = createFileRoute(
  "/_authenticated/teacher/courses/$courseId/assignments/$assignmentId/edit",
)({
  component: EditAssignmentPage,
});

function EditAssignmentPage() {
  const { courseId, assignmentId } = Route.useParams();
  const navigate = useNavigate();
  const [form, setForm] = useState<TeacherAssignmentInput>({
    title: "",
    description: "",
    due_at: "",
    submission_type: "both",
    max_file_size_mb: 50,
    allow_late_submission: false,
    total_marks: 100,
  });

  const assignment = useQuery({
    queryKey: ["teacher-assignment", assignmentId],
    queryFn: () => getTeacherAssignment(assignmentId),
  });

  useEffect(() => {
    if (!assignment.data) return;
    setForm({
      title: assignment.data.title,
      description: assignment.data.description,
      due_at: toDateTimeLocal(assignment.data.due_at),
      submission_type: assignment.data.submission_type,
      max_file_size_mb: assignment.data.max_file_size_mb,
      allow_late_submission: assignment.data.allow_late_submission,
      total_marks: assignment.data.total_marks,
    });
  }, [assignment.data]);

  const update = useMutation({
    mutationFn: () =>
      updateTeacherAssignment(assignmentId, {
        ...form,
        due_at: form.due_at ? new Date(form.due_at).toISOString() : "",
      }),
    onSuccess: () => {
      toast.success("Assignment updated");
      navigate({ to: "/teacher/assignments" });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <AppShell eyebrow="Assignments" title="Edit assignment">
      <div className="mb-5 flex flex-wrap gap-3">
        <Link
          to="/teacher/courses/$courseId/content"
          params={{ courseId }}
          className="border border-brand/15 px-4 py-2 text-xs text-brand/65 hover:bg-brand/[0.04]"
        >
          Course content
        </Link>
        <Link
          to="/teacher/assignments"
          className="border border-brand/15 px-4 py-2 text-xs text-brand/65 hover:bg-brand/[0.04]"
        >
          Gradebook
        </Link>
      </div>

      {assignment.isLoading && (
        <div className="h-72 max-w-3xl border border-brand/10 bg-white/40 animate-pulse" />
      )}
      {assignment.isError && (
        <QueryErrorPanel
          error={assignment.error}
          variant="compact"
          message={(assignment.error as Error)?.message ?? "Failed to load assignment."}
        />
      )}

      {assignment.data && (
        <div className="max-w-3xl border border-brand/10 bg-white/50 p-6 space-y-5">
          <label className="block">
            <span className="eyebrow text-brand/45">Title</span>
            <input
              value={form.title}
              onChange={(event) => setForm({ ...form, title: event.target.value })}
              className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
          <label className="block">
            <span className="eyebrow text-brand/45">Instructions</span>
            <textarea
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
              rows={6}
              className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            />
          </label>
          <div className="grid sm:grid-cols-2 gap-4">
            <label className="block">
              <span className="eyebrow text-brand/45">Due date</span>
              <input
                type="datetime-local"
                value={form.due_at}
                onChange={(event) => setForm({ ...form, due_at: event.target.value })}
                className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              />
            </label>
            <label className="block">
              <span className="eyebrow text-brand/45">Submission type</span>
              <select
                value={form.submission_type}
                onChange={(event) =>
                  setForm({
                    ...form,
                    submission_type: event.target.value as "text" | "file" | "both",
                  })
                }
                className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              >
                <option value="both">Text and file</option>
                <option value="text">Text only</option>
                <option value="file">File only</option>
              </select>
            </label>
            <NumberField
              label="Max file size MB"
              value={form.max_file_size_mb}
              onChange={(value) => setForm({ ...form, max_file_size_mb: value })}
            />
            <NumberField
              label="Total marks"
              value={form.total_marks}
              onChange={(value) => setForm({ ...form, total_marks: value })}
            />
          </div>
          <label className="inline-flex items-center gap-2 text-sm text-brand/70">
            <input
              type="checkbox"
              checked={form.allow_late_submission}
              onChange={(event) =>
                setForm({ ...form, allow_late_submission: event.target.checked })
              }
            />
            Allow late submissions
          </label>
          <div>
            <button
              onClick={() => update.mutate()}
              disabled={!form.title.trim() || !form.description.trim() || update.isPending}
              className="bg-brand text-white px-6 py-3 text-sm disabled:opacity-50"
            >
              {update.isPending ? "Saving..." : "Save assignment"}
            </button>
          </div>
        </div>
      )}
    </AppShell>
  );
}

function NumberField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (value: number) => void;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/45">{label}</span>
      <input
        type="number"
        min={0}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
        className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
      />
    </label>
  );
}

function toDateTimeLocal(value?: string | null) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const offsetMs = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
}

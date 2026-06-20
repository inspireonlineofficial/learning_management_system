import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { useState } from "react";

import { AppShell } from "@/components/layout/app-shell";
import { createTeacherAssignment } from "@/lib/api/assignments";
import { getTeacherCourse } from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/assignments/new")({
  component: CreateAssignmentPage,
});

function CreateAssignmentPage() {
  const { courseId } = Route.useParams();
  const navigate = useNavigate();
  const course = useQuery({
    queryKey: ["teacher-course", courseId, "assignment-new"],
    queryFn: () => getTeacherCourse(courseId),
  });
  const [form, setForm] = useState({
    title: "",
    description: "",
    due_at: "",
    submission_type: "both" as const,
    max_file_size_mb: 50,
    allow_late_submission: false,
    total_marks: 100,
  });

  const create = useMutation({
    mutationFn: () =>
      createTeacherAssignment(courseId, {
        ...form,
        due_at: form.due_at ? new Date(form.due_at).toISOString() : "",
      }),
    onSuccess: () => {
      toast.success("Assignment created");
      navigate({ to: "/teacher/assignments" });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <AppShell
      eyebrow="Assignments"
      title={course.data ? `Create assignment for ${course.data.title}` : "Create an assignment"}
    >
      <Link
        to="/teacher/courses/$courseId/builder"
        params={{ courseId }}
        className="mb-5 inline-flex text-xs text-brand/55 hover:text-brand"
      >
        Back to course builder
      </Link>
      <div className="max-w-3xl border border-brand/10 bg-white/50 p-6 space-y-5">
        <div className="border border-brand/10 bg-white px-3 py-2 text-sm text-brand/65">
          Course: <span className="font-medium text-brand">{course.data?.title ?? courseId}</span>
        </div>
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
            onChange={(event) => setForm({ ...form, allow_late_submission: event.target.checked })}
          />
          Allow late submissions
        </label>
        <div>
          <button
            onClick={() => create.mutate()}
            disabled={
              !form.title.trim() || !form.description.trim() || !form.due_at || create.isPending
            }
            className="bg-brand text-white px-6 py-3 text-sm disabled:opacity-50"
          >
            {create.isPending ? "Creating..." : "Create assignment"}
          </button>
        </div>
      </div>
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

import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { useState } from "react";

import { AppShell } from "@/components/layout/app-shell";
import { createQuiz } from "@/lib/api/teacher-quizzes";
import { listMyTaughtCourses } from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/quiz-builder/new")({
  component: CreateQuizPage,
});

function CreateQuizPage() {
  const navigate = useNavigate();
  const [form, setForm] = useState({
    title: "",
    course_id: "",
    passing_score: 70,
    time_limit_minutes: 30,
    attempts_allowed: 1,
    is_free: true,
    is_published: true,
  });

  const courses = useQuery({
    queryKey: ["taught-courses", "quiz-new"],
    queryFn: () => listMyTaughtCourses({ limit: 100 }),
  });

  const create = useMutation({
    mutationFn: () =>
      createQuiz({
        title: form.title.trim(),
        course_id: form.course_id || undefined,
        passing_score: Number(form.passing_score) || 70,
        time_limit_minutes: Number(form.time_limit_minutes) || 30,
        attempts_allowed: Number(form.attempts_allowed) || 1,
        is_free: form.is_free,
        is_published: form.is_published,
      }),
    onSuccess: (quiz) => {
      toast.success("Quiz draft created");
      navigate({ to: "/teacher/quiz-builder/$quizId/edit", params: { quizId: quiz.id } });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <AppShell eyebrow="Quiz builder" title="Create a quiz draft">
      <div className="max-w-2xl border border-brand/10 bg-white/50 p-6 space-y-5">
        <label className="block">
          <span className="eyebrow text-brand/45">Title</span>
          <input
            value={form.title}
            onChange={(event) => setForm({ ...form, title: event.target.value })}
            className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
          />
        </label>

        <label className="block">
          <span className="eyebrow text-brand/45">Course</span>
          <select
            value={form.course_id}
            onChange={(event) => setForm({ ...form, course_id: event.target.value })}
            className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
          >
            <option value="">Select a course</option>
            {courses.data?.data.map((course) => (
              <option key={course.id} value={course.id}>
                {course.title}
              </option>
            ))}
          </select>
        </label>

        <div className="grid sm:grid-cols-3 gap-4">
          <NumberField
            label="Passing %"
            value={form.passing_score}
            min={0}
            max={100}
            onChange={(value) => setForm({ ...form, passing_score: value })}
          />
          <NumberField
            label="Timer minutes"
            value={form.time_limit_minutes}
            min={1}
            onChange={(value) => setForm({ ...form, time_limit_minutes: value })}
          />
          <NumberField
            label="Attempts"
            value={form.attempts_allowed}
            min={1}
            max={10}
            onChange={(value) => setForm({ ...form, attempts_allowed: value })}
          />
        </div>

        <div className="flex flex-wrap gap-5 text-sm text-brand/70">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={form.is_free}
              onChange={(event) => setForm({ ...form, is_free: event.target.checked })}
            />
            Free access
          </label>
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={form.is_published}
              onChange={(event) => setForm({ ...form, is_published: event.target.checked })}
            />
            Published
          </label>
        </div>

        <button
          onClick={() => create.mutate()}
          disabled={!form.title.trim() || !form.course_id || create.isPending}
          className="bg-brand text-white px-6 py-3 text-sm disabled:opacity-50"
        >
          {create.isPending ? "Creating..." : "Create and edit questions"}
        </button>
      </div>
    </AppShell>
  );
}

function NumberField({
  label,
  value,
  min,
  max,
  onChange,
}: {
  label: string;
  value: number;
  min?: number;
  max?: number;
  onChange: (value: number) => void;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/45">{label}</span>
      <input
        type="number"
        min={min}
        max={max}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
        className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
      />
    </label>
  );
}

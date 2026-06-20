import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { createCourse } from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/courses/new")({
  component: CreateCoursePage,
});

export function CreateCoursePage() {
  const navigate = useNavigate();
  const [form, setForm] = useState({ title: "", subtitle: "", level: "beginner" as const });
  const mut = useMutation({
    mutationFn: () => createCourse(form),
    onSuccess: (c: any) => {
      toast.success("Course created");
      navigate({ to: "/teacher/courses/$courseId/builder", params: { courseId: c.id } });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <AppShell eyebrow="Course" title="Create a course">
      <div className="max-w-xl space-y-4">
        <label className="block">
          <span className="text-xs eyebrow text-brand/45">Title</span>
          <input
            className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            value={form.title}
            onChange={(e) => setForm({ ...form, title: e.target.value })}
          />
        </label>
        <label className="block">
          <span className="text-xs eyebrow text-brand/45">Subtitle</span>
          <input
            className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            value={form.subtitle}
            onChange={(e) => setForm({ ...form, subtitle: e.target.value })}
          />
        </label>
        <button
          onClick={() => mut.mutate()}
          disabled={mut.isPending || !form.title}
          className="bg-brand text-white px-6 py-2 text-sm disabled:opacity-50"
        >
          {mut.isPending ? "Creating…" : "Create course"}
        </button>
      </div>
    </AppShell>
  );
}

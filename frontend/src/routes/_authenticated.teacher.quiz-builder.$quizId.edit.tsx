import { useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { AppShell, SectionHeading } from "@/components/layout/app-shell";
import {
  createQuestion,
  deleteQuestion,
  getTeacherQuiz,
  updateQuestion,
  updateQuiz,
  type TeacherQuestionInput,
} from "@/lib/api/teacher-quizzes";
import type { QuestionType } from "@/lib/api/quizzes";

export const Route = createFileRoute("/_authenticated/teacher/quiz-builder/$quizId/edit")({
  component: Page,
});

function Page() {
  const { quizId } = Route.useParams();
  const qc = useQueryClient();

  const q = useQuery({
    queryKey: ["teacher-quiz", quizId],
    queryFn: () => getTeacherQuiz(quizId),
  });

  if (q.isLoading) {
    return (
      <AppShell>
        <div className="h-10 w-1/2 bg-brand/10 animate-pulse mb-4" />
        <div className="h-64 bg-brand/5 animate-pulse" />
      </AppShell>
    );
  }
  if (q.isError || !q.data) {
    return (
      <AppShell title="Quiz unavailable">
        <p className="text-sm text-brand/60">{(q.error as Error)?.message ?? "Not found"}</p>
      </AppShell>
    );
  }

  const data = q.data;
  const invalidate = () => qc.invalidateQueries({ queryKey: ["teacher-quiz", quizId] });

  return (
    <AppShell>
      <Link
        to="/teacher/quiz-builder"
        className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-6"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        All quizzes
      </Link>

      <h1 className="font-serif text-4xl lg:text-5xl mb-2">{data.title}</h1>
      {data.description && <p className="text-brand/65 mb-6">{data.description}</p>}

      <QuizMeta quizId={quizId} initial={data} onSaved={invalidate} />

      <SectionHeading
        title="Questions"
        action={<AddQuestion quizId={quizId} onAdded={invalidate} />}
      />

      {(data.questions ?? []).length === 0 ? (
        <p className="text-sm text-brand/55 border border-dashed border-brand/15 p-8 text-center">
          No questions yet.
        </p>
      ) : (
        <ul className="space-y-3">
          {data.questions.map((qq, i) => (
            <QuestionCard key={qq.id} index={i} question={qq} onChange={invalidate} />
          ))}
        </ul>
      )}
    </AppShell>
  );
}

function QuizMeta({
  quizId,
  initial,
  onSaved,
}: {
  quizId: string;
  initial: {
    title: string;
    description?: string;
    passing_score: number;
    time_limit_minutes?: number | null;
    attempts_allowed?: number | null;
    is_free?: boolean;
    is_published?: boolean;
  };
  onSaved: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    title: initial.title,
    description: initial.description ?? "",
    passing_score: initial.passing_score,
    time_limit_minutes: initial.time_limit_minutes ?? 0,
    attempts_allowed: initial.attempts_allowed ?? 0,
    is_free: initial.is_free ?? true,
    is_published: initial.is_published ?? true,
  });

  const save = useMutation({
    mutationFn: () =>
      updateQuiz(quizId, {
        title: form.title.trim(),
        description: form.description || undefined,
        passing_score: Number(form.passing_score) || 0,
        time_limit_minutes: Number(form.time_limit_minutes) || null,
        attempts_allowed: Number(form.attempts_allowed) || null,
        is_free: form.is_free,
        is_published: form.is_published,
      }),
    onSuccess: () => {
      toast.success("Saved");
      setOpen(false);
      onSaved();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="mb-8 text-xs text-accent hover:text-brand underline"
      >
        Edit quiz settings
      </button>
    );
  }

  return (
    <div className="border border-brand/15 bg-white/50 p-4 mb-8 space-y-3 max-w-2xl">
      <input
        value={form.title}
        onChange={(e) => setForm({ ...form, title: e.target.value })}
        className="w-full px-3 py-2 text-sm border border-brand/15 bg-white"
      />
      <textarea
        value={form.description}
        onChange={(e) => setForm({ ...form, description: e.target.value })}
        rows={3}
        placeholder="Description"
        className="w-full px-3 py-2 text-sm border border-brand/15 bg-white"
      />
      <div className="grid grid-cols-3 gap-3">
        <label className="text-xs text-brand/65 block">
          Passing %
          <input
            type="number"
            min={0}
            max={100}
            value={form.passing_score}
            onChange={(e) => setForm({ ...form, passing_score: Number(e.target.value) })}
            className="w-full mt-1 px-2 py-1.5 border border-brand/15 bg-white"
          />
        </label>
        <label className="text-xs text-brand/65 block">
          Time limit (min, 0 = none)
          <input
            type="number"
            min={0}
            value={form.time_limit_minutes}
            onChange={(e) => setForm({ ...form, time_limit_minutes: Number(e.target.value) })}
            className="w-full mt-1 px-2 py-1.5 border border-brand/15 bg-white"
          />
        </label>
        <label className="text-xs text-brand/65 block">
          Attempts allowed (0 = ∞)
          <input
            type="number"
            min={0}
            value={form.attempts_allowed}
            onChange={(e) => setForm({ ...form, attempts_allowed: Number(e.target.value) })}
            className="w-full mt-1 px-2 py-1.5 border border-brand/15 bg-white"
          />
        </label>
      </div>
      <div className="flex flex-wrap gap-5 text-xs text-brand/70">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={form.is_free}
            onChange={(e) => setForm({ ...form, is_free: e.target.checked })}
          />
          Free access
        </label>
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={form.is_published}
            onChange={(e) => setForm({ ...form, is_published: e.target.checked })}
          />
          Published
        </label>
      </div>
      <div className="flex justify-end gap-2">
        <button onClick={() => setOpen(false)} className="px-3 py-2 text-xs text-brand/70">
          Cancel
        </button>
        <button
          onClick={() => save.mutate()}
          disabled={save.isPending}
          className="px-4 py-2 bg-brand text-white text-xs disabled:opacity-50"
        >
          {save.isPending ? "Saving…" : "Save"}
        </button>
      </div>
    </div>
  );
}

function AddQuestion({ quizId, onAdded }: { quizId: string; onAdded: () => void }) {
  const create = useMutation({
    mutationFn: (type: QuestionType) =>
      createQuestion(quizId, {
        type,
        prompt: "New question",
        points: 1,
        options:
          type === "single_choice" || type === "multi_select"
            ? [
                { text: "Option 1", is_correct: true },
                { text: "Option 2", is_correct: false },
              ]
            : type === "true_false"
              ? [
                  { text: "True", is_correct: true },
                  { text: "False", is_correct: false },
                ]
              : undefined,
        correct_text: type === "short_answer" ? "" : undefined,
      }),
    onSuccess: () => {
      toast.success("Question added");
      onAdded();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <div className="flex gap-2">
      {(["single_choice", "multi_select", "true_false", "short_answer"] as QuestionType[]).map(
        (t) => (
          <button
            key={t}
            onClick={() => create.mutate(t)}
            disabled={create.isPending}
            className="inline-flex items-center gap-1 px-3 py-2 text-[11px] font-medium border border-brand/15 hover:bg-brand/[0.03] capitalize disabled:opacity-50"
          >
            <Plus className="h-3 w-3" />
            {t.replace("_", " ")}
          </button>
        ),
      )}
    </div>
  );
}

type FullQuestion = {
  id: string;
  type: QuestionType;
  prompt: string;
  content_type?: "text" | "image" | "text_image";
  image_url?: string;
  points: number;
  is_required?: boolean;
  options?: {
    id: string;
    text: string;
    content_type?: "text" | "image" | "text_image";
    image_url?: string;
    is_correct?: boolean;
  }[];
  correct_text?: string;
  explanation?: string;
};

function QuestionCard({
  index,
  question,
  onChange,
}: {
  index: number;
  question: FullQuestion;
  onChange: () => void;
}) {
  const [form, setForm] = useState<TeacherQuestionInput>({
    type: question.type,
    prompt: question.prompt,
    content_type: question.content_type,
    image_url: question.image_url ?? "",
    points: question.points,
    is_required: question.is_required ?? true,
    options:
      question.options?.map((o) => ({
        id: o.id,
        text: o.text,
        content_type: o.content_type,
        image_url: o.image_url ?? "",
        is_correct: !!o.is_correct,
      })) ?? [],
    correct_text: question.correct_text ?? "",
    explanation: question.explanation ?? "",
  });

  const save = useMutation({
    mutationFn: () => updateQuestion(question.id, form),
    onSuccess: () => {
      toast.success("Saved");
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const remove = useMutation({
    mutationFn: () => deleteQuestion(question.id),
    onSuccess: () => {
      toast.success("Deleted");
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const setOption = (
    i: number,
    patch: Partial<{
      text: string;
      image_url: string;
      content_type: "text" | "image" | "text_image";
      is_correct: boolean;
    }>,
  ) => {
    const opts = [...(form.options ?? [])];
    opts[i] = { ...opts[i], ...patch };
    if (patch.is_correct && form.type === "single_choice") {
      opts.forEach((o, j) => (o.is_correct = j === i));
    }
    setForm({ ...form, options: opts });
  };

  return (
    <li className="border border-brand/15 bg-white/50 p-4 space-y-3">
      <div className="flex items-start gap-3">
        <span className="eyebrow text-brand/45 mt-2">
          Q{index + 1} · {form.type.replace("_", " ")}
        </span>
        <textarea
          value={form.prompt}
          onChange={(e) => setForm({ ...form, prompt: e.target.value })}
          rows={2}
          className="flex-1 px-3 py-2 text-sm border border-brand/15 bg-white resize-y"
        />
        <label className="text-xs text-brand/65 flex items-center gap-1">
          Points
          <input
            type="number"
            min={0}
            value={form.points ?? 1}
            onChange={(e) => setForm({ ...form, points: Number(e.target.value) })}
            className="w-16 px-2 py-1 border border-brand/15 bg-white"
          />
        </label>
        <label className="text-xs text-brand/65 flex items-center gap-1 mt-2">
          <input
            type="checkbox"
            checked={form.is_required ?? true}
            onChange={(e) => setForm({ ...form, is_required: e.target.checked })}
            className="accent-brand"
          />
          Required
        </label>
        <button
          onClick={() => {
            if (confirm("Delete question?")) remove.mutate();
          }}
          className="p-1.5 text-brand/45 hover:text-destructive"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="grid md:grid-cols-[1fr_180px] gap-3 pl-0 md:pl-[96px]">
        <input
          value={form.image_url ?? ""}
          onChange={(e) => setForm({ ...form, image_url: e.target.value })}
          placeholder="Question image URL"
          className="px-3 py-2 text-sm border border-brand/15 bg-white"
        />
        <select
          value={form.content_type ?? "text"}
          onChange={(e) =>
            setForm({
              ...form,
              content_type: e.target.value as "text" | "image" | "text_image",
            })
          }
          className="px-3 py-2 text-sm border border-brand/15 bg-white"
        >
          <option value="text">Text</option>
          <option value="image">Image</option>
          <option value="text_image">Text + image</option>
        </select>
      </div>

      {(form.type === "single_choice" ||
        form.type === "multi_select" ||
        form.type === "true_false") && (
        <div className="space-y-2 pl-4">
          {(form.options ?? []).map((opt, i) => (
            <div key={i} className="grid md:grid-cols-[auto_1fr_1fr_150px_auto] items-center gap-2">
              <input
                type={
                  form.type === "single_choice" || form.type === "true_false" ? "radio" : "checkbox"
                }
                checked={!!opt.is_correct}
                onChange={(e) => setOption(i, { is_correct: e.target.checked })}
                name={`q-${question.id}`}
              />
              <input
                value={opt.text}
                onChange={(e) => setOption(i, { text: e.target.value })}
                disabled={form.type === "true_false"}
                className="flex-1 px-3 py-1.5 text-sm border border-brand/15 bg-white disabled:bg-brand/[0.03]"
              />
              <input
                value={opt.image_url ?? ""}
                onChange={(e) => setOption(i, { image_url: e.target.value })}
                placeholder="Option image URL"
                disabled={form.type === "true_false"}
                className="px-3 py-1.5 text-sm border border-brand/15 bg-white disabled:bg-brand/[0.03]"
              />
              <select
                value={opt.content_type ?? "text"}
                onChange={(e) =>
                  setOption(i, {
                    content_type: e.target.value as "text" | "image" | "text_image",
                  })
                }
                disabled={form.type === "true_false"}
                className="px-2 py-1.5 text-xs border border-brand/15 bg-white disabled:bg-brand/[0.03]"
              >
                <option value="text">Text</option>
                <option value="image">Image</option>
                <option value="text_image">Text + image</option>
              </select>
              {form.type !== "true_false" && (
                <button
                  onClick={() => {
                    const opts = [...(form.options ?? [])];
                    opts.splice(i, 1);
                    setForm({ ...form, options: opts });
                  }}
                  className="p-1 text-brand/45 hover:text-destructive"
                >
                  <Trash2 className="h-3 w-3" />
                </button>
              )}
            </div>
          ))}
          {form.type !== "true_false" && (
            <button
              onClick={() =>
                setForm({
                  ...form,
                  options: [...(form.options ?? []), { text: "", is_correct: false }],
                })
              }
              className="text-xs text-accent hover:text-brand inline-flex items-center gap-1"
            >
              <Plus className="h-3 w-3" />
              Add option
            </button>
          )}
        </div>
      )}

      {form.type === "short_answer" && (
        <input
          value={form.correct_text ?? ""}
          onChange={(e) => setForm({ ...form, correct_text: e.target.value })}
          placeholder="Expected answer (case-insensitive match)"
          className="w-full px-3 py-2 text-sm border border-brand/15 bg-white"
        />
      )}

      <textarea
        value={form.explanation ?? ""}
        onChange={(e) => setForm({ ...form, explanation: e.target.value })}
        rows={2}
        placeholder="Explanation (shown after submit)"
        className="w-full px-3 py-2 text-xs border border-brand/15 bg-white resize-y"
      />

      <div className="flex justify-end">
        <button
          onClick={() => save.mutate()}
          disabled={save.isPending}
          className="px-4 py-2 bg-brand text-white text-xs disabled:opacity-50"
        >
          {save.isPending ? "Saving…" : "Save question"}
        </button>
      </div>
    </li>
  );
}

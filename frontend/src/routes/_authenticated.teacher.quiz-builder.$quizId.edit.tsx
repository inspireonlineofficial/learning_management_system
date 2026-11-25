import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowDown,
  ArrowUp,
  BarChart3,
  ChevronDown,
  Copy,
  Eye,
  Plus,
  Save,
  Trash2,
  Users,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import { QuestionView } from "@/components/assessment/question-view";
import { QUESTION_TYPE_LABEL } from "@/lib/assessment";
import { RichContent } from "@/components/assessment/rich-content";
import { AppShell } from "@/components/layout/app-shell";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import type { AnswerMap, QuestionType, QuizQuestion } from "@/lib/api/quizzes";
import { uploadLessonFile } from "@/lib/api/teacher";
import {
  createQuestion,
  deleteQuestion,
  duplicateQuiz,
  getTeacherQuiz,
  questionToInput,
  updateQuestion,
  updateQuiz,
  type TeacherQuestion,
  type TeacherQuestionInput,
  type TeacherQuiz,
  type TeacherQuizSettings,
} from "@/lib/api/teacher-quizzes";

export const Route = createFileRoute("/_authenticated/teacher/quiz-builder/$quizId/edit")({
  component: Page,
});

const LIFECYCLE_STYLE: Record<TeacherQuiz["lifecycle"], string> = {
  draft: "border-brand/20 text-brand/70",
  scheduled: "border-sky-600/30 text-sky-800 bg-sky-50",
  open: "border-emerald-600/30 text-emerald-800 bg-emerald-50",
  closed: "border-brand/20 text-brand/70 bg-brand/[0.03]",
};

function Page() {
  const { quizId } = Route.useParams();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [previewOpen, setPreviewOpen] = useState(false);

  const q = useQuery({ queryKey: ["teacher-quiz", quizId], queryFn: () => getTeacherQuiz(quizId) });
  const refresh = () => qc.invalidateQueries({ queryKey: ["teacher-quiz", quizId] });

  const duplicate = useMutation({
    mutationFn: () => duplicateQuiz(quizId),
    onSuccess: (copy) => {
      toast.success("Copy created as a draft");
      qc.invalidateQueries({ queryKey: ["teacher-quizzes"] });
      navigate({ to: "/teacher/quiz-builder/$quizId/edit", params: { quizId: copy.id } });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (q.isLoading) {
    return (
      <AppShell eyebrow="Quiz builder" title="Loading…">
        <div className="h-40 animate-pulse border border-brand/10 bg-white/30" />
      </AppShell>
    );
  }
  if (q.isError || !q.data) {
    return (
      <AppShell eyebrow="Quiz builder" title="Couldn't load this quiz">
        <p className="text-sm text-brand/70">{(q.error as Error)?.message}</p>
      </AppShell>
    );
  }
  const quiz = q.data;

  return (
    <AppShell eyebrow="Quiz builder" title={quiz.title}>
      <div className="flex flex-wrap items-center gap-2">
        <span
          className={`border px-2.5 py-1 text-xs capitalize ${LIFECYCLE_STYLE[quiz.lifecycle]}`}
        >
          {quiz.lifecycle}
        </span>
        <span className="text-xs text-brand/70 tabular-nums">
          {quiz.total_questions} questions · {quiz.total_points} marks ·{" "}
          {quiz.mode === "practice" ? "Practice" : "Exam"}
        </span>
        <div className="ml-auto flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => setPreviewOpen(true)}
            disabled={quiz.questions.length === 0}
            className="inline-flex items-center gap-2 border border-brand/15 px-3 py-2 text-xs hover:bg-brand/[0.03] disabled:opacity-50"
          >
            <Eye className="h-3.5 w-3.5" aria-hidden /> Preview as student
          </button>
          <Link
            to="/teacher/quiz-builder/$quizId/attempts"
            params={{ quizId }}
            className="inline-flex items-center gap-2 border border-brand/15 px-3 py-2 text-xs hover:bg-brand/[0.03]"
          >
            <Users className="h-3.5 w-3.5" aria-hidden /> Attempts
            {quiz.pending_grading_count > 0 && (
              <span className="bg-accent-ink px-1.5 text-[10px] text-white tabular-nums">
                {quiz.pending_grading_count} to mark
              </span>
            )}
          </Link>
          <Link
            to="/teacher/quiz-builder/$quizId/attempts"
            params={{ quizId }}
            search={{ tab: "analytics" }}
            className="inline-flex items-center gap-2 border border-brand/15 px-3 py-2 text-xs hover:bg-brand/[0.03]"
          >
            <BarChart3 className="h-3.5 w-3.5" aria-hidden /> Analytics
          </Link>
          <button
            type="button"
            onClick={() => duplicate.mutate()}
            disabled={duplicate.isPending}
            className="inline-flex items-center gap-2 border border-brand/15 px-3 py-2 text-xs hover:bg-brand/[0.03] disabled:opacity-50"
          >
            <Copy className="h-3.5 w-3.5" aria-hidden /> Duplicate
          </button>
        </div>
      </div>

      {quiz.attempt_count > 0 && (
        <p className="mt-5 border-l-2 border-amber-500 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          {quiz.attempt_count} attempt{quiz.attempt_count === 1 ? "" : "s"} exist. Edits apply to
          new and in-progress attempts; submitted attempts keep the questions and marks they were
          graded with.
        </p>
      )}

      <SettingsPanel quiz={quiz} onSaved={refresh} />
      <QuestionsPanel quiz={quiz} onChanged={refresh} />

      <PreviewDialog open={previewOpen} onOpenChange={setPreviewOpen} quiz={quiz} />
    </AppShell>
  );
}

// ---------------------------------------------------------------------------
// Settings

function toLocalInput(iso: string | null) {
  if (!iso) return "";
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function fromLocalInput(value: string) {
  return value ? new Date(value).toISOString() : null;
}

function settingsOf(quiz: TeacherQuiz): TeacherQuizSettings {
  return {
    lesson_id: quiz.lesson_id,
    title: quiz.title,
    description: quiz.description,
    instructions: quiz.instructions,
    mode: quiz.mode,
    time_limit_minutes: quiz.time_limit_minutes,
    passing_score: quiz.passing_score,
    attempts_allowed: quiz.attempts_allowed,
    is_free: quiz.is_free,
    is_published: quiz.is_published,
    shuffle_questions: quiz.shuffle_questions,
    shuffle_options: quiz.shuffle_options,
    partial_credit: quiz.partial_credit,
    show_answers_after_submission: quiz.show_answers_after_submission,
    available_from: quiz.available_from,
    available_until: quiz.available_until,
    calculator_policy: quiz.calculator_policy,
    reference_material: quiz.reference_material,
  };
}

const inputClass =
  "mt-1.5 w-full border border-brand/15 bg-white px-3 py-2 text-sm focus:border-brand focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/30";

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block text-sm">
      <span className="font-medium text-brand/80">{label}</span>
      {children}
      {hint && <span className="mt-1 block text-xs text-brand/70">{hint}</span>}
    </label>
  );
}

function Toggle({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string;
  hint?: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex items-start gap-3 text-sm">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-1 h-4 w-4 accent-brand"
      />
      <span>
        <span className="font-medium text-brand/80">{label}</span>
        {hint && <span className="block text-xs text-brand/70">{hint}</span>}
      </span>
    </label>
  );
}

function SettingsPanel({ quiz, onSaved }: { quiz: TeacherQuiz; onSaved: () => void }) {
  const [form, setForm] = useState<TeacherQuizSettings>(() => settingsOf(quiz));
  useEffect(() => setForm(settingsOf(quiz)), [quiz]);
  const set = <K extends keyof TeacherQuizSettings>(key: K, value: TeacherQuizSettings[K]) =>
    setForm((f) => ({ ...f, [key]: value }));
  const dirty = JSON.stringify(form) !== JSON.stringify(settingsOf(quiz));

  const save = useMutation({
    mutationFn: (next: TeacherQuizSettings) => updateQuiz(quiz.id, next, quiz.questions),
    onSuccess: (_r, next) => {
      toast.success(next.is_published && !quiz.is_published ? "Quiz published" : "Settings saved");
      onSaved();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const windowInvalid =
    form.available_from &&
    form.available_until &&
    new Date(form.available_until) <= new Date(form.available_from);

  return (
    <section className="mt-8 border border-brand/10 bg-white/50" aria-labelledby="settings-heading">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-brand/10 px-5 py-4">
        <h2 id="settings-heading" className="font-serif text-xl">
          Settings
        </h2>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            disabled={!dirty || save.isPending || Boolean(windowInvalid)}
            onClick={() => save.mutate(form)}
            className="inline-flex items-center gap-2 border border-brand/20 px-4 py-2 text-sm hover:border-brand disabled:opacity-40"
          >
            <Save className="h-4 w-4" aria-hidden /> {save.isPending ? "Saving…" : "Save settings"}
          </button>
          {quiz.is_published ? (
            <button
              type="button"
              disabled={save.isPending}
              onClick={() => {
                if (
                  confirm(
                    "Unpublish? Students will no longer see this quiz. Existing attempts are kept.",
                  )
                )
                  save.mutate({ ...form, is_published: false });
              }}
              className="border border-brand/20 px-4 py-2 text-sm hover:border-destructive hover:text-destructive"
            >
              Unpublish
            </button>
          ) : (
            <button
              type="button"
              disabled={save.isPending || quiz.questions.length === 0 || Boolean(windowInvalid)}
              title={quiz.questions.length === 0 ? "Add at least one question first" : undefined}
              onClick={() => save.mutate({ ...form, is_published: true })}
              className="bg-brand px-4 py-2 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-40"
            >
              {form.available_from && new Date(form.available_from) > new Date()
                ? "Schedule"
                : "Publish"}
            </button>
          )}
        </div>
      </div>

      <div className="grid gap-5 p-5 md:grid-cols-2">
        <Field label="Title">
          <input
            className={inputClass}
            value={form.title}
            maxLength={255}
            onChange={(e) => set("title", e.target.value)}
          />
        </Field>
        <Field
          label="Mode"
          hint="Practice shows hints and lets students check answers. Exam reveals nothing until submission."
        >
          <select
            className={inputClass}
            value={form.mode}
            onChange={(e) => set("mode", e.target.value as "exam" | "practice")}
          >
            <option value="exam">Exam</option>
            <option value="practice">Practice</option>
          </select>
        </Field>
        <Field label="Time limit (minutes)" hint="Leave empty or 0 for untimed.">
          <input
            className={inputClass}
            type="number"
            min={0}
            max={1440}
            value={form.time_limit_minutes ?? ""}
            onChange={(e) =>
              set("time_limit_minutes", e.target.value === "" ? null : Number(e.target.value))
            }
          />
        </Field>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Passing score (%)">
            <input
              className={inputClass}
              type="number"
              min={0}
              max={100}
              value={form.passing_score ?? 60}
              onChange={(e) => set("passing_score", Number(e.target.value))}
            />
          </Field>
          <Field label="Attempts allowed" hint="Empty = unlimited">
            <input
              className={inputClass}
              type="number"
              min={1}
              value={form.attempts_allowed ?? ""}
              onChange={(e) =>
                set(
                  "attempts_allowed",
                  e.target.value === "" ? null : Math.max(1, Number(e.target.value)),
                )
              }
            />
          </Field>
        </div>
        <div className="md:col-span-2">
          <Field
            label="Description"
            hint="Shown on the assessment page. Supports $math$, \ce{chemistry}, **bold**."
          >
            <textarea
              className={inputClass}
              rows={2}
              value={form.description ?? ""}
              onChange={(e) => set("description", e.target.value)}
            />
          </Field>
        </div>
        <div className="md:col-span-2">
          <Field label="Instructions" hint="Shown before starting and at the top of the exam.">
            <textarea
              className={inputClass}
              rows={3}
              value={form.instructions ?? ""}
              onChange={(e) => set("instructions", e.target.value)}
            />
          </Field>
        </div>
      </div>

      <details className="border-t border-brand/10 px-5 py-4">
        <summary className="flex cursor-pointer items-center gap-2 font-medium text-sm">
          <ChevronDown className="h-4 w-4" aria-hidden /> Availability and exam rules
        </summary>
        <div className="mt-5 grid gap-5 md:grid-cols-2">
          <Field label="Opens" hint="Empty = as soon as it is published.">
            <input
              className={inputClass}
              type="datetime-local"
              value={toLocalInput(form.available_from ?? null)}
              onChange={(e) => set("available_from", fromLocalInput(e.target.value))}
            />
          </Field>
          <Field label="Closes" hint="Timed attempts in progress end at this time.">
            <input
              className={inputClass}
              type="datetime-local"
              value={toLocalInput(form.available_until ?? null)}
              onChange={(e) => set("available_until", fromLocalInput(e.target.value))}
            />
          </Field>
          {windowInvalid && (
            <p className="md:col-span-2 text-sm text-destructive">
              The closing time must be after the opening time.
            </p>
          )}
          <Field label="Calculator">
            <select
              className={inputClass}
              value={form.calculator_policy}
              onChange={(e) =>
                set("calculator_policy", e.target.value as TeacherQuizSettings["calculator_policy"])
              }
            >
              <option value="none">Not permitted</option>
              <option value="basic">Basic calculator</option>
              <option value="scientific">Scientific calculator</option>
              <option value="graphing">Graphing calculator</option>
            </select>
          </Field>
          <Field label="Permitted resources" hint="e.g. periodic table, formula sheet">
            <input
              className={inputClass}
              value={form.reference_material ?? ""}
              onChange={(e) => set("reference_material", e.target.value)}
            />
          </Field>
          <div className="grid gap-3 md:col-span-2 md:grid-cols-2">
            <Toggle
              label="Shuffle question order"
              hint="Each attempt keeps its own fixed order."
              checked={Boolean(form.shuffle_questions)}
              onChange={(v) => set("shuffle_questions", v)}
            />
            <Toggle
              label="Shuffle answer options"
              checked={Boolean(form.shuffle_options)}
              onChange={(v) => set("shuffle_options", v)}
            />
            <Toggle
              label="Partial credit on multiple-select"
              hint="(right picks − wrong picks) ÷ correct options, never below zero."
              checked={Boolean(form.partial_credit)}
              onChange={(v) => set("partial_credit", v)}
            />
            <Toggle
              label="Show correct answers after submission"
              hint="Otherwise students see only their own answers and total score."
              checked={Boolean(form.show_answers_after_submission)}
              onChange={(v) => set("show_answers_after_submission", v)}
            />
            <Toggle
              label="Free for non-enrolled students"
              checked={Boolean(form.is_free)}
              onChange={(v) => set("is_free", v)}
            />
          </div>
        </div>
      </details>
    </section>
  );
}

// ---------------------------------------------------------------------------
// Questions

const NEW_QUESTION: Record<QuestionType, TeacherQuestionInput> = {
  single_choice: {
    type: "single_choice",
    prompt: "New question",
    points: 1,
    options: [{ text: "Option A", is_correct: true }, { text: "Option B" }],
  },
  multi_select: {
    type: "multi_select",
    prompt: "New question",
    points: 1,
    options: [
      { text: "Option A", is_correct: true },
      { text: "Option B", is_correct: true },
      { text: "Option C" },
    ],
  },
  true_false: {
    type: "true_false",
    prompt: "New statement",
    points: 1,
    options: [{ text: "True", is_correct: true }, { text: "False" }],
  },
  short_answer: { type: "short_answer", prompt: "New question", points: 1, correct_text: "answer" },
  numeric: {
    type: "numeric",
    prompt: "New numerical question",
    points: 1,
    numeric_answer: 0,
    tolerance: 0,
    tolerance_type: "absolute",
  },
  long_answer: { type: "long_answer", prompt: "New written question", points: 5 },
};

function QuestionsPanel({ quiz, onChanged }: { quiz: TeacherQuiz; onChanged: () => void }) {
  const [openId, setOpenId] = useState<string | null>(null);
  const [newType, setNewType] = useState<QuestionType>("single_choice");

  const add = useMutation({
    mutationFn: (type: QuestionType) => createQuestion(quiz.id, NEW_QUESTION[type]),
    onSuccess: (q) => {
      onChanged();
      setOpenId(q.id);
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const reorder = useMutation({
    mutationFn: (questions: TeacherQuestion[]) => updateQuiz(quiz.id, settingsOf(quiz), questions),
    onSuccess: onChanged,
    onError: (e: Error) => toast.error(e.message),
  });

  const move = (index: number, delta: number) => {
    const next = quiz.questions.slice();
    const target = index + delta;
    if (target < 0 || target >= next.length) return;
    [next[index], next[target]] = [next[target], next[index]];
    reorder.mutate(next);
  };

  return (
    <section className="mt-8" aria-labelledby="questions-heading">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <h2 id="questions-heading" className="font-serif text-2xl">
          Questions
        </h2>
        <div className="flex items-center gap-2">
          <label className="sr-only" htmlFor="new-question-type">
            Question type
          </label>
          <select
            id="new-question-type"
            value={newType}
            onChange={(e) => setNewType(e.target.value as QuestionType)}
            className="border border-brand/15 bg-white px-3 py-2 text-sm"
          >
            {(Object.keys(QUESTION_TYPE_LABEL) as QuestionType[]).map((t) => (
              <option key={t} value={t}>
                {QUESTION_TYPE_LABEL[t]}
              </option>
            ))}
          </select>
          <button
            type="button"
            onClick={() => add.mutate(newType)}
            disabled={add.isPending}
            className="inline-flex items-center gap-2 bg-brand px-4 py-2 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-50"
          >
            <Plus className="h-4 w-4" aria-hidden /> Add question
          </button>
        </div>
      </div>

      {quiz.questions.length === 0 ? (
        <p className="mt-5 border border-dashed border-brand/20 p-8 text-center text-sm text-brand/70">
          No questions yet. Choose a type and add the first question.
        </p>
      ) : (
        <ol className="mt-5 space-y-3">
          {quiz.questions.map((question, index) => (
            <li key={question.id}>
              <QuestionEditor
                question={question}
                index={index}
                total={quiz.questions.length}
                open={openId === question.id}
                onToggle={() => setOpenId((id) => (id === question.id ? null : question.id))}
                onMove={(delta) => move(index, delta)}
                onChanged={onChanged}
                busy={reorder.isPending}
              />
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

function validateDraft(d: TeacherQuestionInput): string | null {
  if (!d.prompt.trim() && !d.image_url) return "Add question text or an image.";
  if (["single_choice", "multi_select", "true_false"].includes(d.type)) {
    const opts = d.options ?? [];
    if (opts.length < 2) return "Add at least two options.";
    if (opts.some((o) => !o.text.trim() && !o.image_url))
      return "Every option needs text or an image.";
    const correct = opts.filter((o) => o.is_correct).length;
    if (correct === 0) return "Mark the correct option.";
    if (d.type !== "multi_select" && correct !== 1)
      return "Single-answer questions need exactly one correct option.";
  }
  if (d.type === "short_answer" && !(d.correct_text ?? "").trim())
    return "Add at least one accepted answer.";
  if (d.type === "numeric" && (d.numeric_answer == null || Number.isNaN(d.numeric_answer)))
    return "Enter the correct numerical answer.";
  if ((d.points ?? 0) <= 0) return "Marks must be greater than zero.";
  return null;
}

function QuestionEditor({
  question,
  index,
  total,
  open,
  onToggle,
  onMove,
  onChanged,
  busy,
}: {
  question: TeacherQuestion;
  index: number;
  total: number;
  open: boolean;
  onToggle: () => void;
  onMove: (delta: number) => void;
  onChanged: () => void;
  busy: boolean;
}) {
  const [draft, setDraft] = useState<TeacherQuestionInput>(() => questionToInput(question));
  useEffect(() => setDraft(questionToInput(question)), [question]);
  const [previewAnswer, setPreviewAnswer] = useState<AnswerMap>({});
  const dirty = JSON.stringify(draft) !== JSON.stringify(questionToInput(question));
  const error = validateDraft(draft);
  const set = <K extends keyof TeacherQuestionInput>(key: K, value: TeacherQuestionInput[K]) =>
    setDraft((d) => ({ ...d, [key]: value }));

  const save = useMutation({
    mutationFn: () => updateQuestion(question.id, draft),
    onSuccess: () => {
      toast.success(`Question ${index + 1} saved`);
      onChanged();
    },
    onError: (e: Error) => toast.error(e.message),
  });
  const remove = useMutation({
    mutationFn: () => deleteQuestion(question.id),
    onSuccess: () => {
      toast.success("Question deleted");
      onChanged();
    },
    onError: (e: Error) => toast.error(e.message),
  });
  const upload = useMutation({
    mutationFn: async ({ file, option }: { file: File; option?: number }) => ({
      url: (await uploadLessonFile(file)).url,
      option,
    }),
    onSuccess: ({ url, option }) => {
      if (option == null) set("image_url", url);
      else
        setDraft((d) => ({
          ...d,
          options: (d.options ?? []).map((o, i) => (i === option ? { ...o, image_url: url } : o)),
        }));
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const previewQuestion: QuizQuestion = useMemo(
    () => ({
      id: question.id,
      type: draft.type,
      prompt: draft.prompt,
      image_url: draft.image_url,
      points: draft.points ?? 1,
      hint: draft.hint,
      unit: draft.unit,
      max_words: draft.max_words,
      options: (draft.options ?? []).map((o, i) => ({
        id: o.id ?? `new-${i}`,
        text: o.text,
        image_url: o.image_url,
      })),
    }),
    [draft, question.id],
  );

  const changeType = (type: QuestionType) => {
    setDraft((d) => {
      const template = NEW_QUESTION[type];
      const keepOptions =
        ["single_choice", "multi_select"].includes(type) && (d.options?.length ?? 0) >= 2;
      return {
        ...d,
        type,
        options:
          type === "true_false" ? template.options : keepOptions ? d.options : template.options,
        correct_text:
          type === "short_answer" ? d.correct_text || template.correct_text : d.correct_text,
        numeric_answer: type === "numeric" ? (d.numeric_answer ?? 0) : d.numeric_answer,
      };
    });
  };

  return (
    <article className="border border-brand/10 bg-white/60">
      <div className="flex items-center gap-3 px-4 py-3">
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={open}
          className="flex min-w-0 flex-1 items-center gap-3 text-left"
        >
          <span className="grid h-8 w-8 shrink-0 place-items-center border border-brand/15 font-serif tabular-nums">
            {index + 1}
          </span>
          <span className="min-w-0 flex-1">
            <span className="block truncate text-sm">
              {question.prompt.split("\n")[0] || "(image question)"}
            </span>
            <span className="text-xs text-brand/70">
              {QUESTION_TYPE_LABEL[question.type]} · {question.points}{" "}
              {question.points === 1 ? "mark" : "marks"}
              {dirty && <span className="ml-2 text-amber-700">· unsaved changes</span>}
            </span>
          </span>
          <ChevronDown
            className={`h-4 w-4 shrink-0 transition-transform ${open ? "rotate-180" : ""}`}
            aria-hidden
          />
        </button>
        <div className="flex">
          <button
            type="button"
            onClick={() => onMove(-1)}
            disabled={index === 0 || busy}
            className="p-2 text-brand/70 hover:text-brand disabled:opacity-30"
            aria-label={`Move question ${index + 1} up`}
          >
            <ArrowUp className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={() => onMove(1)}
            disabled={index === total - 1 || busy}
            className="p-2 text-brand/70 hover:text-brand disabled:opacity-30"
            aria-label={`Move question ${index + 1} down`}
          >
            <ArrowDown className="h-4 w-4" />
          </button>
        </div>
      </div>

      {open && (
        <div className="grid gap-6 border-t border-brand/10 p-4 xl:grid-cols-2">
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <Field label="Type">
                <select
                  className={inputClass}
                  value={draft.type}
                  onChange={(e) => changeType(e.target.value as QuestionType)}
                >
                  {(Object.keys(QUESTION_TYPE_LABEL) as QuestionType[]).map((t) => (
                    <option key={t} value={t}>
                      {QUESTION_TYPE_LABEL[t]}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="Marks">
                <input
                  className={inputClass}
                  type="number"
                  min={0.5}
                  step={0.5}
                  value={draft.points ?? 1}
                  onChange={(e) => set("points", Number(e.target.value))}
                />
              </Field>
            </div>
            <Field
              label="Question"
              hint="Supports $inline$ and $$display$$ maths, \ce{H2SO4} chemistry, **bold**, and | tables |."
            >
              <textarea
                className={`${inputClass} font-mono text-[13px]`}
                rows={5}
                value={draft.prompt}
                onChange={(e) => set("prompt", e.target.value)}
              />
            </Field>
            <ImageField
              label="Diagram or image (optional)"
              url={draft.image_url}
              uploading={upload.isPending && upload.variables?.option == null}
              onUpload={(file) => upload.mutate({ file })}
              onRemove={() => set("image_url", "")}
            />

            {["single_choice", "multi_select", "true_false"].includes(draft.type) && (
              <fieldset className="space-y-2">
                <legend className="text-sm font-medium text-brand/80">
                  Options{" "}
                  <span className="font-normal text-brand/70">
                    — tick {draft.type === "multi_select" ? "every" : "the"} correct answer
                  </span>
                </legend>
                {(draft.options ?? []).map((o, i) => (
                  <div key={o.id ?? i} className="flex items-start gap-2">
                    <input
                      type={draft.type === "multi_select" ? "checkbox" : "radio"}
                      name={`correct-${question.id}`}
                      checked={Boolean(o.is_correct)}
                      aria-label={`Option ${i + 1} is correct`}
                      onChange={() =>
                        setDraft((d) => ({
                          ...d,
                          options: (d.options ?? []).map((x, j) =>
                            d.type === "multi_select"
                              ? j === i
                                ? { ...x, is_correct: !x.is_correct }
                                : x
                              : { ...x, is_correct: j === i },
                          ),
                        }))
                      }
                      className="mt-3 h-4 w-4 accent-emerald-700"
                    />
                    <div className="min-w-0 flex-1">
                      <input
                        className={`${inputClass} mt-0 font-mono text-[13px]`}
                        value={o.text}
                        disabled={draft.type === "true_false"}
                        aria-label={`Option ${i + 1} text`}
                        onChange={(e) =>
                          setDraft((d) => ({
                            ...d,
                            options: (d.options ?? []).map((x, j) =>
                              j === i ? { ...x, text: e.target.value } : x,
                            ),
                          }))
                        }
                      />
                      {draft.type !== "true_false" && (
                        <ImageField
                          compact
                          label="Option image"
                          url={o.image_url}
                          uploading={upload.isPending && upload.variables?.option === i}
                          onUpload={(file) => upload.mutate({ file, option: i })}
                          onRemove={() =>
                            setDraft((d) => ({
                              ...d,
                              options: (d.options ?? []).map((x, j) =>
                                j === i ? { ...x, image_url: "" } : x,
                              ),
                            }))
                          }
                        />
                      )}
                    </div>
                    {draft.type !== "true_false" && (draft.options?.length ?? 0) > 2 && (
                      <button
                        type="button"
                        onClick={() =>
                          setDraft((d) => ({
                            ...d,
                            options: (d.options ?? []).filter((_, j) => j !== i),
                          }))
                        }
                        className="mt-2 p-1.5 text-brand/70 hover:text-destructive"
                        aria-label={`Remove option ${i + 1}`}
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                ))}
                {draft.type !== "true_false" && (draft.options?.length ?? 0) < 20 && (
                  <button
                    type="button"
                    onClick={() =>
                      setDraft((d) => ({ ...d, options: [...(d.options ?? []), { text: "" }] }))
                    }
                    className="text-sm text-accent-ink hover:underline"
                  >
                    + Add option
                  </button>
                )}
              </fieldset>
            )}

            {draft.type === "short_answer" && (
              <>
                <Field label="Accepted answers" hint="One per line. Matching ignores extra spaces.">
                  <textarea
                    className={`${inputClass} font-mono text-[13px]`}
                    rows={3}
                    value={draft.correct_text ?? ""}
                    onChange={(e) => set("correct_text", e.target.value)}
                  />
                </Field>
                <Toggle
                  label="Case-sensitive"
                  hint="Use for chemical formulae such as CO vs Co."
                  checked={Boolean(draft.case_sensitive)}
                  onChange={(v) => set("case_sensitive", v)}
                />
              </>
            )}

            {draft.type === "numeric" && (
              <div className="grid gap-3 sm:grid-cols-2">
                <Field label="Correct answer" hint="Scientific notation allowed, e.g. 6.022e23">
                  <NumberText
                    value={draft.numeric_answer ?? null}
                    onChange={(v) => set("numeric_answer", v)}
                  />
                </Field>
                <Field
                  label="Unit (optional)"
                  hint="Shown next to the box; students may type it too."
                >
                  <input
                    className={inputClass}
                    value={draft.unit ?? ""}
                    onChange={(e) => set("unit", e.target.value)}
                  />
                </Field>
                <Field label="Tolerance">
                  <input
                    className={inputClass}
                    type="number"
                    min={0}
                    step="any"
                    value={draft.tolerance ?? 0}
                    onChange={(e) => set("tolerance", Number(e.target.value))}
                  />
                </Field>
                <Field label="Tolerance type">
                  <select
                    className={inputClass}
                    value={draft.tolerance_type ?? "absolute"}
                    onChange={(e) =>
                      set("tolerance_type", e.target.value as "absolute" | "percent")
                    }
                  >
                    <option value="absolute">± absolute value</option>
                    <option value="percent">± percent of answer</option>
                  </select>
                </Field>
              </div>
            )}

            {draft.type === "long_answer" && (
              <Field
                label="Word limit (optional)"
                hint="Students see a live word count. Marked manually from the Attempts page."
              >
                <input
                  className={inputClass}
                  type="number"
                  min={0}
                  value={draft.max_words ?? ""}
                  onChange={(e) =>
                    set("max_words", e.target.value === "" ? undefined : Number(e.target.value))
                  }
                />
              </Field>
            )}

            <Field label="Hint (practice mode only)">
              <textarea
                className={inputClass}
                rows={2}
                value={draft.hint ?? ""}
                onChange={(e) => set("hint", e.target.value)}
              />
            </Field>
            <Field
              label="Explanation / worked solution"
              hint="Shown after submission only if answers are released (or on a practice check)."
            >
              <textarea
                className={`${inputClass} font-mono text-[13px]`}
                rows={3}
                value={draft.explanation ?? ""}
                onChange={(e) => set("explanation", e.target.value)}
              />
            </Field>

            {error && dirty && (
              <p className="text-sm text-destructive" role="alert">
                {error}
              </p>
            )}
            <div className="flex flex-wrap gap-2 pt-1">
              <button
                type="button"
                onClick={() => save.mutate()}
                disabled={!dirty || Boolean(error) || save.isPending}
                className="inline-flex items-center gap-2 bg-brand px-4 py-2 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-40"
              >
                <Save className="h-4 w-4" aria-hidden />{" "}
                {save.isPending ? "Saving…" : "Save question"}
              </button>
              {dirty && (
                <button
                  type="button"
                  onClick={() => setDraft(questionToInput(question))}
                  className="border border-brand/15 px-4 py-2 text-sm"
                >
                  Discard changes
                </button>
              )}
              <button
                type="button"
                onClick={() =>
                  confirm("Delete this question? Submitted attempts keep their copy.") &&
                  remove.mutate()
                }
                className="ml-auto inline-flex items-center gap-2 px-3 py-2 text-sm text-destructive hover:bg-destructive/5"
              >
                <Trash2 className="h-4 w-4" aria-hidden /> Delete
              </button>
            </div>
          </div>

          <div className="xl:sticky xl:top-24 xl:self-start">
            <p className="eyebrow mb-3 text-brand/70">Live student preview</p>
            <div className="border border-brand/10 bg-surface p-5">
              <QuestionView
                question={previewQuestion}
                number={index + 1}
                value={previewAnswer[question.id]}
                onChange={(v) => setPreviewAnswer({ [question.id]: v })}
                showHint
              />
              {draft.explanation && (
                <div className="mt-5 border-l-2 border-accent bg-accent/5 px-4 py-3 text-sm">
                  <p className="eyebrow mb-1 text-accent-ink">Explanation (after submission)</p>
                  <RichContent text={draft.explanation} />
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </article>
  );
}

/** Text box for numbers so scientific notation can be typed freely. */
function NumberText({
  value,
  onChange,
}: {
  value: number | null;
  onChange: (v: number | null) => void;
}) {
  const [text, setText] = useState(value == null ? "" : String(value));
  useEffect(() => {
    if (value != null && Number(text) !== value) setText(String(value));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);
  const parsed = text.trim() === "" ? null : Number(text.replace(/−/g, "-").replace(/\s/g, ""));
  return (
    <>
      <input
        className={inputClass}
        inputMode="decimal"
        value={text}
        onChange={(e) => {
          setText(e.target.value);
          const n =
            e.target.value.trim() === ""
              ? null
              : Number(e.target.value.replace(/−/g, "-").replace(/\s/g, ""));
          onChange(n != null && Number.isFinite(n) ? n : null);
        }}
        aria-invalid={parsed != null && !Number.isFinite(parsed)}
      />
      {parsed != null && !Number.isFinite(parsed) && (
        <span className="mt-1 block text-xs text-destructive">Not a number</span>
      )}
    </>
  );
}

function ImageField({
  label,
  url,
  uploading,
  onUpload,
  onRemove,
  compact = false,
}: {
  label: string;
  url?: string;
  uploading: boolean;
  onUpload: (file: File) => void;
  onRemove: () => void;
  compact?: boolean;
}) {
  return (
    <div
      className={`${compact ? "mt-1" : ""} flex flex-wrap items-center gap-3 text-xs text-brand/70`}
    >
      {!compact && <span className="text-sm font-medium text-brand/80">{label}</span>}
      <label className="cursor-pointer border border-brand/15 px-2.5 py-1 hover:bg-brand/[0.03]">
        {uploading ? "Uploading…" : url ? "Replace image" : compact ? "+ image" : "Upload image"}
        <input
          type="file"
          accept="image/png,image/jpeg,image/webp"
          className="sr-only"
          disabled={uploading}
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) onUpload(file);
            e.currentTarget.value = "";
          }}
        />
      </label>
      {url && (
        <>
          <img src={url} alt="" className="h-12 w-auto border border-brand/10" />
          <button type="button" onClick={onRemove} className="text-destructive">
            Remove
          </button>
        </>
      )}
    </div>
  );
}

function PreviewDialog({
  open,
  onOpenChange,
  quiz,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  quiz: TeacherQuiz;
}) {
  const [idx, setIdx] = useState(0);
  const [answers, setAnswers] = useState<AnswerMap>({});
  const question = quiz.questions[Math.min(idx, quiz.questions.length - 1)];
  if (!question) return null;
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="font-serif">Student preview — {quiz.title}</DialogTitle>
        </DialogHeader>
        <p className="text-xs text-brand/70">Answers here are not saved and nothing is graded.</p>
        <div className="bg-surface p-5">
          <QuestionView
            question={question}
            number={idx + 1}
            value={answers[question.id]}
            onChange={(v) => setAnswers((a) => ({ ...a, [question.id]: v }))}
            showHint={quiz.mode === "practice"}
          />
        </div>
        <div className="flex items-center justify-between">
          <button
            type="button"
            disabled={idx === 0}
            onClick={() => setIdx((i) => i - 1)}
            className="border border-brand/15 px-4 py-2 text-sm disabled:opacity-40"
          >
            Previous
          </button>
          <span className="text-xs tabular-nums text-brand/70">
            {idx + 1} / {quiz.questions.length}
          </span>
          <button
            type="button"
            disabled={idx >= quiz.questions.length - 1}
            onClick={() => setIdx((i) => i + 1)}
            className="border border-brand/15 px-4 py-2 text-sm disabled:opacity-40"
          >
            Next
          </button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

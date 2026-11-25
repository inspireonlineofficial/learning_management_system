import { Check, ImagePlus, Lightbulb, X } from "lucide-react";
import { useId, useRef, useState } from "react";

import { RichContent, ZoomableImage } from "@/components/assessment/rich-content";
import { countWords, QUESTION_TYPE_LABEL } from "@/lib/assessment";
import type { AnswerValue, QuizQuestion } from "@/lib/api/quizzes";

/**
 * One question with its answer control. Used by the student exam, the
 * teacher live preview and (read-only) by the review screens, so every view
 * renders questions identically.
 */

const OPTION_LETTERS = "ABCDEFGHIJKLMNOPQRST";

function textOf(value: AnswerValue | undefined): string {
  if (typeof value === "string") return value;
  if (value && !Array.isArray(value)) return value.text ?? "";
  return "";
}

function selectedOf(value: AnswerValue | undefined): string[] {
  if (Array.isArray(value)) return value;
  if (typeof value === "string" && value) return [value];
  return [];
}

export function QuestionView({
  question,
  number,
  value,
  onChange,
  disabled = false,
  showHint = false,
  feedback,
  onUploadImage,
}: {
  question: QuizQuestion;
  number: number;
  value: AnswerValue | undefined;
  onChange: (value: AnswerValue) => void;
  disabled?: boolean;
  showHint?: boolean;
  /** Enables image attachments on written answers; returns the stored URL. */
  onUploadImage?: (file: File) => Promise<string>;
  /** Practice-mode check result for this question. */
  feedback?: { correct: boolean; explanation?: string } | null;
}) {
  const [hintOpen, setHintOpen] = useState(false);
  const headingId = useId();
  const promptId = `${headingId}-prompt`;

  return (
    <section aria-labelledby={headingId} className="min-w-0">
      <header className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <h2 id={headingId} className="font-serif text-xl text-brand">
          Question {number}
        </h2>
        <p className="text-xs text-brand/70">
          <span>{QUESTION_TYPE_LABEL[question.type]}</span>
          <span aria-hidden> · </span>
          <span>
            {question.points} {question.points === 1 ? "mark" : "marks"}
          </span>
        </p>
      </header>

      <div id={promptId} className="mt-4 text-[1.05rem] text-brand">
        <RichContent text={question.prompt} />
        {question.image_url && (
          <div className="mt-4">
            <ZoomableImage src={question.image_url} alt={`Figure for question ${number}`} />
          </div>
        )}
      </div>

      {showHint && question.hint && (
        <div className="mt-4">
          {hintOpen ? (
            <div
              className="border-l-2 border-accent bg-accent/5 px-4 py-3 text-sm text-brand/80"
              role="note"
            >
              <p className="eyebrow text-accent-ink mb-1">Hint</p>
              <RichContent text={question.hint} />
            </div>
          ) : (
            <button
              type="button"
              onClick={() => setHintOpen(true)}
              className="inline-flex items-center gap-2 text-sm text-accent-ink hover:underline underline-offset-4"
            >
              <Lightbulb className="h-4 w-4" aria-hidden /> Show hint
            </button>
          )}
        </div>
      )}

      <div className="mt-6">
        <AnswerControl
          question={question}
          value={value}
          onChange={onChange}
          disabled={disabled}
          describedBy={promptId}
          onUploadImage={onUploadImage}
        />
      </div>

      {feedback && (
        <div
          role="status"
          className={`mt-5 border px-4 py-3 text-sm ${
            feedback.correct
              ? "border-emerald-600/30 bg-emerald-50 text-emerald-900"
              : "border-amber-600/30 bg-amber-50 text-amber-900"
          }`}
        >
          <p className="font-medium">{feedback.correct ? "Correct." : "Not quite."}</p>
          {feedback.explanation && (
            <div className="mt-2 text-brand/80">
              <RichContent text={feedback.explanation} />
            </div>
          )}
        </div>
      )}
    </section>
  );
}

function imageOf(value: AnswerValue | undefined): string {
  if (value && typeof value === "object" && !Array.isArray(value)) return value.image_url ?? "";
  return "";
}

function AttachImage({
  value,
  onChange,
  disabled,
  onUploadImage,
}: {
  value: AnswerValue | undefined;
  onChange: (value: AnswerValue) => void;
  disabled: boolean;
  onUploadImage: (file: File) => Promise<string>;
}) {
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const image = imageOf(value);
  const text = textOf(value);
  return (
    <div className="mt-3">
      {image ? (
        <div className="inline-flex items-start gap-2">
          <img
            src={image}
            alt="Your attached working"
            className="max-h-40 w-auto border border-brand/15 bg-white"
          />
          {!disabled && (
            <button
              type="button"
              onClick={() => onChange({ text, image_url: "" })}
              className="p-1.5 text-brand/70 hover:text-destructive"
              aria-label="Remove attached image"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      ) : (
        !disabled && (
          <button
            type="button"
            disabled={busy}
            onClick={() => input.current?.click()}
            className="inline-flex items-center gap-2 border border-brand/15 px-3 py-2 text-sm text-brand/70 hover:border-brand/40 disabled:opacity-60"
          >
            <ImagePlus className="h-4 w-4" aria-hidden />
            {busy ? "Uploading…" : "Attach a photo of your working"}
          </button>
        )
      )}
      <input
        ref={input}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="sr-only"
        tabIndex={-1}
        onChange={async (e) => {
          const file = e.target.files?.[0];
          e.target.value = "";
          if (!file) return;
          if (file.size > 5 * 1024 * 1024) {
            setError("Images must be 5 MB or smaller.");
            return;
          }
          setBusy(true);
          setError(null);
          try {
            const url = await onUploadImage(file);
            onChange({ text, image_url: url });
          } catch (err) {
            setError((err as Error).message || "Upload failed.");
          } finally {
            setBusy(false);
          }
        }}
      />
      {error && (
        <p role="alert" className="mt-1 text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}

function AnswerControl({
  question,
  value,
  onChange,
  disabled,
  describedBy,
  onUploadImage,
}: {
  question: QuizQuestion;
  value: AnswerValue | undefined;
  onChange: (value: AnswerValue) => void;
  disabled: boolean;
  describedBy: string;
  onUploadImage?: (file: File) => Promise<string>;
}) {
  const inputId = useId();
  switch (question.type) {
    case "single_choice":
    case "true_false":
    case "multi_select": {
      const multi = question.type === "multi_select";
      const selected = selectedOf(value);
      return (
        <fieldset aria-describedby={describedBy} disabled={disabled}>
          <legend className="text-sm text-brand/70 mb-3">
            {multi ? "Select all that apply." : "Select one answer."}
          </legend>
          <div className="grid gap-2.5">
            {(question.options ?? []).map((opt, i) => {
              const active = selected.includes(opt.id);
              return (
                <label
                  key={opt.id}
                  className={`flex items-start gap-3 border px-4 py-3.5 cursor-pointer transition-colors min-h-[3rem] ${
                    active
                      ? "border-brand bg-brand/[0.05]"
                      : "border-brand/15 bg-white hover:border-brand/40"
                  } ${disabled ? "cursor-default opacity-80" : ""} has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-accent`}
                >
                  <input
                    type={multi ? "checkbox" : "radio"}
                    name={`q-${question.id}`}
                    value={opt.id}
                    checked={active}
                    disabled={disabled}
                    onChange={() => {
                      if (multi) {
                        const next = new Set(selected);
                        if (active) next.delete(opt.id);
                        else next.add(opt.id);
                        onChange(Array.from(next));
                      } else {
                        onChange(opt.id);
                      }
                    }}
                    className="sr-only"
                  />
                  <span
                    aria-hidden
                    className={`mt-0.5 grid h-6 w-6 shrink-0 place-items-center border text-xs font-medium ${
                      multi ? "" : "rounded-full"
                    } ${active ? "border-brand bg-brand text-white" : "border-brand/30 text-brand/70"}`}
                  >
                    {active ? <Check className="h-3.5 w-3.5" /> : OPTION_LETTERS[i]}
                  </span>
                  <span className="min-w-0 flex-1 grid gap-2 text-[0.98rem] leading-relaxed">
                    <span className="sr-only">Option {OPTION_LETTERS[i]}: </span>
                    {opt.image_url && (
                      <img
                        src={opt.image_url}
                        alt={`Option ${OPTION_LETTERS[i]}`}
                        loading="lazy"
                        className="max-h-48 w-auto max-w-full object-contain border border-brand/10 bg-white"
                      />
                    )}
                    {opt.text && <RichContent text={opt.text} />}
                  </span>
                </label>
              );
            })}
          </div>
        </fieldset>
      );
    }
    case "short_answer":
      return (
        <div>
          <label htmlFor={inputId} className="text-sm text-brand/70">
            Your answer
          </label>
          <input
            id={inputId}
            type="text"
            value={textOf(value)}
            disabled={disabled}
            autoComplete="off"
            spellCheck={false}
            aria-describedby={describedBy}
            onChange={(e) =>
              onChange({ text: e.target.value, image_url: imageOf(value) || undefined })
            }
            className="mt-2 w-full border border-brand/20 bg-white px-4 py-3 text-base focus:border-brand focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
          />
          {onUploadImage && (
            <AttachImage
              value={value}
              onChange={onChange}
              disabled={disabled}
              onUploadImage={onUploadImage}
            />
          )}
        </div>
      );
    case "numeric": {
      const text = textOf(value);
      return (
        <div>
          <label htmlFor={inputId} className="text-sm text-brand/70">
            Numerical answer{question.unit ? ` (in ${question.unit})` : ""}
          </label>
          <div className="mt-2 flex max-w-md items-stretch border border-brand/20 bg-white focus-within:border-brand focus-within:ring-2 focus-within:ring-accent/40">
            <input
              id={inputId}
              type="text"
              inputMode="decimal"
              value={text}
              disabled={disabled}
              autoComplete="off"
              spellCheck={false}
              aria-describedby={`${describedBy} ${inputId}-help`}
              onChange={(e) => onChange({ text: e.target.value })}
              className="min-w-0 flex-1 bg-transparent px-4 py-3 text-base tabular-nums focus:outline-none"
            />
            {question.unit && (
              <span className="flex items-center border-l border-brand/15 px-3 text-sm text-brand/70">
                <RichContent text={question.unit} />
              </span>
            )}
          </div>
          <p id={`${inputId}-help`} className="mt-2 text-xs text-brand/70">
            Decimals, negatives and scientific notation are accepted, e.g. 0.25, −3.1, 6.02e23 or
            3.0×10^8.
          </p>
        </div>
      );
    }
    case "long_answer": {
      const text = textOf(value);
      const words = countWords(text);
      const over = question.max_words ? words > question.max_words : false;
      return (
        <div>
          <label htmlFor={inputId} className="text-sm text-brand/70">
            Your answer
          </label>
          <textarea
            id={inputId}
            value={text}
            disabled={disabled}
            rows={10}
            aria-describedby={`${describedBy} ${inputId}-count`}
            onChange={(e) =>
              onChange({ text: e.target.value, image_url: imageOf(value) || undefined })
            }
            className="mt-2 w-full resize-y border border-brand/20 bg-white px-4 py-3 text-base leading-relaxed focus:border-brand focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
          />
          <p
            id={`${inputId}-count`}
            className={`mt-1 text-xs tabular-nums ${over ? "text-destructive font-medium" : "text-brand/70"}`}
            aria-live="polite"
          >
            {words} {words === 1 ? "word" : "words"}
            {question.max_words ? ` of ${question.max_words} allowed` : ""}
            {over ? " — over the limit" : ""}
          </p>
          <p className="mt-1 text-xs text-brand/70">
            Written answers are marked by your teacher. Use $…$ for maths, e.g. $v = u + at$.
          </p>
          {onUploadImage && (
            <AttachImage
              value={value}
              onChange={onChange}
              disabled={disabled}
              onUploadImage={onUploadImage}
            />
          )}
        </div>
      );
    }
    default:
      return <p className="text-sm text-destructive">This question type is not supported.</p>;
  }
}

import { Check, Clock3, MinusCircle, X } from "lucide-react";

import { QUESTION_TYPE_LABEL } from "@/lib/assessment";
import { RichContent, ZoomableImage } from "@/components/assessment/rich-content";
import { toFrontendType, type QuestionReview } from "@/lib/api/quizzes";

const STATUS_STYLE: Record<string, { label: string; tone: string; icon: React.ReactNode }> = {
  correct: {
    label: "Correct",
    tone: "text-emerald-700 border-emerald-600/30 bg-emerald-50",
    icon: <Check className="h-3.5 w-3.5" aria-hidden />,
  },
  partial: {
    label: "Partly correct",
    tone: "text-accent-ink border-amber-600/30 bg-amber-50",
    icon: <MinusCircle className="h-3.5 w-3.5" aria-hidden />,
  },
  incorrect: {
    label: "Incorrect",
    tone: "text-destructive border-destructive/30 bg-destructive/5",
    icon: <X className="h-3.5 w-3.5" aria-hidden />,
  },
  unanswered: {
    label: "Not answered",
    tone: "text-brand/70 border-brand/20 bg-brand/[0.03]",
    icon: <MinusCircle className="h-3.5 w-3.5" aria-hidden />,
  },
  pending: {
    label: "Awaiting marking",
    tone: "text-brand/70 border-brand/20 bg-brand/[0.03]",
    icon: <Clock3 className="h-3.5 w-3.5" aria-hidden />,
  },
  graded: {
    label: "Marked",
    tone: "text-brand border-brand/20 bg-white",
    icon: <Check className="h-3.5 w-3.5" aria-hidden />,
  },
};

function formatNumber(n: number) {
  return Math.abs(n) >= 1e6 || (Math.abs(n) > 0 && Math.abs(n) < 1e-3)
    ? n.toExponential(4)
    : String(n);
}

/**
 * Question-by-question review of a submitted attempt. What is shown follows
 * the release policy already applied by the server: without released answers
 * the review has no marks, correctness or keys, so none are displayed.
 */
export function AttemptReview({
  review,
  released,
}: {
  review: QuestionReview[];
  released: boolean;
}) {
  return (
    <ol className="space-y-5">
      {review.map((q, i) => {
        const type = toFrontendType(q.type);
        const status = q.pending ? "pending" : q.status;
        const style = status ? STATUS_STYLE[status] : null;
        const selected = new Set(q.selected_option_ids ?? []);
        const correct = new Set(q.correct_option_ids ?? []);
        return (
          <li key={q.question_id} className="border border-brand/10 bg-white/60 p-5 sm:p-6">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h3 className="font-serif text-lg">Question {i + 1}</h3>
              <div className="flex items-center gap-2 text-xs">
                <span className="text-brand/70">{QUESTION_TYPE_LABEL[type]}</span>
                {released && q.awarded != null ? (
                  <span className="tabular-nums font-medium">
                    {q.awarded} / {q.marks}
                  </span>
                ) : (
                  <span className="tabular-nums text-brand/70">{q.marks} marks</span>
                )}
                {style && (
                  <span
                    className={`inline-flex items-center gap-1 border px-2 py-0.5 ${style.tone}`}
                  >
                    {style.icon} {style.label}
                  </span>
                )}
              </div>
            </div>
            <div className="mt-3 text-brand">
              <RichContent text={q.body} />
              {q.image_url && (
                <div className="mt-3">
                  <ZoomableImage src={q.image_url} alt={`Figure for question ${i + 1}`} />
                </div>
              )}
            </div>

            {q.options && q.options.length > 0 ? (
              <ul className="mt-4 grid gap-2">
                {q.options
                  .slice()
                  .sort((a, b) => a.position - b.position)
                  .map((o) => {
                    const picked = selected.has(o.id);
                    const isKey = released && correct.has(o.id);
                    return (
                      <li
                        key={o.id}
                        className={`flex items-start gap-3 border px-3 py-2.5 text-sm ${
                          isKey
                            ? "border-emerald-600/40 bg-emerald-50/60"
                            : picked && released
                              ? "border-destructive/30 bg-destructive/5"
                              : picked
                                ? "border-brand/40 bg-brand/[0.04]"
                                : "border-brand/10"
                        }`}
                      >
                        <span className="mt-0.5 w-24 shrink-0 text-xs text-brand/70">
                          {picked ? "Your answer" : ""}
                          {isKey ? (picked ? " ✓" : "Correct answer") : ""}
                        </span>
                        <span className="min-w-0 flex-1">
                          {o.image_url && (
                            <img src={o.image_url} alt="" className="mb-1 max-h-32 w-auto" />
                          )}
                          <RichContent text={o.body} />
                        </span>
                      </li>
                    );
                  })}
              </ul>
            ) : (
              <div className="mt-4 grid gap-3 text-sm">
                <div>
                  <p className="eyebrow text-brand/70 mb-1">Your answer</p>
                  {q.answered ? (
                    <div className="border border-brand/10 bg-white px-3 py-2 whitespace-pre-wrap">
                      {type === "long_answer" ? (
                        <RichContent text={q.text_answer} />
                      ) : (
                        q.text_answer
                      )}
                      {q.answer_image_url && (
                        <div className="mt-2">
                          <ZoomableImage src={q.answer_image_url} alt="Your attached working" />
                        </div>
                      )}
                    </div>
                  ) : (
                    <p className="text-brand/70">No answer given.</p>
                  )}
                </div>
                {released && q.accepted_answers && q.accepted_answers.length > 0 && (
                  <p>
                    <span className="eyebrow text-brand/70 mr-2">Accepted</span>
                    {q.accepted_answers.join(" · ")}
                  </p>
                )}
                {released && q.numeric_answer != null && (
                  <p>
                    <span className="eyebrow text-brand/70 mr-2">Correct value</span>
                    <span className="tabular-nums">{formatNumber(q.numeric_answer)}</span>
                    {q.unit ? ` ${q.unit}` : ""}
                    {q.tolerance ? (
                      <span className="text-brand/70">
                        {" "}
                        (± {q.tolerance}
                        {q.tolerance_type === "percent" ? "%" : ""})
                      </span>
                    ) : null}
                  </p>
                )}
              </div>
            )}

            {q.feedback && (
              <div className="mt-4 border-l-2 border-brand/30 bg-brand/[0.03] px-4 py-3 text-sm">
                <p className="eyebrow text-brand/70 mb-1">Teacher feedback</p>
                <RichContent text={q.feedback} />
              </div>
            )}
            {released && q.explanation && (
              <div className="mt-4 border-l-2 border-accent bg-accent/5 px-4 py-3 text-sm">
                <p className="eyebrow text-accent-ink mb-1">Explanation</p>
                <RichContent text={q.explanation} />
              </div>
            )}
          </li>
        );
      })}
    </ol>
  );
}

import { Maximize2, Minus, Plus } from "lucide-react";
import { Fragment, memo, useEffect, useMemo, useState, type ReactNode } from "react";

import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { resolveAssetUrl } from "@/lib/api/client";
import { hasMath, tokenize } from "@/lib/math-text";

/**
 * Academic content renderer shared by the exam, the result review and the
 * teacher preview, so all three show exactly the same thing.
 *
 * Supported syntax (plain text, never HTML — content is always escaped):
 *   $…$ or \(…\)        inline math (KaTeX)
 *   $$…$$ or \[…\]       display math
 *   \ce{…}               chemistry (mhchem), inline or inside math
 *   **bold**, *italic*, `code`
 *   | a | b |            pipe tables (first row is the header; a |---| row is optional)
 *   blank line           new paragraph
 *   \$                   a literal dollar sign
 *
 * KaTeX runs with trust disabled and expansion limits, and is loaded lazily so
 * quizzes without math never download it.
 */

type KatexModule = typeof import("katex");
let katexPromise: Promise<KatexModule> | null = null;

function loadKatex(): Promise<KatexModule> {
  if (!katexPromise) {
    katexPromise = Promise.all([
      import("katex"),
      import("katex/contrib/mhchem"),
      import("katex/dist/katex.min.css"),
    ]).then(([mod]) => {
      const m = mod as unknown as { default?: KatexModule };
      return (m.default ?? (mod as unknown as KatexModule)) as KatexModule;
    });
  }
  return katexPromise;
}

function useKatex(enabled: boolean): KatexModule | null {
  const [katex, setKatex] = useState<KatexModule | null>(null);
  useEffect(() => {
    if (!enabled || katex) return;
    let alive = true;
    loadKatex()
      .then((k) => alive && setKatex(() => k))
      .catch(() => {
        /* rendering falls back to the raw TeX source */
      });
    return () => {
      alive = false;
    };
  }, [enabled, katex]);
  return katex;
}

function renderInline(text: string, keyPrefix: string): ReactNode[] {
  // **bold**, *italic*, `code` — simple, non-nested.
  const parts: ReactNode[] = [];
  const pattern = /(\*\*[^*]+\*\*|\*[^*\s][^*]*\*|`[^`]+`)/g;
  let last = 0;
  let match: RegExpExecArray | null;
  let n = 0;
  while ((match = pattern.exec(text))) {
    if (match.index > last) parts.push(text.slice(last, match.index));
    const m = match[0];
    const key = `${keyPrefix}-${n++}`;
    if (m.startsWith("**")) parts.push(<strong key={key}>{m.slice(2, -2)}</strong>);
    else if (m.startsWith("`"))
      parts.push(
        <code key={key} className="font-mono text-[0.92em] bg-brand/[0.06] px-1 py-0.5">
          {m.slice(1, -1)}
        </code>,
      );
    else parts.push(<em key={key}>{m.slice(1, -1)}</em>);
    last = match.index + m.length;
  }
  if (last < text.length) parts.push(text.slice(last));
  return parts;
}

const MathSpan = memo(function MathSpan({
  tex,
  display,
  katex,
}: {
  tex: string;
  display: boolean;
  katex: KatexModule | null;
}) {
  const html = useMemo(() => {
    if (!katex) return null;
    try {
      return katex.renderToString(tex, {
        displayMode: display,
        throwOnError: false,
        trust: false,
        strict: "ignore",
        maxSize: 20,
        maxExpand: 500,
        output: "htmlAndMathml",
      });
    } catch {
      return null;
    }
  }, [tex, display, katex]);

  if (!html) {
    return (
      <code className={`font-mono text-[0.9em] text-brand/70 ${display ? "block my-3" : ""}`}>
        {tex}
      </code>
    );
  }
  if (display) {
    return (
      <span
        className="block my-3 overflow-x-auto overflow-y-hidden max-w-full py-1"
        // KaTeX output from escaped TeX with trust disabled.
        dangerouslySetInnerHTML={{ __html: html }}
      />
    );
  }
  return (
    <span
      className="inline-block max-w-full align-middle"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
});

function Paragraph({ text, katex, k }: { text: string; katex: KatexModule | null; k: string }) {
  // Tokenise the whole paragraph so display maths may span several lines;
  // line breaks inside plain text become <br />.
  return (
    <p className="leading-relaxed [&:not(:first-child)]:mt-3">
      {tokenize(text).map((t, ti) =>
        t.kind === "math" ? (
          <MathSpan key={ti} tex={t.value} display={t.display} katex={katex} />
        ) : (
          <Fragment key={ti}>
            {t.value.split("\n").map((line, li) => (
              <Fragment key={li}>
                {li > 0 && <br />}
                {renderInline(line, `${k}-${ti}-${li}`)}
              </Fragment>
            ))}
          </Fragment>
        ),
      )}
    </p>
  );
}

function Cell({ text, katex }: { text: string; katex: KatexModule | null }) {
  return (
    <>
      {tokenize(text).map((t, i) =>
        t.kind === "math" ? (
          <MathSpan key={i} tex={t.value} display={false} katex={katex} />
        ) : (
          <Fragment key={i}>{renderInline(t.value, `c${i}`)}</Fragment>
        ),
      )}
    </>
  );
}

function Table({ rows, katex }: { rows: string[]; katex: KatexModule | null }) {
  const parsed = rows
    .filter((r) => !/^\s*\|?\s*:?-{3,}/.test(r))
    .map((r) =>
      r
        .trim()
        .replace(/^\|/, "")
        .replace(/\|$/, "")
        .split("|")
        .map((c) => c.trim()),
    );
  const [head, ...body] = parsed;
  return (
    <div className="my-3 overflow-x-auto max-w-full">
      <table className="text-sm border-collapse min-w-[16rem]">
        <thead>
          <tr>
            {head.map((c, i) => (
              <th
                key={i}
                scope="col"
                className="border border-brand/15 bg-brand/[0.04] px-3 py-1.5 text-left font-medium"
              >
                <Cell text={c} katex={katex} />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {body.map((row, ri) => (
            <tr key={ri}>
              {row.map((c, ci) => (
                <td key={ci} className="border border-brand/15 px-3 py-1.5 tabular-nums">
                  <Cell text={c} katex={katex} />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function RichContent({
  text,
  className = "",
}: {
  text?: string | null;
  className?: string;
}) {
  const source = text ?? "";
  const katex = useKatex(hasMath(source));
  const blocks = useMemo(() => {
    // Keep display math with blank lines inside it in one block.
    const out: { type: "p" | "table"; lines: string[] }[] = [];
    let current: { type: "p" | "table"; lines: string[] } | null = null;
    for (const line of source.replace(/\r\n/g, "\n").split("\n")) {
      const isTable = /^\s*\|.*\|\s*$/.test(line);
      if (!line.trim()) {
        if (current?.type === "p" && (current.lines.join("\n").split("$$").length - 1) % 2 === 1) {
          current.lines.push("");
          continue;
        }
        current = null;
        continue;
      }
      const type = isTable ? "table" : "p";
      if (!current || current.type !== type) {
        current = { type, lines: [] };
        out.push(current);
      }
      current.lines.push(line);
    }
    return out;
  }, [source]);

  if (!source.trim()) return null;
  return (
    <div className={`rich-content break-words ${className}`}>
      {blocks.map((b, i) =>
        b.type === "table" ? (
          <Table key={i} rows={b.lines} katex={katex} />
        ) : (
          <Paragraph key={i} text={b.lines.join("\n")} katex={katex} k={String(i)} />
        ),
      )}
    </div>
  );
}

/** Image with a zoomable full-screen viewer (diagrams, graphs, circuits). */
export function ZoomableImage({ src, alt }: { src: string; alt: string }) {
  const [open, setOpen] = useState(false);
  const [scale, setScale] = useState(1);
  const url = resolveAssetUrl(src);
  return (
    <>
      <button
        type="button"
        onClick={() => {
          setScale(1);
          setOpen(true);
        }}
        className="group relative block max-w-full border border-brand/10 bg-white focus-visible:outline-2 focus-visible:outline-accent"
        aria-label={`Enlarge image: ${alt}`}
      >
        <img
          src={url}
          alt={alt}
          loading="lazy"
          className="max-h-[22rem] w-auto max-w-full object-contain"
        />
        <span className="absolute right-2 top-2 bg-white/90 p-1 text-brand/70 opacity-80 group-hover:opacity-100">
          <Maximize2 className="h-4 w-4" aria-hidden />
        </span>
      </button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-[95vw] w-[95vw] h-[90vh] p-0 flex flex-col">
          <DialogTitle className="sr-only">{alt}</DialogTitle>
          <div className="flex items-center gap-2 border-b border-brand/10 px-4 py-2 text-sm">
            <button
              type="button"
              className="p-2 hover:bg-brand/5"
              onClick={() => setScale((s) => Math.max(0.5, s - 0.25))}
              aria-label="Zoom out"
            >
              <Minus className="h-4 w-4" />
            </button>
            <span className="tabular-nums w-12 text-center" aria-live="polite">
              {Math.round(scale * 100)}%
            </span>
            <button
              type="button"
              className="p-2 hover:bg-brand/5"
              onClick={() => setScale((s) => Math.min(4, s + 0.25))}
              aria-label="Zoom in"
            >
              <Plus className="h-4 w-4" />
            </button>
          </div>
          <div className="flex-1 overflow-auto bg-brand/[0.03]">
            <img
              src={url}
              alt={alt}
              style={{ width: `${scale * 100}%`, maxWidth: "none" }}
              className="mx-auto block h-auto"
            />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

/** Parsing helpers for the academic content renderer (see components/assessment/rich-content.tsx). */

const MATH_PATTERN = /\$\$|\\\[|\\\(|\$|\\ce\{/;

export function hasMath(text: string | undefined | null): boolean {
  return Boolean(text && MATH_PATTERN.test(text));
}

export type Token =
  { kind: "text"; value: string } | { kind: "math"; value: string; display: boolean };

/** Split one paragraph into text and math tokens. */
export function tokenize(input: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;
  let buffer = "";
  const flush = () => {
    if (buffer) tokens.push({ kind: "text", value: buffer });
    buffer = "";
  };
  const takeUntil = (close: string, from: number) => {
    const end = input.indexOf(close, from);
    return end === -1 ? null : { body: input.slice(from, end), next: end + close.length };
  };
  while (i < input.length) {
    if (input.startsWith("\\$", i)) {
      buffer += "$";
      i += 2;
      continue;
    }
    const opener = input.startsWith("$$", i)
      ? { open: "$$", close: "$$", display: true }
      : input.startsWith("\\[", i)
        ? { open: "\\[", close: "\\]", display: true }
        : input.startsWith("\\(", i)
          ? { open: "\\(", close: "\\)", display: false }
          : input[i] === "$"
            ? { open: "$", close: "$", display: false }
            : null;
    if (opener) {
      const found = takeUntil(opener.close, i + opener.open.length);
      if (found && found.body.trim()) {
        flush();
        tokens.push({ kind: "math", value: found.body, display: opener.display });
        i = found.next;
        continue;
      }
    }
    if (input.startsWith("\\ce{", i)) {
      // Balance braces so \ce{Ca(OH)2 + {}^{14}C} works.
      let depth = 0;
      let j = i + 3;
      for (; j < input.length; j++) {
        if (input[j] === "{") depth++;
        else if (input[j] === "}") {
          depth--;
          if (depth === 0) break;
        }
      }
      if (depth === 0 && j < input.length) {
        flush();
        tokens.push({ kind: "math", value: input.slice(i, j + 1), display: false });
        i = j + 1;
        continue;
      }
    }
    buffer += input[i];
    i++;
  }
  flush();
  return tokens;
}

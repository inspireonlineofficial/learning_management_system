export function scorePassword(pw: string): { score: number; label: string } {
  let score = 0;
  if (pw.length >= 8) score++;
  if (pw.length >= 12) score++;
  if (/[A-Z]/.test(pw) && /[a-z]/.test(pw)) score++;
  if (/\d/.test(pw)) score++;
  if (/[^A-Za-z0-9]/.test(pw)) score++;
  const labels = ["Too weak", "Weak", "Fair", "Good", "Strong", "Excellent"];
  return { score, label: labels[Math.min(score, 5)] };
}

export function PasswordStrength({ value }: { value: string }) {
  if (!value) return null;
  const { score, label } = scorePassword(value);
  const pct = (score / 5) * 100;
  const tone =
    score <= 1
      ? "bg-destructive"
      : score <= 2
        ? "bg-amber-500"
        : score <= 3
          ? "bg-accent"
          : "bg-emerald-600";
  return (
    <div className="mt-3" aria-live="polite">
      <div className="h-[3px] w-full bg-brand/10 overflow-hidden">
        <div
          className={`h-full ${tone} transition-all duration-300`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <div className="flex justify-between mt-2 text-[10px] uppercase tracking-widest">
        <span className="text-brand/45">Strength</span>
        <span className="text-brand/70 font-semibold">{label}</span>
      </div>
    </div>
  );
}

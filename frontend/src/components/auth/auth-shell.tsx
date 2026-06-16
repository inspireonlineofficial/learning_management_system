import type { ReactNode } from "react";
import { Link } from "@tanstack/react-router";

import libraryHero from "@/assets/library-hero.jpg";

type AuthShellProps = {
  eyebrow?: string;
  heading: string;
  subheading?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
};

export function AuthShell({ eyebrow, heading, subheading, children, footer }: AuthShellProps) {
  return (
    <div className="min-h-screen overflow-x-hidden bg-surface text-brand flex flex-col md:flex-row font-sans">
      {/* Left: brand panel */}
      <aside className="hidden md:flex md:w-1/2 lg:w-3/5 bg-brand relative overflow-hidden p-12 lg:p-16 flex-col justify-between text-white">
        <div className="relative z-10">
          <Link to="/" className="font-serif italic text-2xl tracking-tight text-accent">
            Inspire LMS
          </Link>
          <h1 className="font-serif text-5xl lg:text-6xl mt-10 leading-[1.05] max-w-lg text-balance">
            Science tutoring that keeps students on track.
          </h1>
          <p className="mt-6 text-white/55 max-w-md leading-relaxed">
            Courses, live classes, assessments, progress tracking, certificates, and curated study
            materials for high school and college learners.
          </p>
        </div>

        <div className="relative z-10 flex items-center gap-4">
          <div className="flex -space-x-2">
            <div className="size-10 rounded-full border-2 border-brand bg-stone-200" />
            <div className="size-10 rounded-full border-2 border-brand bg-stone-300" />
            <div className="size-10 rounded-full border-2 border-brand bg-stone-400" />
          </div>
          <p className="text-white/55 text-sm max-w-xs leading-relaxed">
            Join focused science learners building repeatable study habits.
          </p>
        </div>

        <div className="absolute inset-0 opacity-25 mix-blend-overlay grayscale">
          <img src={libraryHero} alt="" className="w-full h-full object-cover" aria-hidden="true" />
        </div>
        <div className="absolute inset-0 bg-gradient-to-b from-brand/20 via-transparent to-brand/60" />
      </aside>

      {/* Right: form surface */}
      <main className="min-w-0 flex-1 flex flex-col p-6 sm:p-10 md:p-16 lg:p-20 justify-center min-h-screen md:min-h-0">
        <div className="auth-form-shell mx-auto w-full">
          <div className="md:hidden mb-10">
            <Link to="/" className="font-serif italic text-xl text-accent">
              Inspire LMS
            </Link>
          </div>

          <header className="mb-10">
            {eyebrow ? <p className="eyebrow text-brand/40 mb-3">{eyebrow}</p> : null}
            <h2 className="text-3xl sm:text-4xl font-serif mb-3 text-balance break-words">
              {heading}
            </h2>
            {subheading ? (
              <p className="text-brand/60 leading-relaxed break-words">{subheading}</p>
            ) : null}
          </header>

          {children}

          {footer ? (
            <div className="mt-12 pt-8 border-t border-brand/5 text-center">{footer}</div>
          ) : null}
        </div>
      </main>
    </div>
  );
}

type FieldProps = {
  label: string;
  htmlFor: string;
  error?: string;
  hint?: ReactNode;
  trailing?: ReactNode;
  children: ReactNode;
};

export function Field({ label, htmlFor, error, hint, trailing, children }: FieldProps) {
  return (
    <div>
      <div className="flex justify-between items-center mb-2">
        <label htmlFor={htmlFor} className="eyebrow text-brand/45">
          {label}
        </label>
        {trailing}
      </div>
      {children}
      {error ? (
        <p className="mt-2 text-xs text-destructive">{error}</p>
      ) : hint ? (
        <p className="mt-2 text-xs text-brand/45">{hint}</p>
      ) : null}
    </div>
  );
}

export const inputClass =
  "w-full max-w-full min-w-0 bg-transparent border-b border-brand/20 py-3 focus:outline-none focus:border-accent transition-colors text-lg placeholder:text-brand/25 disabled:opacity-50";

export const primaryButtonClass =
  "box-border w-full max-w-full bg-brand text-white py-4 font-serif text-lg sm:text-xl hover:bg-brand/90 transition-all flex items-center justify-center gap-2 group disabled:opacity-60 disabled:cursor-not-allowed";

export const ghostButtonClass =
  "w-full border border-brand/15 text-brand py-3 text-sm font-medium hover:bg-brand/[0.03] transition-colors flex items-center justify-center gap-3 disabled:opacity-60";

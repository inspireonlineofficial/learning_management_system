import type { ReactNode } from "react";

import libraryHero from "@/assets/library-hero.jpg";
import { BrandLogo } from "@/components/layout/brand-logo";

type AuthShellProps = {
  eyebrow?: string;
  heading: string;
  subheading?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
};

export function AuthShell({ eyebrow, heading, subheading, children, footer }: AuthShellProps) {
  return (
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy flex flex-col md:flex-row font-sans antialiased">
      {/* Left: atmospheric navy brand panel */}
      <aside className="hidden md:flex md:w-[46%] lg:w-[54%] hero-atmosphere relative overflow-hidden p-12 lg:p-16 flex-col justify-between text-white">
        <div className="relative z-10">
          <div className="inline-flex rounded-md bg-white px-3.5 py-2">
            <BrandLogo imageClassName="max-h-11 max-w-[190px]" />
          </div>
          <p className="mt-12 eyebrow text-amber-200/90">SSC, HSC &amp; Admission Prep</p>
          <h1 className="font-serif text-3xl lg:text-4xl font-bold mt-4 leading-tight max-w-lg text-pretty">
            Science tutoring that keeps students on track.
          </h1>
          <p className="mt-5 text-slate-200 max-w-md leading-relaxed text-base">
            Courses, live classes, assessments, progress tracking, certificates, and curated study
            materials for high school and college learners.
          </p>
        </div>

        <div className="relative z-10 pt-8 border-t border-white/15">
          <p className="text-slate-200 text-sm max-w-xs leading-relaxed">
            Join focused science learners building repeatable weekly study habits.
          </p>
        </div>

        <div className="absolute inset-0 opacity-20 mix-blend-overlay grayscale pointer-events-none">
          <img src={libraryHero} alt="" className="w-full h-full object-cover" aria-hidden="true" />
        </div>
        <div className="absolute inset-0 bg-gradient-to-b from-navy/30 via-transparent to-navy/75 pointer-events-none" />
      </aside>

      {/* Right: form surface */}
      <main className="min-w-0 flex-1 flex flex-col px-4 py-8 sm:px-8 sm:py-12 md:p-12 lg:p-16 justify-center min-h-screen md:min-h-0 bg-canvas">
        <div className="auth-form-shell mx-auto w-full bg-white md:bg-transparent rounded-xl md:rounded-none border border-borderui md:border-0 p-6 sm:p-8 md:p-0 shadow-sm md:shadow-none">
          <div className="md:hidden mb-7 pb-5 border-b border-borderui flex items-center justify-between">
            <BrandLogo imageClassName="max-h-10 max-w-[165px]" />
          </div>

          <header className="mb-8">
            {eyebrow ? (
              <p className="text-xs font-bold tracking-[0.12em] uppercase text-bronze mb-2.5">
                {eyebrow}
              </p>
            ) : null}
            <h2 className="text-2xl sm:text-3xl lg:text-4xl font-serif font-bold text-navy mb-2.5 text-balance break-words">
              {heading}
            </h2>
            {subheading ? (
              <p className="text-sm sm:text-base text-slatebody leading-relaxed break-words">
                {subheading}
              </p>
            ) : null}
          </header>

          {children}

          {footer ? (
            <div className="mt-8 pt-6 border-t border-borderui text-center text-sm text-slatebody">
              {footer}
            </div>
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
      <div className="flex justify-between items-center gap-2 mb-1.5">
        <label htmlFor={htmlFor} className="text-sm font-semibold text-navy">
          {label}
        </label>
        {trailing}
      </div>
      {children}
      {error ? (
        <p className="mt-1.5 text-xs font-medium text-red-700" role="alert">
          {error}
        </p>
      ) : hint ? (
        <p className="mt-1.5 text-xs text-slatebody">{hint}</p>
      ) : null}
    </div>
  );
}

export const inputClass =
  "w-full max-w-full min-w-0 min-h-11 px-3.5 py-2.5 rounded-lg border border-borderui bg-canvas md:bg-white text-navy text-sm sm:text-base placeholder:text-slatebody/70 focus:bg-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-navy transition-colors disabled:opacity-50";

export const primaryButtonClass =
  "box-border w-full max-w-full min-h-[48px] px-5 py-3 rounded-lg bg-navy text-white font-semibold text-sm sm:text-base hover:bg-navy-light shadow-sm transition-colors flex items-center justify-center gap-2 group disabled:opacity-60 disabled:cursor-not-allowed focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy";

export const ghostButtonClass =
  "w-full min-h-11 px-4 py-2.5 rounded-lg border border-borderui bg-white text-navy text-sm font-semibold hover:bg-stoneband transition-colors flex items-center justify-center gap-3 disabled:opacity-60 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy";

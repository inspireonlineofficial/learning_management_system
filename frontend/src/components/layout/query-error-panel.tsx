import type { ReactNode } from "react";

import { ApiError } from "@/lib/api/client";

const FRIENDLY_MESSAGES: Record<string, string> = {
  NO_VIDEO: "This lesson doesn't have a video.",
  VIDEO_NOT_READY: "This video is still processing. It'll be ready in a moment.",
  LESSON_NOT_FOUND: "This lesson is no longer available.",
  NOT_ENROLLED: "You're not enrolled in this course.",
  PROFILE_INCOMPLETE: "Complete your profile to access this lesson.",
  ENROLLMENT_REVOKED: "Your enrollment has been cancelled.",
  ACCESS_DENIED: "You don't have access to this content.",
};

function getFriendlyMessage(error: unknown): string | null {
  if (error instanceof ApiError && FRIENDLY_MESSAGES[error.code]) {
    return FRIENDLY_MESSAGES[error.code];
  }
  return null;
}

function getErrorMessage(error: unknown): string {
  const friendly = getFriendlyMessage(error);
  if (friendly) return friendly;
  if (error instanceof Error) return error.message;
  return "Something went wrong.";
}

export type QueryErrorPanelProps = {
  error: unknown;
  title?: string;
  message?: string;
  onRetry?: () => void;
  retryLabel?: string;
  variant?: "panel" | "compact";
  className?: string;
  children?: ReactNode;
};

export function QueryErrorPanel({
  error,
  title = "Couldn't load",
  message,
  onRetry,
  retryLabel = "Try again",
  variant = "panel",
  className = "",
  children,
}: QueryErrorPanelProps) {
  const text = message ?? getErrorMessage(error);
  const retry = onRetry;

  if (variant === "compact") {
    return (
      <p className={`text-sm text-red-700 ${className}`.trim()}>
        {text}
        {retry && (
          <>
            {" "}
            <button
              type="button"
              onClick={retry}
              className="font-semibold underline underline-offset-2 hover:text-red-900"
            >
              {retryLabel}
            </button>
          </>
        )}
      </p>
    );
  }

  return (
    <div
      className={`rounded-xl border border-red-200 bg-red-50/80 p-5 sm:p-6 text-sm ${className}`.trim()}
      role="alert"
    >
      <p className="font-semibold text-red-900">{title}</p>
      <p className="mt-1 text-red-800/80">{text}</p>
      {retry && (
        <button
          type="button"
          onClick={retry}
          className="mt-4 inline-flex items-center justify-center min-h-[40px] rounded-lg bg-navy px-4 py-2 text-xs font-semibold text-white hover:bg-navy-light transition-colors"
        >
          {retryLabel}
        </button>
      )}
      {children}
    </div>
  );
}

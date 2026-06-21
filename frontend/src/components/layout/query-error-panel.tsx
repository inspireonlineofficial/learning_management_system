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
      <p className={`text-sm text-destructive ${className}`.trim()}>
        {text}
        {retry && (
          <>
            {" "}
            <button
              type="button"
              onClick={retry}
              className="underline underline-offset-2 hover:text-destructive/80"
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
      className={`border border-destructive/20 bg-destructive/5 p-6 text-sm ${className}`.trim()}
      role="alert"
    >
      <p className="font-medium text-destructive">{title}</p>
      <p className="mt-1 text-brand/60">{text}</p>
      {retry && (
        <button
          type="button"
          onClick={retry}
          className="mt-3 bg-brand px-4 py-2 text-xs text-white"
        >
          {retryLabel}
        </button>
      )}
      {children}
    </div>
  );
}

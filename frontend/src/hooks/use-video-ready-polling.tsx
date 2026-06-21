import { useCallback, useEffect, useRef, useState } from "react";

import { ApiError } from "@/lib/api/client";

const POLL_INTERVAL_MS = 15_000;

export type VideoReadyPollingState = {
  retrying: boolean;
  secondsUntilNextRetry: number;
  cancel: () => void;
};

/**
 * Polls `refetch()` every 15s while the most recent error is
 * `ApiError` with code `VIDEO_NOT_READY`. Stops automatically when
 * the error changes, when the query succeeds, or on unmount.
 *
 * The `enabled` flag lets callers opt out without changing their
 * queryKey (e.g., user navigates away from the lesson).
 */
export function useVideoReadyPolling({
  error,
  enabled,
  refetch,
}: {
  error: unknown;
  enabled: boolean;
  refetch: () => unknown;
}): VideoReadyPollingState {
  const [secondsUntilNextRetry, setSecondsUntilNextRetry] = useState(POLL_INTERVAL_MS / 1000);
  const cancelledRef = useRef(false);

  const shouldPoll = enabled && error instanceof ApiError && error.code === "VIDEO_NOT_READY";

  const cancel = useCallback(() => {
    cancelledRef.current = true;
    setSecondsUntilNextRetry(POLL_INTERVAL_MS / 1000);
  }, []);

  useEffect(() => {
    cancelledRef.current = false;
  }, [error]);

  useEffect(() => {
    if (!shouldPoll) {
      setSecondsUntilNextRetry(POLL_INTERVAL_MS / 1000);
      return;
    }

    let active = true;
    let timeoutId: ReturnType<typeof setTimeout> | undefined;
    let tickId: ReturnType<typeof setInterval> | undefined;

    const schedule = () => {
      setSecondsUntilNextRetry(POLL_INTERVAL_MS / 1000);
      let remaining = POLL_INTERVAL_MS / 1000;
      tickId = setInterval(() => {
        remaining -= 1;
        if (remaining <= 0) remaining = POLL_INTERVAL_MS / 1000;
        if (active) setSecondsUntilNextRetry(remaining);
      }, 1000);

      timeoutId = setTimeout(async () => {
        clearInterval(tickId);
        if (cancelledRef.current || !active) return;
        try {
          await refetch();
        } catch {
          // refetch will update `error`; effect re-evaluates and schedules again.
        }
      }, POLL_INTERVAL_MS);
    };

    schedule();

    return () => {
      active = false;
      if (timeoutId) clearTimeout(timeoutId);
      if (tickId) clearInterval(tickId);
    };
  }, [shouldPoll, refetch]);

  return {
    retrying: shouldPoll && !cancelledRef.current,
    secondsUntilNextRetry,
    cancel,
  };
}

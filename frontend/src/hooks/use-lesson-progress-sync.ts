import { useCallback, useEffect, useRef } from "react";

import { saveLessonProgress } from "@/lib/api/student";

/** Save at most this often while playing. */
const SAVE_INTERVAL_MS = 15_000;
/** Watched share at which a video lesson is reported complete. */
const COMPLETE_AT_PERCENT = 90;
/** timeupdate gaps larger than this are seeks, not watching. */
const MAX_TICK_SECONDS = 2.5;

/**
 * Persists video progress without touching React state on every timeupdate:
 *  - position + watched% are tracked in refs;
 *  - writes are throttled (every 15 s while playing, plus on pause, seek,
 *    end, tab hide and page unload via a keepalive request);
 *  - watched% counts time actually played (seeking ahead does not count), so
 *    it is a reasonable signal for the server-side completion check;
 *  - completion is reported once per lesson.
 */
export function useLessonProgressSync(options: {
  courseId: string;
  lessonId: string | null;
  enabled: boolean;
  initialWatchedPercent?: number;
  alreadyCompleted?: boolean;
  onCompleted?: (lessonId: string) => void;
}) {
  const {
    courseId,
    lessonId,
    enabled,
    initialWatchedPercent = 0,
    alreadyCompleted = false,
    onCompleted,
  } = options;
  const state = useRef({
    position: 0,
    duration: 0,
    watchedSeconds: 0,
    lastTick: -1,
    lastSavedAt: 0,
    lastSavedPosition: -1,
    completedSent: alreadyCompleted,
    saving: false,
  });
  const onCompletedRef = useRef(onCompleted);
  onCompletedRef.current = onCompleted;

  // Reset per lesson.
  useEffect(() => {
    state.current = {
      position: 0,
      duration: 0,
      watchedSeconds: 0,
      lastTick: -1,
      lastSavedAt: 0,
      lastSavedPosition: -1,
      completedSent: alreadyCompleted,
      saving: false,
    };
    // initialWatchedPercent is applied once duration is known.
  }, [lessonId, alreadyCompleted]);

  const watchedPercent = useCallback(() => {
    const s = state.current;
    if (!s.duration) return 0;
    const fromSession = (s.watchedSeconds / s.duration) * 100;
    return Math.min(100, Math.max(fromSession, initialWatchedPercent));
  }, [initialWatchedPercent]);

  const flush = useCallback(
    (reason: "interval" | "pause" | "ended" | "seeked" | "hidden" | "unload") => {
      const s = state.current;
      if (!enabled || !lessonId || !s.duration) return;
      if (
        reason !== "ended" &&
        Math.abs(s.position - s.lastSavedPosition) < 1 &&
        reason !== "unload"
      )
        return;
      const percent =
        reason === "ended"
          ? Math.max(watchedPercent(), (s.position / s.duration) * 100)
          : watchedPercent();
      const complete = !s.completedSent && percent >= COMPLETE_AT_PERCENT;
      if (s.saving && reason === "interval") return;
      s.saving = true;
      s.lastSavedAt = Date.now();
      s.lastSavedPosition = s.position;
      const currentLesson = lessonId;
      saveLessonProgress(
        courseId,
        currentLesson,
        { position_seconds: s.position, watched_percent: percent, completed: complete },
        { keepalive: reason === "unload" || reason === "hidden" },
      )
        .then((res) => {
          if (complete && res?.completed && !state.current.completedSent) {
            state.current.completedSent = true;
            onCompletedRef.current?.(currentLesson);
          }
        })
        .catch(() => {
          // Network hiccup: the next interval/pause retries with fresh values.
          state.current.lastSavedPosition = -1;
        })
        .finally(() => {
          state.current.saving = false;
        });
    },
    [courseId, enabled, lessonId, watchedPercent],
  );

  const onProgress = useCallback(
    (currentTime: number, duration: number) => {
      const s = state.current;
      if (duration && Number.isFinite(duration)) s.duration = duration;
      if (s.lastTick >= 0) {
        const delta = currentTime - s.lastTick;
        if (delta > 0 && delta <= MAX_TICK_SECONDS) s.watchedSeconds += delta;
      }
      s.lastTick = currentTime;
      s.position = currentTime;
      if (Date.now() - s.lastSavedAt >= SAVE_INTERVAL_MS) flush("interval");
    },
    [flush],
  );

  const onPlaybackEvent = useCallback(
    (event: "pause" | "ended" | "seeked", currentTime: number, duration: number) => {
      const s = state.current;
      if (duration) s.duration = duration;
      s.position = currentTime;
      s.lastTick = currentTime;
      flush(event);
    },
    [flush],
  );

  // Save when the tab is hidden or the page unloads (refresh, device switch).
  useEffect(() => {
    if (!enabled) return;
    const onVisibility = () => {
      if (document.visibilityState === "hidden") flush("hidden");
    };
    const onPageHide = () => flush("unload");
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("pagehide", onPageHide);
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("pagehide", onPageHide);
      flush("unload"); // switching lessons
    };
  }, [enabled, flush]);

  return { onProgress, onPlaybackEvent };
}

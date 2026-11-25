import { useCallback, useEffect, useRef, useState } from "react";

import { ApiError } from "@/lib/api/client";
import { saveQuizAnswers, type AnswerMap, type AnswerValue } from "@/lib/api/quizzes";

/**
 * Reliable answer persistence for one quiz attempt.
 *
 * - Every change bumps a revision number. The server stores a draft only if
 *   its revision is newer than the stored one, so a slow request can never
 *   overwrite newer answers.
 * - At most one save is in flight; changes made meanwhile are sent next.
 * - Failed saves retry with backoff and immediately when the browser comes
 *   back online. While offline, answers are kept in localStorage.
 * - On load, a local backup with a higher revision than the server's draft
 *   (e.g. the network dropped before the last save) wins and is re-sent.
 * - Unsaved changes trigger the browser's leave-page warning and a
 *   keepalive save.
 */

export type SaveStatus =
  | "idle" // nothing changed since load
  | "pending" // changes waiting for the debounce
  | "saving"
  | "saved"
  | "offline" // no connection; kept on this device
  | "retrying" // server error; retrying
  | "auth" // session expired
  | "conflict" // newer answers were saved from another tab or device
  | "closed"; // attempt no longer accepts answers

type Backup = { revision: number; answers: AnswerMap; at: number };

const backupKey = (attemptId: string) => `inspire:attempt-draft:${attemptId}`;

function readBackup(attemptId: string): Backup | null {
  try {
    const raw = window.localStorage.getItem(backupKey(attemptId));
    return raw ? (JSON.parse(raw) as Backup) : null;
  } catch {
    return null;
  }
}

function writeBackup(attemptId: string, backup: Backup) {
  try {
    window.localStorage.setItem(backupKey(attemptId), JSON.stringify(backup));
  } catch {
    /* storage unavailable or full: the server copy is still the source of truth */
  }
}

export function clearAttemptBackup(attemptId: string) {
  try {
    window.localStorage.removeItem(backupKey(attemptId));
  } catch {
    /* ignore */
  }
}

const DEBOUNCE_MS = 700;
const BACKOFF_MS = [1500, 3000, 6000, 12000, 20000, 30000];

export function useAttemptAutosave(opts: {
  attemptId: string | null;
  serverAnswers: AnswerMap | null;
  serverRevision: number;
  onClosed?: (code: string) => void;
}) {
  const { attemptId, serverAnswers, serverRevision, onClosed } = opts;
  const [answers, setAnswers] = useState<AnswerMap>({});
  const [status, setStatus] = useState<SaveStatus>("idle");
  const [lastSavedAt, setLastSavedAt] = useState<Date | null>(null);
  const [ready, setReady] = useState(false);

  const answersRef = useRef<AnswerMap>({});
  const revisionRef = useRef(0); // latest local revision
  const savedRevisionRef = useRef(0); // latest revision confirmed by the server
  const inFlightRef = useRef(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const retryRef = useRef(0);
  const closedRef = useRef(false);
  const onClosedRef = useRef(onClosed);
  onClosedRef.current = onClosed;

  // Initialise from the server draft, or a newer local backup.
  useEffect(() => {
    if (!attemptId || !serverAnswers || ready) return;
    const backup = readBackup(attemptId);
    let initial = serverAnswers;
    let revision = serverRevision;
    let dirty = false;
    if (backup && backup.revision > serverRevision) {
      initial = backup.answers;
      revision = backup.revision;
      dirty = true;
    }
    answersRef.current = initial;
    revisionRef.current = revision;
    savedRevisionRef.current = dirty ? serverRevision : revision;
    setAnswers(initial);
    setReady(true);
    if (dirty) {
      setStatus("pending");
      timerRef.current = setTimeout(() => void flushRef.current(), 50);
    }
  }, [attemptId, serverAnswers, serverRevision, ready]);

  const flush = useCallback(
    async (keepalive = false) => {
      if (!attemptId || closedRef.current) return;
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
      if (inFlightRef.current && !keepalive) return; // the running save re-checks on completion
      if (savedRevisionRef.current >= revisionRef.current) return;

      const revision = revisionRef.current;
      const snapshot = answersRef.current;
      inFlightRef.current = true;
      setStatus("saving");
      try {
        const result = await saveQuizAnswers(attemptId, snapshot, revision, { keepalive });
        if (!result.saved && result.draft_revision > revision) {
          // Another tab or device stored a newer revision. Stop writing from
          // here so neither copy silently overwrites the other; the student
          // reloads to continue with the latest saved answers. (An equal
          // revision is our own retried save whose response was lost.)
          closedRef.current = true;
          setStatus("conflict");
          return;
        }
        savedRevisionRef.current = Math.max(savedRevisionRef.current, revision);
        retryRef.current = 0;
        setLastSavedAt(new Date());
        if (savedRevisionRef.current >= revisionRef.current) setStatus("saved");
      } catch (error) {
        const apiError = error instanceof ApiError ? error : null;
        if (apiError && ["ATTEMPT_ALREADY_SUBMITTED", "ATTEMPT_EXPIRED"].includes(apiError.code)) {
          closedRef.current = true;
          setStatus("closed");
          onClosedRef.current?.(apiError.code);
          return;
        }
        if (apiError?.status === 401) {
          setStatus("auth");
        } else {
          setStatus(typeof navigator !== "undefined" && !navigator.onLine ? "offline" : "retrying");
        }
        const delay = BACKOFF_MS[Math.min(retryRef.current, BACKOFF_MS.length - 1)];
        retryRef.current += 1;
        timerRef.current = setTimeout(() => void flushRef.current(), delay);
        return;
      } finally {
        inFlightRef.current = false;
      }
      if (savedRevisionRef.current < revisionRef.current) {
        // Changes arrived while saving.
        timerRef.current = setTimeout(() => void flushRef.current(), DEBOUNCE_MS);
      }
    },
    [attemptId],
  );

  const flushRef = useRef(flush);
  flushRef.current = flush;

  const setAnswer = useCallback(
    (questionId: string, value: AnswerValue) => {
      if (!attemptId || closedRef.current) return;
      const next = { ...answersRef.current, [questionId]: value };
      answersRef.current = next;
      revisionRef.current += 1;
      setAnswers(next);
      writeBackup(attemptId, { revision: revisionRef.current, answers: next, at: Date.now() });
      setStatus((s) => (s === "offline" || s === "auth" ? s : "pending"));
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => void flushRef.current(), DEBOUNCE_MS);
    },
    [attemptId],
  );

  // Retry as soon as the connection returns.
  useEffect(() => {
    const online = () => {
      retryRef.current = 0;
      void flushRef.current();
    };
    const offline = () => {
      if (savedRevisionRef.current < revisionRef.current) setStatus("offline");
    };
    window.addEventListener("online", online);
    window.addEventListener("offline", offline);
    return () => {
      window.removeEventListener("online", online);
      window.removeEventListener("offline", offline);
    };
  }, []);

  // Warn before leaving with unsaved answers; try a last keepalive save.
  useEffect(() => {
    const beforeUnload = (event: BeforeUnloadEvent) => {
      if (closedRef.current || savedRevisionRef.current >= revisionRef.current) return;
      void flushRef.current(true);
      event.preventDefault();
      event.returnValue = "";
    };
    const hidden = () => {
      if (document.visibilityState === "hidden") void flushRef.current(true);
    };
    window.addEventListener("beforeunload", beforeUnload);
    document.addEventListener("visibilitychange", hidden);
    return () => {
      window.removeEventListener("beforeunload", beforeUnload);
      document.removeEventListener("visibilitychange", hidden);
    };
  }, []);

  useEffect(
    () => () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    },
    [],
  );

  /** Save everything now (before submit). Resolves when the server has the latest revision. */
  const saveNow = useCallback(async () => {
    for (let i = 0; i < 3 && savedRevisionRef.current < revisionRef.current; i++) {
      while (inFlightRef.current) await new Promise((r) => setTimeout(r, 100));
      await flushRef.current();
    }
    return savedRevisionRef.current >= revisionRef.current;
  }, []);

  const markClosed = useCallback(() => {
    closedRef.current = true;
    if (timerRef.current) clearTimeout(timerRef.current);
    setStatus("closed");
  }, []);

  return {
    answers,
    setAnswer,
    status,
    lastSavedAt,
    ready,
    saveNow,
    markClosed,
    hasUnsaved: () => savedRevisionRef.current < revisionRef.current,
    answersRef,
  };
}

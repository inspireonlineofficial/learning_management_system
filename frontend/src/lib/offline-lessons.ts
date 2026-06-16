// Offline lesson cache backed by the browser Cache Storage API,
// with metadata (and a retry queue for failed downloads) mirrored in
// localStorage so the UI can render without the network.

const CACHE_NAME = "lovable-offline-lessons-v1";
const META_KEY = "offline-lessons:v1";
const QUEUE_KEY = "offline-lessons:queue:v1";

export type OfflineLessonMeta = {
  courseId: string;
  courseTitle: string;
  lessonId: string;
  lessonTitle: string;
  url: string;
  bytes?: number;
  savedAt: string;
};

export type QueuedDownload = Omit<OfflineLessonMeta, "savedAt" | "bytes"> & {
  attempts: number;
  lastError?: string;
  queuedAt: string;
};

export type DownloadProgress = {
  received: number;
  total: number; // 0 when Content-Length is unknown
};

// ---------- metadata ----------

function readMeta(): OfflineLessonMeta[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(META_KEY);
    return raw ? (JSON.parse(raw) as OfflineLessonMeta[]) : [];
  } catch {
    return [];
  }
}

function writeMeta(meta: OfflineLessonMeta[]) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(META_KEY, JSON.stringify(meta));
}

export function listOfflineLessons(): OfflineLessonMeta[] {
  return readMeta();
}

export function isLessonDownloaded(lessonId: string): boolean {
  return readMeta().some((m) => m.lessonId === lessonId);
}

// ---------- queue ----------

function readQueue(): QueuedDownload[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(QUEUE_KEY);
    return raw ? (JSON.parse(raw) as QueuedDownload[]) : [];
  } catch {
    return [];
  }
}

function writeQueue(q: QueuedDownload[]) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(QUEUE_KEY, JSON.stringify(q));
}

export function listQueuedDownloads(): QueuedDownload[] {
  return readQueue();
}

export function removeQueued(lessonId: string) {
  writeQueue(readQueue().filter((q) => q.lessonId !== lessonId));
}

function enqueue(entry: Omit<OfflineLessonMeta, "savedAt" | "bytes">, err: unknown) {
  const queue = readQueue().filter((q) => q.lessonId !== entry.lessonId);
  const prev = readQueue().find((q) => q.lessonId === entry.lessonId);
  queue.push({
    ...entry,
    attempts: (prev?.attempts ?? 0) + 1,
    lastError: err instanceof Error ? err.message : String(err),
    queuedAt: new Date().toISOString(),
  });
  writeQueue(queue);
}

// ---------- download ----------

async function streamToBlob(
  res: Response,
  onProgress?: (p: DownloadProgress) => void,
): Promise<Blob> {
  const total = Number(res.headers.get("content-length") ?? 0);
  if (!res.body || !onProgress) return res.blob();

  const reader = res.body.getReader();
  const chunks: Uint8Array[] = [];
  let received = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    if (value) {
      chunks.push(value);
      received += value.byteLength;
      onProgress({ received, total });
    }
  }
  return new Blob(chunks as BlobPart[], {
    type: res.headers.get("content-type") ?? undefined,
  });
}

export async function downloadLesson(
  meta: Omit<OfflineLessonMeta, "savedAt" | "bytes">,
  onProgress?: (p: DownloadProgress) => void,
): Promise<OfflineLessonMeta> {
  if (typeof window === "undefined" || !("caches" in window)) {
    throw new Error("Offline downloads aren't supported in this browser.");
  }
  try {
    const res = await fetch(meta.url, { mode: "cors" });
    if (!res.ok) throw new Error(`Download failed (${res.status})`);

    const blob = await streamToBlob(res.clone(), onProgress);

    const cache = await caches.open(CACHE_NAME);
    await cache.put(
      meta.url,
      new Response(blob, {
        headers: {
          "content-type": blob.type || "application/octet-stream",
          "content-length": String(blob.size),
        },
      }),
    );

    const entry: OfflineLessonMeta = {
      ...meta,
      bytes: blob.size,
      savedAt: new Date().toISOString(),
    };
    const next = readMeta().filter((m) => m.lessonId !== meta.lessonId);
    next.push(entry);
    writeMeta(next);
    removeQueued(meta.lessonId);
    return entry;
  } catch (err) {
    enqueue(meta, err);
    throw err;
  }
}

export async function removeOfflineLesson(lessonId: string) {
  const meta = readMeta();
  const entry = meta.find((m) => m.lessonId === lessonId);
  if (entry && typeof window !== "undefined" && "caches" in window) {
    const cache = await caches.open(CACHE_NAME);
    await cache.delete(entry.url);
  }
  writeMeta(meta.filter((m) => m.lessonId !== lessonId));
  removeQueued(lessonId);
}

// ---------- background sync ----------

let syncing = false;

export async function syncQueuedDownloads(
  onItem?: (lessonId: string, ok: boolean, err?: string) => void,
): Promise<{ ok: number; failed: number }> {
  if (syncing) return { ok: 0, failed: 0 };
  if (typeof navigator !== "undefined" && navigator.onLine === false) {
    return { ok: 0, failed: 0 };
  }
  syncing = true;
  let ok = 0;
  let failed = 0;
  try {
    const queue = readQueue();
    for (const item of queue) {
      try {
        await downloadLesson({
          courseId: item.courseId,
          courseTitle: item.courseTitle,
          lessonId: item.lessonId,
          lessonTitle: item.lessonTitle,
          url: item.url,
        });
        ok++;
        onItem?.(item.lessonId, true);
      } catch (e) {
        failed++;
        onItem?.(item.lessonId, false, e instanceof Error ? e.message : String(e));
      }
    }
  } finally {
    syncing = false;
  }
  return { ok, failed };
}

let listenerInstalled = false;
export function installOfflineSyncListener(onChange?: () => void) {
  if (typeof window === "undefined" || listenerInstalled) return;
  listenerInstalled = true;
  const run = () => {
    void syncQueuedDownloads().then((r) => {
      if (r.ok || r.failed) onChange?.();
    });
  };
  window.addEventListener("online", run);
  // Try once on install in case we came back online before the listener attached.
  if (navigator.onLine) run();
}

// ---------- storage quota ----------

export type StorageEstimate = { usage: number; quota: number };

export async function getStorageEstimate(): Promise<StorageEstimate | null> {
  if (typeof navigator === "undefined" || !navigator.storage?.estimate) return null;
  const e = await navigator.storage.estimate();
  return { usage: e.usage ?? 0, quota: e.quota ?? 0 };
}

// ---------- formatting ----------

export function formatBytes(bytes?: number): string {
  if (!bytes) return "—";
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  let n = bytes;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

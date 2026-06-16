import { createFileRoute, Link } from "@tanstack/react-router";
import { CloudOff, Download, RefreshCw, Trash2, Wifi, WifiOff } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import {
  formatBytes,
  getStorageEstimate,
  installOfflineSyncListener,
  listOfflineLessons,
  listQueuedDownloads,
  removeOfflineLesson,
  removeQueued,
  syncQueuedDownloads,
  type OfflineLessonMeta,
  type QueuedDownload,
  type StorageEstimate,
} from "@/lib/offline-lessons";

export const Route = createFileRoute("/_authenticated/student/downloads")({
  component: DownloadsPage,
});

function DownloadsPage() {
  const [items, setItems] = useState<OfflineLessonMeta[]>([]);
  const [queue, setQueue] = useState<QueuedDownload[]>([]);
  const [storage, setStorage] = useState<StorageEstimate | null>(null);
  const [online, setOnline] = useState<boolean>(
    typeof navigator === "undefined" ? true : navigator.onLine,
  );
  const [syncing, setSyncing] = useState(false);

  const refresh = useCallback(async () => {
    setItems(listOfflineLessons());
    setQueue(listQueuedDownloads());
    setStorage(await getStorageEstimate());
  }, []);

  useEffect(() => {
    void refresh();
    installOfflineSyncListener(() => {
      void refresh();
    });
    const on = () => setOnline(true);
    const off = () => setOnline(false);
    window.addEventListener("online", on);
    window.addEventListener("offline", off);
    return () => {
      window.removeEventListener("online", on);
      window.removeEventListener("offline", off);
    };
  }, [refresh]);

  async function handleRemove(lessonId: string) {
    await removeOfflineLesson(lessonId);
    void refresh();
    toast.success("Removed from offline library");
  }

  async function handleSync() {
    if (!online) {
      toast.error("You're offline — connect to retry.");
      return;
    }
    setSyncing(true);
    const { ok, failed } = await syncQueuedDownloads();
    setSyncing(false);
    void refresh();
    if (ok && !failed) toast.success(`Synced ${ok} lesson${ok === 1 ? "" : "s"}`);
    else if (ok && failed) toast.message(`Synced ${ok}, ${failed} still failing`);
    else if (failed) toast.error(`${failed} download${failed === 1 ? "" : "s"} still failing`);
    else toast.message("Nothing to sync");
  }

  const totalBytes = items.reduce((sum, i) => sum + (i.bytes ?? 0), 0);
  const quotaPct =
    storage && storage.quota > 0 ? Math.min(100, (storage.usage / storage.quota) * 100) : 0;

  return (
    <AppShell title="Offline downloads">
      <div className="flex items-center gap-3 mb-4 text-xs">
        <span
          className={`inline-flex items-center gap-1.5 px-2 py-1 border ${
            online
              ? "border-brand/15 text-brand/70"
              : "border-amber-400/30 text-amber-700 bg-amber-50"
          }`}
        >
          {online ? <Wifi className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}
          {online ? "Online" : "Offline"}
        </span>
        {queue.length > 0 && (
          <button
            onClick={handleSync}
            disabled={syncing || !online}
            className="inline-flex items-center gap-1.5 px-2.5 py-1 border border-brand/20 text-brand hover:bg-brand/[0.04] disabled:opacity-60"
          >
            <RefreshCw className={`h-3 w-3 ${syncing ? "animate-spin" : ""}`} />
            {syncing ? "Syncing…" : `Retry ${queue.length} pending`}
          </button>
        )}
      </div>

      <p className="text-sm text-brand/60 mb-2">
        Lessons saved to this device. {items.length} item{items.length === 1 ? "" : "s"} ·{" "}
        {formatBytes(totalBytes)}
      </p>

      {storage && storage.quota > 0 && (
        <div className="mb-6 max-w-md">
          <div className="flex justify-between text-[11px] text-brand/55 mb-1">
            <span>Device storage</span>
            <span>
              {formatBytes(storage.usage)} of {formatBytes(storage.quota)}
            </span>
          </div>
          <div className="h-1.5 bg-brand/10 overflow-hidden">
            <div className="h-full bg-brand/60" style={{ width: `${quotaPct}%` }} />
          </div>
        </div>
      )}

      {queue.length > 0 && (
        <section className="mb-8">
          <h2 className="font-serif text-lg mb-2 flex items-center gap-2">
            <CloudOff className="h-4 w-4 text-amber-600" />
            Pending sync
          </h2>
          <ul className="divide-y divide-brand/10 border border-amber-400/20 bg-amber-50/40">
            {queue.map((q) => (
              <li key={q.lessonId} className="flex items-center justify-between gap-4 px-4 py-3">
                <div className="min-w-0">
                  <p className="text-sm font-medium truncate">{q.lessonTitle}</p>
                  <p className="text-xs text-brand/55 truncate">
                    {q.courseTitle} · attempt {q.attempts}
                    {q.lastError ? ` · ${q.lastError}` : ""}
                  </p>
                </div>
                <button
                  onClick={() => {
                    removeQueued(q.lessonId);
                    void refresh();
                  }}
                  className="inline-flex items-center gap-1.5 text-xs text-brand/55 hover:text-destructive"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  Cancel
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      {items.length === 0 ? (
        <div className="border border-brand/15 bg-white/40 p-10 text-center">
          <Download className="h-8 w-8 text-brand/30 mx-auto mb-3" />
          <p className="text-sm text-brand/60">
            No offline lessons yet. Open a lesson and tap "Save offline".
          </p>
        </div>
      ) : (
        <ul className="divide-y divide-brand/10 border border-brand/10 bg-white/40">
          {items.map((item) => (
            <li key={item.lessonId} className="flex items-center justify-between gap-4 px-4 py-3">
              <div className="min-w-0">
                <Link
                  to="/student/player/$courseId"
                  params={{ courseId: item.courseId }}
                  className="block text-sm font-medium text-brand hover:underline truncate"
                >
                  {item.lessonTitle}
                </Link>
                <p className="text-xs text-brand/55 truncate">
                  {item.courseTitle} · {formatBytes(item.bytes)} ·{" "}
                  {new Date(item.savedAt).toLocaleDateString()}
                </p>
              </div>
              <button
                onClick={() => handleRemove(item.lessonId)}
                className="inline-flex items-center gap-1.5 text-xs text-brand/55 hover:text-destructive"
              >
                <Trash2 className="h-3.5 w-3.5" />
                Remove
              </button>
            </li>
          ))}
        </ul>
      )}
    </AppShell>
  );
}

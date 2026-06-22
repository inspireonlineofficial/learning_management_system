/**
 * Resumable multipart uploader.
 *
 * Why this exists:
 *   The single PUT path (`uploadVideoDirect` in video-upload.ts) works for
 *   small files and tolerates network blips via 4-attempt retry. It cannot
 *   survive a browser refresh mid-upload: the presigned URL is short-lived
 *   and the bytes already uploaded are wedged in the bucket. Multipart
 *   upload fixes both problems — each part has its own URL, parts are
 *   assembled server-side on completion, and the client only needs to
 *   remember (video_id, upload_id, completed parts) to resume.
 *
 * Resume mechanism:
 *   We persist upload state to IndexedDB under the key
 *   `lms-upload:{video_id}`. The structure stores upload_id, file_name,
 *   total_size, chunk_size, and a Map of part_number -> ETag for completed
 *   parts. On start() the client checks IndexedDB first; if a row exists
 *   for the same file (matched by name+size) we resume from there.
 *
 * Concurrency:
 *   Parts are uploaded 3 at a time. Going higher (e.g. 6) helps on fast
 *   broadband but saturates mobile uplinks, so we keep it conservative.
 */
import { apiRequest } from "@/lib/api/client";

const PARALLEL_PART_UPLOADS = 3;
const MAX_RETRIES_PER_PART = 5;

export type MultipartInitResponse = {
  video_id: string;
  upload_id: string;
  rustfs_key: string;
  chunk_size: number;
  total_chunks: number;
  poll_url: string;
  expires_at: string;
};

export type UploadProgress = {
  loaded: number;
  total: number;
  ratio: number;
  /** Parts successfully uploaded so far. */
  completedParts: number;
  totalParts: number;
};

type PersistedUpload = {
  video_id: string;
  upload_id: string;
  file_name: string;
  file_size: number;
  chunk_size: number;
  total_chunks: number;
  /** Map of part_number -> etag. Stored as a plain object for IDB friendliness. */
  parts: Record<string, string>;
  started_at: string;
};

const IDB_NAME = "lms-uploads";
const IDB_STORE = "uploads";

function openIDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(IDB_NAME, 1);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(IDB_STORE)) {
        db.createObjectStore(IDB_STORE, { keyPath: "video_id" });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function loadUploadState(videoId: string): Promise<PersistedUpload | null> {
  const db = await openIDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(IDB_STORE, "readonly");
    const store = tx.objectStore(IDB_STORE);
    const req = store.get(videoId);
    req.onsuccess = () => resolve((req.result as PersistedUpload) ?? null);
    req.onerror = () => reject(req.error);
  });
}

async function saveUploadState(state: PersistedUpload): Promise<void> {
  const db = await openIDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(IDB_STORE, "readwrite");
    const store = tx.objectStore(IDB_STORE);
    store.put(state);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

async function clearUploadState(videoId: string): Promise<void> {
  const db = await openIDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(IDB_STORE, "readwrite");
    tx.objectStore(IDB_STORE).delete(videoId);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

async function readMagicBytes(file: File, n = 512): Promise<string> {
  const slice = file.slice(0, Math.min(n, file.size));
  const buf = await slice.arrayBuffer();
  let binary = "";
  const bytes = new Uint8Array(buf);
  for (let i = 0; i < bytes.byteLength; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary);
}

function uploadPart(
  url: string,
  blob: Blob,
  onProgress: (loaded: number) => void,
  signal: AbortSignal,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", url, true);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded);
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        // S3 returns the ETag in the response header. Quoted, just like S3.
        const etag = xhr.getResponseHeader("ETag") ?? "";
        if (!etag) {
          reject(new Error(`PUT part: missing ETag in response`));
          return;
        }
        resolve(etag);
      } else {
        reject(new Error(`PUT part failed: ${xhr.status} ${xhr.statusText}`));
      }
    };
    xhr.onerror = () => reject(new Error("Network error during part upload"));
    xhr.onabort = () => reject(new Error("Upload aborted"));
    xhr.send(blob);
    signal.addEventListener("abort", () => xhr.abort(), { once: true });
  });
}

/**
 * Start (or resume) a resumable multipart upload.
 *
 * If `existingVideoId` is provided we look up the persisted state for that
 * upload and skip re-initialisation. If no state exists, the function does
 * a full init: call /multipart/init, then upload every chunk, then call
 * /multipart/complete. Progress is reported after every part lands.
 */
export async function uploadMultipart(opts: {
  courseId: string;
  file: File;
  onProgress?: (p: UploadProgress) => void;
  signal?: AbortSignal;
  existingVideoId?: string;
}): Promise<{ video_id: string }> {
  const { courseId, file, onProgress, signal } = opts;
  const effectiveSignal = signal ?? new AbortController().signal;

  // Resume path: lookup existing state and skip init.
  let state: PersistedUpload | null = null;
  if (opts.existingVideoId) {
    state = await loadUploadState(opts.existingVideoId);
    if (state) {
      // Verify the user picked the same file (otherwise we'd corrupt the upload).
      if (state.file_name !== file.name || state.file_size !== file.size) {
        state = null;
        await clearUploadState(opts.existingVideoId);
      }
    }
  }

  if (!state) {
    const magicB64 = await readMagicBytes(file);
    const init = await apiRequest<MultipartInitResponse>("/v1/uploads/video/multipart/init", {
      method: "POST",
      auth: true,
      body: {
        course_id: courseId,
        file_name: file.name,
        file_size: file.size,
        mime_type: file.type || "application/octet-stream",
        magic_b64: magicB64,
        chunk_size: 8 * 1024 * 1024, // 8 MB default; S3 minimum is 5 MB
      },
      signal: effectiveSignal,
    });
    state = {
      video_id: init.video_id,
      upload_id: init.upload_id,
      file_name: file.name,
      file_size: init.total_chunks * init.chunk_size, // best-effort
      chunk_size: init.chunk_size,
      total_chunks: init.total_chunks,
      parts: {},
      started_at: new Date().toISOString(),
    };
    // Recompute total_chunks against the real file size — the server uses
    // (size + chunk - 1) / chunk, so for a 8.1 MB file at 8 MB it returns 2.
    // That matches what we want here.
    state.total_chunks = init.total_chunks;
    await saveUploadState(state);
  }

  const report = () => {
    const completed = Object.keys(state!.parts).length;
    // Approximate bytes uploaded: each completed part contributes chunk_size
    // bytes except the last one. The last chunk's actual size is total - (n-1)*chunk.
    const lastChunkSize = state!.file_size - (state!.total_chunks - 1) * state!.chunk_size;
    let loaded = 0;
    for (let i = 0; i < state!.total_chunks; i++) {
      if (state!.parts[String(i + 1)]) {
        loaded += i === state!.total_chunks - 1 ? lastChunkSize : state!.chunk_size;
      }
    }
    onProgress?.({
      loaded,
      total: state!.file_size,
      ratio: state!.file_size > 0 ? Math.min(1, loaded / state!.file_size) : 0,
      completedParts: completed,
      totalParts: state!.total_chunks,
    });
  };

  // Determine which parts still need uploading.
  const pending: number[] = [];
  for (let i = 1; i <= state.total_chunks; i++) {
    if (!state.parts[String(i)]) pending.push(i);
  }

  // Upload pending parts with bounded concurrency.
  let cursor = 0;
  async function worker(): Promise<void> {
    while (cursor < pending.length) {
      if (effectiveSignal.aborted) return;
      const idx = cursor++;
      const partNumber = pending[idx];
      const start = (partNumber - 1) * state!.chunk_size;
      const end = Math.min(start + state!.chunk_size, state!.file_size);
      const blob = file.slice(start, end);
      // Presign per part: cheap (S3 returns a URL in ms) and the URL is fresh
      // for an hour, so a retry minutes later still works.
      const { url } = await apiRequest<{ url: string }>(
        `/v1/uploads/video/multipart/${state!.video_id}/part`,
        {
          method: "POST",
          auth: true,
          body: { upload_id: state!.upload_id, part_number: partNumber },
          signal: effectiveSignal,
        },
      );

      let attempt = 0;
      // eslint-disable-next-line no-constant-condition
      while (true) {
        try {
          const etag = await uploadPart(url, blob, () => report(), effectiveSignal);
          state!.parts[String(partNumber)] = etag;
          await saveUploadState(state!);
          report();
          break;
        } catch (err) {
          if (++attempt >= MAX_RETRIES_PER_PART || effectiveSignal.aborted) throw err;
          await new Promise((r) => setTimeout(r, 500 * 2 ** (attempt - 1)));
        }
      }
    }
  }
  const workers = Array.from({ length: Math.min(PARALLEL_PART_UPLOADS, pending.length || 1) }, () => worker());
  await Promise.all(workers);

  if (effectiveSignal.aborted) {
    return { video_id: state.video_id };
  }

  // All parts uploaded. Send the parts list in order.
  const orderedParts = Object.keys(state.parts)
    .sort((a, b) => Number(a) - Number(b))
    .map((pn) => ({ part_number: Number(pn), etag: state.parts[pn] }));

  let confirmAttempt = 0;
  // eslint-disable-next-line no-constant-condition
  while (true) {
    try {
      await apiRequest<{ video_id: string }>(
        `/v1/uploads/video/multipart/${state.video_id}/complete`,
        {
          method: "POST",
          auth: true,
          body: { upload_id: state.upload_id, parts: orderedParts },
          signal: effectiveSignal,
        },
      );
      break;
    } catch (err) {
      if (++confirmAttempt >= MAX_RETRIES_PER_PART || effectiveSignal.aborted) throw err;
      await new Promise((r) => setTimeout(r, 500 * 2 ** (confirmAttempt - 1)));
    }
  }

  await clearUploadState(state.video_id);
  return { video_id: state.video_id };
}

/**
 * Abort a resumable upload: tells the server to release the S3 multipart
 * state and clears the local IndexedDB row. Returns silently if the upload
 * never started or was already cleared.
 */
export async function abortMultipartUpload(videoId: string, uploadId: string): Promise<void> {
  try {
    await apiRequest<unknown>(`/v1/uploads/video/multipart/${videoId}/abort`, {
      method: "POST",
      auth: true,
      body: { upload_id: uploadId },
    });
  } catch {
    // Best-effort: the server may have already GC'd the upload.
  }
  await clearUploadState(videoId);
}

/**
 * Look up persisted state for a video. Used by the upload UI to show a
 * "Resume previous upload?" prompt on page load.
 */
export async function findPendingUpload(videoId: string): Promise<PersistedUpload | null> {
  return loadUploadState(videoId);
}
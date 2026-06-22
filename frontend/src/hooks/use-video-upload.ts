/**
 * useVideoUpload
 *
 * React-friendly wrapper around uploadLessonVideoDirect. The teacher
 * curriculum editor calls this from its file input; the hook returns
 * progress (0..1) and a handle to cancel the upload.
 */
import { useCallback, useRef, useState } from "react";
import { uploadLessonVideoDirect } from "@/lib/api/teacher";

export type UploadState = {
  status: "idle" | "uploading" | "polling" | "ready" | "failed";
  progress: number; // 0..1
  error?: string;
  videoId?: string;
};

export function useVideoUpload(courseId: string) {
  const [state, setState] = useState<UploadState>({ status: "idle", progress: 0 });
  const controllerRef = useRef<AbortController | null>(null);

  const start = useCallback(
    async (file: File) => {
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;
      setState({ status: "uploading", progress: 0 });
      try {
        const { video_id } = await uploadLessonVideoDirect(
          courseId,
          file,
          (loaded, total) => {
            if (total > 0) setState((s) => ({ ...s, progress: loaded / total }));
          },
          controller.signal,
        );
        setState({ status: "ready", progress: 1, videoId: video_id });
        return video_id;
      } catch (e) {
        const err = e as Error;
        if (err.name === "AbortError") {
          setState({ status: "idle", progress: 0 });
          return null;
        }
        setState({ status: "failed", progress: 0, error: err.message });
        return null;
      }
    },
    [courseId],
  );

  const cancel = useCallback(() => {
    controllerRef.current?.abort();
  }, []);

  return { state, start, cancel };
}

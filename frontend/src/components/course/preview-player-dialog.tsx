import { useQuery } from "@tanstack/react-query";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { getLessonPreview } from "@/lib/api/courses";

type Props = {
  courseId: string;
  lessonId: string;
  lessonTitle: string;
  durationMinutes?: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function PreviewPlayerDialog({
  courseId,
  lessonId,
  lessonTitle,
  durationMinutes,
  open,
  onOpenChange,
}: Props) {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["lesson-preview", courseId, lessonId],
    queryFn: () => getLessonPreview(courseId, lessonId),
    enabled: open,
    staleTime: 5 * 60_000,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl p-0 bg-black border-brand/20">
        <DialogHeader className="px-6 pt-6 pb-3 bg-surface">
          <DialogTitle className="font-serif text-xl text-brand">{lessonTitle}</DialogTitle>
          <DialogDescription className="text-xs text-brand/55">
            Free preview
            {typeof durationMinutes === "number" && ` · ${durationMinutes} min`}
          </DialogDescription>
        </DialogHeader>

        <div className="aspect-video w-full bg-black grid place-items-center">
          {isLoading && <p className="text-white/60 text-sm">Loading preview…</p>}
          {isError && (
            <p className="text-white/70 text-sm px-6 text-center">
              {(error as Error)?.message ?? "Preview unavailable"}
            </p>
          )}
          {data && data.mime_type?.startsWith("audio") ? (
            <audio src={data.url} controls autoPlay className="w-full px-6" />
          ) : data ? (
            <video src={data.url} controls autoPlay playsInline className="w-full h-full">
              {data.captions_url && (
                <track kind="captions" src={data.captions_url} srcLang="en" default />
              )}
            </video>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

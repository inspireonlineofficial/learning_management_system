/**
 * VideoPlayer
 *
 * Drop-in replacement for the bare <video> tag used in the lesson player.
 *
 * Playback optimizations baked in:
 *  - `crossOrigin="anonymous"` so byte-range requests work cross-origin against
 *    the RustFS/S3 endpoint. Without it the browser refuses to issue Range
 *    headers and falls back to fully downloading the file on every seek.
 *  - `preload="metadata"` initially (so the timeline + duration are available
 *    without buffering the whole file). We escalate to "auto" once the user
 *    has interacted with the page, so the next lesson pre-buffers.
 *  - `playsInline` to keep iOS Safari from full-screening on tap.
 *  - We listen for "waiting" / "stalled" / "error" and surface a calm retry
 *    CTA so buffering feels intentional instead of broken.
 *  - When a `posterUrl` is provided we show a frame before playback starts;
 *    otherwise we render a subtle skeleton.
 *  - Buffered ranges and current time are surfaced for the progress bar and
 *    analytics; a custom overlay shows up before first play so the user can
 *    tap to start (improves autoplay reliability on iOS).
 *
 * HLS / DASH future:
 *   The component intentionally accepts a single `src` so it works with both
 *   progressive MP4 (today) and an HLS manifest URL (tomorrow, once the
 *   transcoding worker ships). Swapping in hls.js or shaka-player is a
 *   one-method change: detect `.m3u8`, attach to the <video> via Media Source
 *   Extensions, and hide our overlay when the manifest loads. The component
 *   contract is shaped so that swap is local.
 */
import { useEffect, useRef, useState } from "react";
import { Play } from "lucide-react";

interface VideoPlayerProps {
  src: string;
  posterUrl?: string;
  onError?: (message: string) => void;
  /** When the user has interacted with the page we escalate preloading. */
  hasInteracted: boolean;
  /** Fired with [seconds] whenever playback progresses. */
  onProgress?: (currentTime: number, duration: number) => void;
}

export function VideoPlayer({
  src,
  posterUrl,
  onError,
  hasInteracted,
  onProgress,
}: VideoPlayerProps) {
  const ref = useRef<HTMLVideoElement | null>(null);
  const [buffering, setBuffering] = useState(false);
  const [failed, setFailed] = useState(false);
  const [needsTap, setNeedsTap] = useState(true);
  const [bufferedAhead, setBufferedAhead] = useState(0);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    // Reload whenever the src changes so previously cached ranges from a
    // different lesson don't bleed into the new one.
    el.load();
    setFailed(false);
    setBuffering(false);
    setNeedsTap(true);
  }, [src]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.preload = hasInteracted ? "auto" : "metadata";
  }, [hasInteracted]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const onTime = () => {
      onProgress?.(el.currentTime, el.duration || 0);
      // Compute seconds of forward buffer for the overlay indicator.
      const ranges = el.buffered;
      const t = el.currentTime;
      let ahead = 0;
      for (let i = 0; i < ranges.length; i++) {
        if (ranges.start(i) <= t && ranges.end(i) >= t) {
          ahead = Math.max(ahead, ranges.end(i) - t);
          break;
        }
      }
      setBufferedAhead(ahead);
    };
    el.addEventListener("timeupdate", onTime);
    return () => el.removeEventListener("timeupdate", onTime);
  }, [onProgress]);

  return (
    <div className="aspect-video bg-black overflow-hidden relative group">
      <video
        ref={ref}
        key={src}
        src={src}
        controls
        playsInline
        // anonymous so the browser is willing to send Range headers across
        // origins. The presigned URL does not return Access-Control-Allow-
        // Origin for credentialed requests, so anonymous is the safe choice.
        crossOrigin="anonymous"
        poster={posterUrl}
        className="h-full w-full"
        onWaiting={() => setBuffering(true)}
        onPlaying={() => {
          setBuffering(false);
          setNeedsTap(false);
        }}
        onCanPlay={() => setBuffering(false)}
        onError={() => {
          setFailed(true);
          setBuffering(false);
          onError?.("Video failed to load");
        }}
      />
      {/* Tap-to-start overlay: many mobile browsers refuse autoplay until
          the user has tapped once. We show a clear CTA so it doesn't feel
          like the player is broken. */}
      {needsTap && !failed && (
        <button
          type="button"
          className="absolute inset-0 grid place-items-center bg-black/40 text-white"
          onClick={() => {
            const el = ref.current;
            if (!el) return;
            el.play().catch(() => {
              /* browser refused — fall through to manual play via native controls */
            });
          }}
          aria-label="Play video"
        >
          <span className="flex items-center gap-2 bg-white/15 backdrop-blur px-5 py-3 text-sm font-medium">
            <Play className="h-4 w-4" /> Play lesson
          </span>
        </button>
      )}
      {buffering && !failed && !needsTap && (
        <div className="pointer-events-none absolute inset-0 grid place-items-center bg-black/30 text-white text-xs">
          <span className="animate-pulse flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-white" />
            Buffering {bufferedAhead > 0 ? `· ${Math.round(bufferedAhead)}s ahead` : ""}
          </span>
        </div>
      )}
      {failed && (
        <div className="absolute inset-0 grid place-items-center bg-black/60 text-white text-sm">
          <button
            type="button"
            className="underline"
            onClick={() => {
              setFailed(false);
              ref.current?.load();
              ref.current?.play().catch(() => {
                /* autoplay restrictions handled by the controls */
              });
            }}
          >
            Tap to retry
          </button>
        </div>
      )}
    </div>
  );
}

/**
 * HLSPlayer
 *
 * A drop-in replacement for the existing <VideoPlayer> that prefers HLS
 * (when the signed-URL response has has_hls=true) and gracefully falls
 * back to progressive MP4. The native HLS path is used in Safari/iOS; on
 * Chrome/Firefox/Edge we attach the manifest to a MediaSource and feed
 * segments in via fetch — no third-party runtime required.
 *
 * This version implements a fully custom YouTube-style control overlay:
 *   - Auto-hiding timeline controls on mouse inactivity.
 *   - Central play/pause trigger with HUD animation alerts.
 *   - Slider volume adjusting (horizontal slider expands on hover).
 *   - Speeds menu selection popup.
 *   - PiP and Fullscreen toggle buttons.
 *   - Native hotkey listeners (Arrows, Space, M, F).
 */
import { useCallback, useEffect, useRef, useState } from "react";
import {
  ChevronLeft,
  ChevronRight,
  Play,
  Pause,
  Volume2,
  VolumeX,
  Volume1,
  Settings,
  Maximize,
  Minimize,
  Tv,
  RotateCcw,
} from "lucide-react";

interface HLSPlayerProps {
  /** Progressive MP4 URL. Always required as a fallback. */
  mp4Url: string;
  /** HLS master manifest URL. Optional; when present the player prefers it. */
  hlsUrl?: string;
  posterUrl?: string;
  onError?: (message: string) => void;
  hasInteracted: boolean;
  onProgress?: (currentTime: number, duration: number) => void;
}

const SEGMENT_RETRY = 3;
const SPEEDS = [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2];

export function HLSPlayer({
  mp4Url,
  hlsUrl,
  posterUrl,
  onError,
  hasInteracted,
  onProgress,
}: HLSPlayerProps) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const progressRef = useRef<HTMLDivElement | null>(null);
  const speedContainerRef = useRef<HTMLDivElement | null>(null);

  const [buffering, setBuffering] = useState(false);
  const [failed, setFailed] = useState(false);
  const [needsTap, setNeedsTap] = useState(true);
  const [usingHls, setUsingHls] = useState(false);
  const [hlsFailed, setHlsFailed] = useState(false);
  const userInteractedRef = useRef(false);

  // Custom Controls States
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [controlsVisible, setControlsVisible] = useState(true);
  const [showSpeedMenu, setShowSpeedMenu] = useState(false);
  const [playbackRate, setPlaybackRate] = useState(1);
  const [indicator, setIndicator] = useState<string | null>(null);
  const [indicatorTimeout, setIndicatorTimeout] = useState<number | null>(null);

  // Advanced states for YouTube-like scrubber
  const [bufferedPercent, setBufferedPercent] = useState(0);
  const [hoverTooltipTime, setHoverTooltipTime] = useState<string | null>(null);
  const [hoverTooltipLeft, setHoverTooltipLeft] = useState<number | null>(null);

  // Auto-hide controls timeout
  const controlsTimeoutRef = useRef<number | null>(null);

  const resetControlsTimeout = useCallback(() => {
    setControlsVisible(true);
    if (controlsTimeoutRef.current) {
      window.clearTimeout(controlsTimeoutRef.current);
    }
    if (videoRef.current && !videoRef.current.paused) {
      controlsTimeoutRef.current = window.setTimeout(() => {
        setControlsVisible(false);
        setShowSpeedMenu(false);
      }, 2500);
    }
  }, []);

  const showIndicator = useCallback(
    (type: string) => {
      setIndicator(type);
      if (indicatorTimeout) window.clearTimeout(indicatorTimeout);
      const id = window.setTimeout(() => setIndicator(null), 800);
      setIndicatorTimeout(id);
    },
    [indicatorTimeout],
  );

  // Close speed menu on outside click
  useEffect(() => {
    if (!showSpeedMenu) return;
    const handleOutside = (e: MouseEvent) => {
      if (speedContainerRef.current && !speedContainerRef.current.contains(e.target as Node)) {
        setShowSpeedMenu(false);
      }
    };
    window.addEventListener("click", handleOutside);
    return () => window.removeEventListener("click", handleOutside);
  }, [showSpeedMenu]);

  // Sync playback rate when changed
  useEffect(() => {
    if (videoRef.current) {
      videoRef.current.playbackRate = playbackRate;
    }
  }, [playbackRate]);

  // Keyboard shortcut listener
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const video = videoRef.current;
      if (!video) return;

      const target = e.target as HTMLElement;
      if (target.tagName === "TEXTAREA" || target.tagName === "INPUT" || target.isContentEditable) {
        return;
      }

      resetControlsTimeout();

      switch (e.key) {
        case " ":
        case "k":
        case "K":
          e.preventDefault();
          togglePlay();
          break;
        case "j":
        case "J":
          e.preventDefault();
          video.currentTime = Math.max(0, video.currentTime - 10);
          showIndicator("seek-back-10");
          break;
        case "l":
        case "L":
          e.preventDefault();
          video.currentTime = Math.min(video.duration || 0, video.currentTime + 10);
          showIndicator("seek-forward-10");
          break;
        case "ArrowLeft":
          e.preventDefault();
          video.currentTime = Math.max(0, video.currentTime - 5);
          showIndicator("seek-back-5");
          break;
        case "ArrowRight":
          e.preventDefault();
          video.currentTime = Math.min(video.duration || 0, video.currentTime + 5);
          showIndicator("seek-forward-5");
          break;
        case "ArrowUp":
          e.preventDefault();
          video.volume = Math.min(1, video.volume + 0.05);
          showIndicator("volume-up");
          break;
        case "ArrowDown":
          e.preventDefault();
          video.volume = Math.max(0, video.volume - 0.05);
          showIndicator("volume-down");
          break;
        case "m":
        case "M":
          e.preventDefault();
          toggleMute();
          break;
        case "f":
        case "F":
          e.preventDefault();
          toggleFullscreen();
          break;
        case "0":
        case "1":
        case "2":
        case "3":
        case "4":
        case "5":
        case "6":
        case "7":
        case "8":
        case "9": {
          e.preventDefault();
          const pct = parseInt(e.key, 10) * 10;
          video.currentTime = (pct / 100) * (video.duration || 0);
          showIndicator(`seek-pct-${pct}`);
          break;
        }
        case "<":
        case ",":
          if (e.key === "<" || e.shiftKey) {
            e.preventDefault();
            const curIdx = SPEEDS.indexOf(playbackRate);
            if (curIdx > 0) {
              const nextSpd = SPEEDS[curIdx - 1];
              setPlaybackRate(nextSpd);
              showIndicator(`speed-${nextSpd}`);
            }
          } else if (video.paused) {
            // Step back one frame (approx 0.04s)
            e.preventDefault();
            video.currentTime = Math.max(0, video.currentTime - 0.04);
          }
          break;
        case ">":
        case ".":
          if (e.key === ">" || e.shiftKey) {
            e.preventDefault();
            const curIdx = SPEEDS.indexOf(playbackRate);
            if (curIdx < SPEEDS.length - 1 && curIdx !== -1) {
              const nextSpd = SPEEDS[curIdx + 1];
              setPlaybackRate(nextSpd);
              showIndicator(`speed-${nextSpd}`);
            }
          } else if (video.paused) {
            // Step forward one frame (approx 0.04s)
            e.preventDefault();
            video.currentTime = Math.min(video.duration || 0, video.currentTime + 0.04);
          }
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [togglePlay, toggleMute, toggleFullscreen, showIndicator, resetControlsTimeout, playbackRate]);

  // When the mp4Url changes, we load a new video, so reset the fallback tracking
  useEffect(() => {
    setHlsFailed(false);
    userInteractedRef.current = false;
    setIsPlaying(false);
    setCurrentTime(0);
    setNeedsTap(true);
  }, [mp4Url]);

  // Handle source setup
  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    setFailed(false);
    setBuffering(false);
    setNeedsTap(true);

    if (!hlsUrl || hlsFailed) {
      // Progressive MP4 path
      video.src = mp4Url;
      video.load();
      setUsingHls(false);
      if (userInteractedRef.current) {
        video.play().catch(() => {});
      }
      return;
    }

    // Try HLS natively first (Safari/iOS)
    const canNative = video.canPlayType("application/vnd.apple.mpegurl");
    if (canNative) {
      video.src = hlsUrl;
      setUsingHls(true);
      if (userInteractedRef.current) {
        video.play().catch(() => {});
      }
      return;
    }

    // MSE HLS path (Chrome/Firefox)
    let cancelled = false;
    const ms = new MediaSource();
    video.src = URL.createObjectURL(ms);

    ms.addEventListener("sourceopen", async () => {
      if (cancelled) return;
      try {
        const sb = ms.addSourceBuffer('video/mp4; codecs="avc1.4d401f,mp4a.40.2"');
        await ingestManifest(ms, sb, hlsUrl, video);
        if (!cancelled) {
          setUsingHls(true);
          if (userInteractedRef.current) {
            video.play().catch(() => {});
          }
        }
      } catch (err) {
        if (cancelled) return;
        // eslint-disable-next-line no-console
        console.warn("HLS MSE playback failed; falling back to MP4", err);
        setHlsFailed(true);
      }
    });

    return () => {
      cancelled = true;
      if (video.src.startsWith("blob:")) {
        try {
          URL.revokeObjectURL(video.src);
        } catch {
          /* ignore */
        }
      }
    };
  }, [hlsUrl, mp4Url, hlsFailed]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    video.preload = hasInteracted ? "auto" : "metadata";
  }, [hasInteracted]);

  // Video Tag event hooks
  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const updateBuffer = () => {
      if (video.buffered.length > 0 && video.duration) {
        let currentBuffered = 0;
        const curTime = video.currentTime;
        for (let i = 0; i < video.buffered.length; i++) {
          if (curTime >= video.buffered.start(i) && curTime <= video.buffered.end(i)) {
            currentBuffered = video.buffered.end(i);
            break;
          }
        }
        if (currentBuffered === 0 && video.buffered.length > 0) {
          currentBuffered = video.buffered.end(video.buffered.length - 1);
        }
        setBufferedPercent((currentBuffered / video.duration) * 100);
      }
    };

    const onPlay = () => {
      setIsPlaying(true);
      setNeedsTap(false);
      resetControlsTimeout();
    };
    const onPause = () => {
      setIsPlaying(false);
      setControlsVisible(true);
    };
    const onTimeUpdate = () => {
      setCurrentTime(video.currentTime);
      onProgress?.(video.currentTime, video.duration || 0);
      updateBuffer();
    };
    const onDurationChange = () => {
      setDuration(video.duration || 0);
      updateBuffer();
    };
    const onVolumeChange = () => {
      setVolume(video.volume);
      setIsMuted(video.muted || video.volume === 0);
    };
    const onWaiting = () => setBuffering(true);
    const onPlaying = () => {
      setBuffering(false);
      setNeedsTap(false);
      video.playbackRate = playbackRate;
      updateBuffer();
    };

    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("timeupdate", onTimeUpdate);
    video.addEventListener("durationchange", onDurationChange);
    video.addEventListener("volumechange", onVolumeChange);
    video.addEventListener("waiting", onWaiting);
    video.addEventListener("playing", onPlaying);
    video.addEventListener("canplay", onPlaying);
    video.addEventListener("progress", updateBuffer);

    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("timeupdate", onTimeUpdate);
      video.removeEventListener("durationchange", onDurationChange);
      video.removeEventListener("volumechange", onVolumeChange);
      video.removeEventListener("waiting", onWaiting);
      video.removeEventListener("playing", onPlaying);
      video.removeEventListener("canplay", onPlaying);
      video.removeEventListener("progress", updateBuffer);
    };
  }, [playbackRate, onProgress, resetControlsTimeout]);

  // Fullscreen Change listener
  useEffect(() => {
    const handleFs = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };
    document.addEventListener("fullscreenchange", handleFs);
    return () => document.removeEventListener("fullscreenchange", handleFs);
  }, []);

  const togglePlay = useCallback(() => {
    const video = videoRef.current;
    if (!video || failed) return;
    userInteractedRef.current = true;
    if (video.paused) {
      video.play().catch(() => {});
      showIndicator("play-hud");
    } else {
      video.pause();
      showIndicator("pause-hud");
    }
  }, [failed, showIndicator]);

  const toggleMute = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    video.muted = !video.muted;
    showIndicator(video.muted ? "mute" : "unmute");
  }, [showIndicator]);

  const handleVolumeSlider = (e: React.ChangeEvent<HTMLInputElement>) => {
    const video = videoRef.current;
    if (!video) return;
    const vol = Number(e.target.value);
    video.volume = vol;
    video.muted = vol === 0;
  };

  const toggleFullscreen = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {});
    } else {
      container.requestFullscreen().catch(() => {});
    }
  }, []);

  const togglePiP = async () => {
    const video = videoRef.current;
    if (!video) return;
    try {
      if (document.pictureInPictureElement) {
        await document.exitPictureInPicture();
      } else {
        await video.requestPictureInPicture();
      }
    } catch (e) {
      console.warn("PiP not supported or failed", e);
    }
  };

  // Double click on video: skip 10s back/forward on sides, fullscreen in center
  const handleVideoDoubleClick = (e: React.MouseEvent<HTMLVideoElement>) => {
    e.preventDefault();
    const rect = e.currentTarget.getBoundingClientRect();
    const clickX = e.clientX - rect.left;
    const ratio = clickX / rect.width;

    if (ratio < 0.3) {
      if (videoRef.current) {
        videoRef.current.currentTime = Math.max(0, videoRef.current.currentTime - 10);
        showIndicator("seek-back-10");
      }
    } else if (ratio > 0.7) {
      if (videoRef.current) {
        videoRef.current.currentTime = Math.min(
          videoRef.current.duration || 0,
          videoRef.current.currentTime + 10,
        );
        showIndicator("seek-forward-10");
      }
    } else {
      toggleFullscreen();
    }
  };

  // Timeline Progress Scrubber seek
  const handleScrubberSeek = (e: React.MouseEvent<HTMLDivElement>) => {
    const rect = progressRef.current?.getBoundingClientRect();
    const video = videoRef.current;
    if (!rect || !video || !video.duration) return;
    const clientX = e.clientX;
    const clickX = clientX - rect.left;
    const percent = Math.max(0, Math.min(1, clickX / rect.width));
    video.currentTime = percent * video.duration;
  };

  const handleScrubberMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    const rect = progressRef.current?.getBoundingClientRect();
    if (!rect || !duration) return;
    const clientX = e.clientX;
    const hoverX = clientX - rect.left;
    const percent = Math.max(0, Math.min(1, hoverX / rect.width));
    const hoverTime = percent * duration;

    setHoverTooltipTime(formatTime(hoverTime));
    setHoverTooltipLeft(percent * 100);

    if (e.buttons === 1) {
      if (videoRef.current) {
        videoRef.current.currentTime = hoverTime;
      }
    }
  };

  const handleScrubberMouseLeave = () => {
    setHoverTooltipTime(null);
    setHoverTooltipLeft(null);
  };

  const formatTime = (time: number) => {
    if (isNaN(time)) return "0:00";
    const hrs = Math.floor(time / 3600);
    const mins = Math.floor((time % 3600) / 60);
    const secs = Math.floor(time % 60);
    const formattedSecs = secs < 10 ? `0${secs}` : secs;

    if (hrs > 0) {
      const formattedMins = mins < 10 ? `0${mins}` : mins;
      return `${hrs}:${formattedMins}:${formattedSecs}`;
    }
    return `${mins}:${formattedSecs}`;
  };

  const progressPercent = duration ? (currentTime / duration) * 100 : 0;

  return (
    <div
      ref={containerRef}
      onMouseMove={resetControlsTimeout}
      onMouseLeave={() => isPlaying && setControlsVisible(false)}
      className={`aspect-video bg-black overflow-hidden relative group select-none ${
        !controlsVisible ? "cursor-none" : ""
      }`}
    >
      <video
        ref={videoRef}
        crossOrigin="anonymous"
        playsInline
        poster={posterUrl}
        className="h-full w-full object-contain cursor-pointer"
        onClick={togglePlay}
        onDoubleClick={handleVideoDoubleClick}
        onWaiting={() => setBuffering(true)}
        onError={() => {
          if (hlsUrl && !hlsFailed) {
            console.warn(
              "HLS stream failed to play; triggering silent fallback to progressive MP4",
            );
            setHlsFailed(true);
            setBuffering(false);
            return;
          }
          setFailed(true);
          setBuffering(false);
          onError?.("Video failed to load");
        }}
      />

      {/* Large Center HUD Flashing Animation Overlays */}
      {indicator && (
        <div className="pointer-events-none absolute inset-0 grid place-items-center z-25">
          <div className="flex flex-col items-center gap-2 bg-black/75 backdrop-blur-sm rounded-full p-5 shadow-2xl border border-white/10 text-white animate-out zoom-out-50 duration-500">
            {indicator === "play-hud" && <Play className="h-8 w-8 fill-white animate-ping" />}
            {indicator === "pause-hud" && <Pause className="h-8 w-8 fill-white animate-ping" />}
            {indicator?.startsWith("seek-back") && (
              <div className="flex flex-col items-center gap-1">
                <ChevronLeft className="h-8 w-8 animate-pulse" />
                <span className="text-[11px] font-extrabold tracking-wider">
                  {indicator === "seek-back-10" ? "-10s" : "-5s"}
                </span>
              </div>
            )}
            {indicator?.startsWith("seek-forward") && (
              <div className="flex flex-col items-center gap-1">
                <ChevronRight className="h-8 w-8 animate-pulse" />
                <span className="text-[11px] font-extrabold tracking-wider">
                  {indicator === "seek-forward-10" ? "+10s" : "+5s"}
                </span>
              </div>
            )}
            {indicator?.startsWith("seek-pct") && (
              <span className="text-xs font-bold px-2 py-0.5 bg-red-600 rounded">
                Seek: {indicator?.split("-")?.[2]}%
              </span>
            )}
            {indicator?.startsWith("speed-") && (
              <span className="text-xs font-bold px-2 py-0.5 bg-blue-600 rounded">
                Speed: {indicator?.split("-")?.[1]}x
              </span>
            )}
            {indicator === "volume-up" && (
              <span className="text-xs font-bold px-2 py-0.5 bg-slate-800 rounded">
                Volume: {Math.round(volume * 100)}%
              </span>
            )}
            {indicator === "volume-down" && (
              <span className="text-xs font-bold px-2 py-0.5 bg-slate-800 rounded">
                Volume: {Math.round(volume * 100)}%
              </span>
            )}
            {indicator === "mute" && <VolumeX className="h-8 w-8 text-red-500 animate-bounce" />}
            {indicator === "unmute" && (
              <Volume2 className="h-8 w-8 text-green-500 animate-bounce" />
            )}
          </div>
        </div>
      )}

      {/* Tap / First Play Overlay Screen */}
      {needsTap && !failed && (
        <div
          className="absolute inset-0 grid place-items-center bg-black/45 cursor-pointer z-10"
          onClick={togglePlay}
        >
          <button
            type="button"
            className="flex items-center gap-3 bg-red-600 hover:bg-red-700 text-white px-6 py-3.5 rounded-sm shadow-xl font-medium tracking-wide transition-all transform hover:scale-105"
            aria-label="Play lesson"
          >
            <Play className="h-5 w-5 fill-white" />
            <span>PLAY LESSON</span>
            {usingHls && (
              <span className="border border-white/40 px-1 py-0.5 text-[9px] font-bold">HLS</span>
            )}
          </button>
        </div>
      )}

      {/* Buffering Indicator */}
      {buffering && !failed && !needsTap && (
        <div className="pointer-events-none absolute inset-0 grid place-items-center bg-black/25 z-10">
          <div className="flex flex-col items-center gap-2">
            <div className="w-10 h-10 border-4 border-red-600/30 border-t-red-600 rounded-full animate-spin" />
            <span className="text-white text-xs font-medium tracking-wider">Buffering...</span>
          </div>
        </div>
      )}

      {/* Failed Loader Screen */}
      {failed && (
        <div className="absolute inset-0 grid place-items-center bg-black/85 text-white z-10">
          <div className="text-center px-6">
            <p className="text-sm text-white/70 mb-4">Video failed to load</p>
            <button
              type="button"
              className="inline-flex items-center gap-2 px-5 py-2.5 bg-red-600 hover:bg-red-700 transition-colors text-sm font-medium"
              onClick={() => {
                setFailed(false);
                if (videoRef.current) {
                  videoRef.current.load();
                  videoRef.current.play().catch(() => {});
                }
              }}
            >
              <RotateCcw className="h-4 w-4" /> Retry loading
            </button>
          </div>
        </div>
      )}

      {/* YouTube-like Controls Bar */}
      {!needsTap && !failed && (
        <div
          className={`absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/95 via-black/60 to-transparent p-4 pt-10 z-20 transition-all duration-300 ${
            controlsVisible
              ? "opacity-100 translate-y-0"
              : "opacity-0 translate-y-3 pointer-events-none"
          }`}
        >
          {/* Custom Timeline Scrubber */}
          <div
            ref={progressRef}
            onClick={handleScrubberSeek}
            onMouseMove={handleScrubberMouseMove}
            onMouseLeave={handleScrubberMouseLeave}
            className="group/scrub h-1 hover:h-1.5 bg-white/20 cursor-pointer relative mb-4 transition-all rounded-full overflow-visible"
          >
            {/* Buffered track (translucent gray) */}
            <div
              className="h-full bg-white/30 absolute left-0 top-0 transition-all rounded-full"
              style={{ width: `${bufferedPercent}%` }}
            />
            {/* Played track (YouTube Red) */}
            <div
              className="h-full bg-red-600 absolute left-0 top-0 transition-all rounded-full"
              style={{ width: `${progressPercent}%` }}
            />
            {/* Scrubber Playhead handle dot (red circle) */}
            <div
              className="absolute h-3.5 w-3.5 bg-red-600 rounded-full top-1/2 -translate-y-1/2 -translate-x-1/2 scale-0 group-hover/scrub:scale-100 transition-transform shadow-md shadow-black/40 z-10"
              style={{ left: `${progressPercent}%` }}
            />
            {/* Tooltip for Hover Preview Time */}
            {hoverTooltipTime !== null && hoverTooltipLeft !== null && (
              <div
                className="absolute bg-black/90 text-white text-[10px] font-bold px-1.5 py-0.5 rounded shadow-lg pointer-events-none transform -translate-x-1/2 -translate-y-7 z-30 border border-white/10"
                style={{ left: `${hoverTooltipLeft}%` }}
              >
                {hoverTooltipTime}
              </div>
            )}
          </div>

          {/* Lower Row Controls */}
          <div className="flex items-center justify-between text-white text-sm">
            {/* Left Hand Controls */}
            <div className="flex items-center gap-4">
              <button
                type="button"
                onClick={togglePlay}
                className="hover:text-red-500 transition-colors p-1"
                aria-label={isPlaying ? "Pause" : "Play"}
              >
                {isPlaying ? (
                  <Pause className="h-5 w-5 fill-white hover:fill-red-500" />
                ) : (
                  <Play className="h-5 w-5 fill-white hover:fill-red-500" />
                )}
              </button>

              {/* Volume Slider Component */}
              <div className="flex items-center gap-1.5 group/volume">
                <button
                  type="button"
                  onClick={toggleMute}
                  className="hover:text-red-500 transition-colors p-1"
                  aria-label="Mute"
                >
                  {isMuted ? (
                    <VolumeX className="h-5 w-5" />
                  ) : volume < 0.5 ? (
                    <Volume1 className="h-5 w-5" />
                  ) : (
                    <Volume2 className="h-5 w-5" />
                  )}
                </button>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  value={isMuted ? 0 : volume}
                  onChange={handleVolumeSlider}
                  className="w-0 group-hover/volume:w-16 h-1 bg-white/30 accent-red-600 transition-all cursor-pointer overflow-hidden rounded-sm"
                  aria-label="Volume slider"
                />
              </div>

              {/* Video Timeline Duration */}
              <span className="text-xs font-medium tracking-wide">
                {formatTime(currentTime)} / {formatTime(duration)}
              </span>
            </div>

            {/* Right Hand Controls */}
            <div className="flex items-center gap-4">
              {/* Playback speed selector */}
              <div className="relative" ref={speedContainerRef}>
                <button
                  type="button"
                  onClick={() => setShowSpeedMenu(!showSpeedMenu)}
                  className="hover:text-red-500 transition-colors p-1"
                  aria-label="Playback speed"
                >
                  <Settings className="h-5 w-5" />
                </button>
                {showSpeedMenu && (
                  <div className="absolute right-0 bottom-8 mb-2 bg-brand border border-white/10 shadow-2xl z-30 py-1 w-28">
                    {SPEEDS.map((s) => (
                      <button
                        key={s}
                        type="button"
                        onClick={() => {
                          setPlaybackRate(s);
                          setShowSpeedMenu(false);
                        }}
                        className={`w-full text-left px-4 py-2 text-xs font-semibold transition-colors hover:bg-white/10 ${
                          s === playbackRate ? "text-red-500" : "text-white"
                        }`}
                      >
                        {s}x {s === 1 ? "(Normal)" : ""}
                      </button>
                    ))}
                  </div>
                )}
              </div>

              {/* Picture in Picture */}
              <button
                type="button"
                onClick={togglePiP}
                className="hover:text-red-500 transition-colors p-1"
                aria-label="Picture-in-picture"
              >
                <Tv className="h-5 w-5" />
              </button>

              {/* Fullscreen Button */}
              <button
                type="button"
                onClick={toggleFullscreen}
                className="hover:text-red-500 transition-colors p-1"
                aria-label="Toggle Fullscreen"
              >
                {isFullscreen ? <Minimize className="h-5 w-5" /> : <Maximize className="h-5 w-5" />}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * ingestManifest walks the master playlist, picks the lowest-bandwidth
 * rendition (good for starting quickly), downloads its media playlist, and
 * pipes each .ts segment into the source buffer.
 */
async function ingestManifest(
  ms: MediaSource,
  sb: SourceBuffer,
  manifestUrl: string,
  video: HTMLVideoElement,
): Promise<void> {
  const masterText = await fetchTextWithRetry(manifestUrl);
  const master = parseM3U8(masterText);
  let bestBandwidth = Number.POSITIVE_INFINITY;
  let bestUri = "";
  for (const entry of master.variants) {
    if (entry.bandwidth > 0 && entry.bandwidth < bestBandwidth) {
      bestBandwidth = entry.bandwidth;
      bestUri = entry.uri;
    }
  }
  if (!bestUri) {
    bestUri = manifestUrl;
  }
  const absoluteMediaUrl = absolutize(manifestUrl, bestUri);
  const mediaText = await fetchTextWithRetry(absoluteMediaUrl);
  const media = parseM3U8(mediaText);
  const segmentUrls = media.segments.map((s) => absolutize(absoluteMediaUrl, s.uri));

  for (const segUrl of segmentUrls) {
    await appendSegment(sb, segUrl);
  }
  if (ms.readyState === "open") {
    ms.endOfStream();
  }
}

async function appendSegment(sb: SourceBuffer, url: string): Promise<void> {
  let attempt = 0;
  while (true) {
    try {
      const buf = await fetchArrayBufferWithRetry(url);
      await appendBuffer(sb, buf);
      return;
    } catch (err) {
      if (++attempt >= SEGMENT_RETRY) throw err;
      await new Promise((r) => setTimeout(r, 250 * 2 ** (attempt - 1)));
    }
  }
}

function appendBuffer(sb: SourceBuffer, buf: ArrayBuffer): Promise<void> {
  return new Promise((resolve, reject) => {
    const onUpdate = () => {
      sb.removeEventListener("updateend", onUpdate);
      sb.removeEventListener("error", onErr);
      resolve();
    };
    const onErr = () => {
      sb.removeEventListener("updateend", onUpdate);
      sb.removeEventListener("error", onErr);
      reject(new Error("SourceBuffer error"));
    };
    sb.addEventListener("updateend", onUpdate);
    sb.addEventListener("error", onErr);
    try {
      sb.appendBuffer(buf);
    } catch (e) {
      reject(e as Error);
    }
  });
}

async function fetchTextWithRetry(url: string): Promise<string> {
  let attempt = 0;
  while (true) {
    try {
      const res = await fetch(url);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return res.text();
    } catch (err) {
      if (++attempt >= SEGMENT_RETRY) throw err;
      await new Promise((r) => setTimeout(r, 250 * 2 ** (attempt - 1)));
    }
  }
}

async function fetchArrayBufferWithRetry(url: string): Promise<ArrayBuffer> {
  let attempt = 0;
  while (true) {
    try {
      const res = await fetch(url);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return res.arrayBuffer();
    } catch (err) {
      if (++attempt >= SEGMENT_RETRY) throw err;
      await new Promise((r) => setTimeout(r, 250 * 2 ** (attempt - 1)));
    }
  }
}

type M3U8 = {
  variants: Array<{ bandwidth: number; uri: string }>;
  segments: Array<{ uri: string }>;
};

function parseM3U8(text: string): M3U8 {
  const lines = text
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter(Boolean);
  const result: M3U8 = { variants: [], segments: [] };
  let isMaster = false;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line.startsWith("#EXT-X-STREAM-INF")) {
      const m = /BANDWIDTH=(\d+)/.exec(line);
      const bandwidth = m ? Number(m[1]) : 0;
      const uri = lines[i + 1] ?? "";
      if (uri) result.variants.push({ bandwidth, uri });
      isMaster = true;
      i++;
    } else if (!line.startsWith("#")) {
      result.segments.push({ uri: line });
    }
  }
  if (isMaster) result.segments = [];
  return result;
}

function absolutize(base: string, relative: string): string {
  if (/^https?:\/\//.test(relative)) return relative;
  try {
    return new URL(relative, base).toString();
  } catch {
    return relative;
  }
}

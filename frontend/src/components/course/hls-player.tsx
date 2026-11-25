/**
 * HLSPlayer
 *
 * Prefers adaptive HLS (when the signed-URL response has has_hls=true) and
 * gracefully falls back to progressive MP4. Chrome/Firefox/Edge/Android use
 * hls.js (Media Source Extensions) for adaptive bitrate, seeking and network
 * recovery; Safari/iOS use native HLS. The previous hand-written MSE loader
 * pushed MPEG-TS segments into an MP4 SourceBuffer (always failing), and
 * downloaded every segment of the lowest rendition up front.
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
import Hls from "hls.js/light";
import { API_BASE_URL } from "@/lib/api/client";
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
  /** Resume position in seconds (ignored when within the last 5 seconds). */
  startPosition?: number;
  /** Called on playback events relevant for progress persistence. */
  onPlaybackEvent?: (
    event: "pause" | "ended" | "seeked",
    currentTime: number,
    duration: number,
  ) => void;
}

type QualityLevel = { index: number; height: number; bitrate: number };

/** Resolve backend-relative HLS URLs (/v1/stream/...) against the API host. */
function resolveMediaUrl(url: string): string {
  if (/^https?:\/\//i.test(url)) return url;
  try {
    return new URL(url, API_BASE_URL).toString();
  } catch {
    return url;
  }
}

const MAX_NETWORK_RECOVERIES = 6;
const SPEEDS = [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2];

export function HLSPlayer({
  mp4Url,
  hlsUrl,
  posterUrl,
  onError,
  hasInteracted,
  onProgress,
  startPosition,
  onPlaybackEvent,
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
  const hlsRef = useRef<Hls | null>(null);
  const startPositionRef = useRef(startPosition);
  startPositionRef.current = startPosition;
  const onPlaybackEventRef = useRef(onPlaybackEvent);
  onPlaybackEventRef.current = onPlaybackEvent;
  // Position/playing state carried across a source refresh (renewed signed
  // URLs) so playback continues where it was instead of restarting.
  const carryOverRef = useRef<{ time: number; playing: boolean } | null>(null);
  const [levels, setLevels] = useState<QualityLevel[]>([]);
  const [selectedLevel, setSelectedLevel] = useState(-1); // -1 = auto
  const [activeLevel, setActiveLevel] = useState(-1);

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

  // A new source (e.g. renewed signed URLs) gets a fresh chance at HLS. The
  // player is keyed by lesson, so lesson switches remount it entirely.
  useEffect(() => {
    setHlsFailed(false);
  }, [mp4Url]);

  // Handle source setup (re-runs when the lesson changes; tears down the
  // previous hls.js instance and media element source to free memory and
  // stop any in-flight segment downloads).
  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    setFailed(false);
    setBuffering(false);
    setNeedsTap(true);
    setLevels([]);
    setSelectedLevel(-1);
    setActiveLevel(-1);

    const carry = carryOverRef.current;
    carryOverRef.current = null;
    const resume = carry ? carry.time : (startPositionRef.current ?? 0);
    if (carry?.playing) userInteractedRef.current = true;
    const applyResume = () => {
      if (resume > 1 && video.duration && resume < video.duration - 5) {
        video.currentTime = resume;
      }
    };

    let hls: Hls | null = null;
    let detachPlayListener = () => {};
    const src = hlsUrl && !hlsFailed ? resolveMediaUrl(hlsUrl) : null;

    if (src && Hls.isSupported()) {
      hls = new Hls({
        // Conservative initial estimate (~800 kbps) suits mobile networks;
        // ABR ramps up after the first segments are measured.
        abrEwmaDefaultEstimate: 800_000,
        startLevel: -1,
        capLevelToPlayerSize: true,
        maxBufferLength: 30,
        maxMaxBufferLength: 60,
        backBufferLength: 30,
        startPosition: resume > 1 ? resume : -1,
        // Fetch only the playlists until the student presses play: no
        // segment downloads (mobile data) for lessons that are just opened.
        autoStartLoad: false,
        enableWorker: true,
        fragLoadPolicy: {
          default: {
            maxTimeToFirstByteMs: 10_000,
            maxLoadTimeMs: 60_000,
            timeoutRetry: { maxNumRetry: 4, retryDelayMs: 500, maxRetryDelayMs: 8_000 },
            errorRetry: { maxNumRetry: 6, retryDelayMs: 1_000, maxRetryDelayMs: 8_000 },
          },
        },
      });
      hlsRef.current = hls;
      let networkRecoveries = 0;
      let mediaRecoveries = 0;
      hls.on(Hls.Events.MANIFEST_PARSED, (_event, data) => {
        setLevels(
          data.levels
            .map((level, index) => ({ index, height: level.height, bitrate: level.bitrate }))
            .sort((x, y) => y.height - x.height),
        );
        setUsingHls(true);
        if (userInteractedRef.current) video.play().catch(() => {});
      });
      hls.on(Hls.Events.LEVEL_SWITCHED, (_event, data) => setActiveLevel(data.level));
      hls.on(Hls.Events.FRAG_LOADED, () => {
        networkRecoveries = 0;
      });
      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (!data.fatal || !hls) return;
        if (data.type === Hls.ErrorTypes.NETWORK_ERROR && !navigator.onLine) {
          // Offline: keep playing what is buffered and resume loading from the
          // current position when the "online" event fires (see below).
          setBuffering(true);
          return;
        }
        if (
          data.type === Hls.ErrorTypes.NETWORK_ERROR &&
          networkRecoveries < MAX_NETWORK_RECOVERIES
        ) {
          networkRecoveries++;
          setBuffering(true);
          const delay = Math.min(5000, 1000 * networkRecoveries);
          window.setTimeout(() => hls?.startLoad(video.currentTime || -1), delay);
        } else if (data.type === Hls.ErrorTypes.MEDIA_ERROR && mediaRecoveries < 2) {
          mediaRecoveries++;
          hls.recoverMediaError();
        } else {
          console.warn("HLS playback failed; falling back to progressive MP4", data.details);
          setHlsFailed(true);
        }
      });
      hls.loadSource(src);
      hls.attachMedia(video);
      const startLoadOnPlay = () => hls?.startLoad(resume > 1 ? resume : -1);
      video.addEventListener("play", startLoadOnPlay, { once: true });
      detachPlayListener = () => video.removeEventListener("play", startLoadOnPlay);
    } else if (src && video.canPlayType("application/vnd.apple.mpegurl")) {
      // Safari / iOS native HLS
      video.src = src;
      setUsingHls(true);
      video.addEventListener("loadedmetadata", applyResume, { once: true });
    } else {
      // Progressive MP4 fallback (range requests; preload metadata only)
      video.src = mp4Url;
      setUsingHls(false);
      video.addEventListener("loadedmetadata", applyResume, { once: true });
    }
    if (!hls && userInteractedRef.current) {
      video.play().catch(() => {});
    }

    // Resume downloading when connectivity returns.
    const onOnline = () => {
      if (hlsRef.current) {
        hlsRef.current.startLoad(video.currentTime || -1);
        if (!video.paused) video.play().catch(() => {});
      } else if (video.error || video.networkState === video.NETWORK_NO_SOURCE) {
        // Progressive MP4: reload and continue from the same position.
        const at = video.currentTime;
        video.load();
        video.addEventListener("loadedmetadata", () => (video.currentTime = at), { once: true });
        video.play().catch(() => {});
      }
    };
    window.addEventListener("online", onOnline);

    return () => {
      if (video.currentTime > 0) {
        carryOverRef.current = { time: video.currentTime, playing: !video.paused };
      }
      window.removeEventListener("online", onOnline);
      video.removeEventListener("loadedmetadata", applyResume);
      detachPlayListener();
      if (hls) {
        hls.destroy();
      }
      hlsRef.current = null;
      video.removeAttribute("src");
      video.load();
    };
  }, [hlsUrl, mp4Url, hlsFailed]);

  const selectQuality = (index: number) => {
    setSelectedLevel(index);
    if (hlsRef.current) {
      // currentLevel switches immediately (flushes buffer); -1 restores ABR.
      hlsRef.current.currentLevel = index;
    }
  };

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    // Never preload whole files: metadata only until the student presses
    // play; afterwards the browser / hls.js buffer just ahead of playback.
    void hasInteracted;
    video.preload = "metadata";
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
      onPlaybackEventRef.current?.("pause", video.currentTime, video.duration || 0);
    };
    const onEnded = () => {
      onPlaybackEventRef.current?.("ended", video.currentTime, video.duration || 0);
    };
    const onSeeked = () => {
      onPlaybackEventRef.current?.("seeked", video.currentTime, video.duration || 0);
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
    video.addEventListener("ended", onEnded);
    video.addEventListener("seeked", onSeeked);

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
      video.removeEventListener("ended", onEnded);
      video.removeEventListener("seeked", onSeeked);
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

  // Timeline Progress Scrubber seek
  const handleScrubberSeek = (e: React.PointerEvent<HTMLDivElement>) => {
    const rect = progressRef.current?.getBoundingClientRect();
    const video = videoRef.current;
    if (!rect || !video || !video.duration) return;
    const clientX = e.clientX;
    const clickX = clientX - rect.left;
    const percent = Math.max(0, Math.min(1, clickX / rect.width));
    video.currentTime = percent * video.duration;
  };

  const handleScrubberMouseMove = (e: React.PointerEvent<HTMLDivElement>) => {
    const rect = progressRef.current?.getBoundingClientRect();
    if (!rect || !duration) return;
    const clientX = e.clientX;
    const hoverX = clientX - rect.left;
    const percent = Math.max(0, Math.min(1, hoverX / rect.width));
    const hoverTime = percent * duration;

    setHoverTooltipTime(formatTime(hoverTime));
    setHoverTooltipLeft(percent * 100);

    if (e.buttons === 1 || e.currentTarget.hasPointerCapture?.(e.pointerId)) {
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
            role="slider"
            tabIndex={0}
            aria-label="Seek"
            aria-valuemin={0}
            aria-valuemax={Math.round(duration) || 0}
            aria-valuenow={Math.round(currentTime)}
            aria-valuetext={`${formatTime(currentTime)} of ${formatTime(duration)}`}
            onPointerDown={(e) => {
              e.currentTarget.setPointerCapture(e.pointerId);
              handleScrubberSeek(e);
            }}
            onPointerMove={handleScrubberMouseMove}
            onPointerLeave={handleScrubberMouseLeave}
            className="group/scrub h-1.5 sm:h-1 hover:h-1.5 touch-none py-0 before:absolute before:-inset-y-3 before:inset-x-0 before:content-[''] bg-white/20 cursor-pointer relative mb-4 transition-all rounded-full overflow-visible"
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
            <div className="flex items-center gap-2 sm:gap-4 min-w-0">
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
                  className="hidden sm:block w-0 group-hover/volume:w-16 focus:w-16 h-1 bg-white/30 accent-red-600 transition-all cursor-pointer overflow-hidden rounded-sm"
                  aria-label="Volume slider"
                />
              </div>

              {/* Video Timeline Duration */}
              <span className="text-[11px] sm:text-xs font-medium tracking-wide tabular-nums whitespace-nowrap">
                {formatTime(currentTime)} / {formatTime(duration)}
              </span>
            </div>

            {/* Right Hand Controls */}
            <div className="flex items-center gap-2 sm:gap-4 shrink-0">
              {/* Playback speed selector */}
              <div className="relative" ref={speedContainerRef}>
                <button
                  type="button"
                  onClick={() => setShowSpeedMenu(!showSpeedMenu)}
                  className="hover:text-red-500 transition-colors p-1"
                  aria-label="Playback settings"
                  aria-haspopup="menu"
                  aria-expanded={showSpeedMenu}
                >
                  <Settings className="h-5 w-5" />
                </button>
                {showSpeedMenu && (
                  <div
                    role="menu"
                    className="absolute right-0 bottom-8 mb-2 bg-brand border border-white/10 shadow-2xl z-30 py-1 w-36 max-h-[60vh] overflow-y-auto"
                  >
                    {levels.length > 1 && (
                      <>
                        <p className="px-4 pt-1 pb-1 text-[10px] uppercase tracking-wider text-white/50">
                          Quality
                        </p>
                        <button
                          type="button"
                          role="menuitemradio"
                          aria-checked={selectedLevel === -1}
                          onClick={() => {
                            selectQuality(-1);
                            setShowSpeedMenu(false);
                          }}
                          className={`w-full text-left px-4 py-2 text-xs font-semibold transition-colors hover:bg-white/10 ${
                            selectedLevel === -1 ? "text-red-500" : "text-white"
                          }`}
                        >
                          Auto
                          {selectedLevel === -1 && activeLevel >= 0
                            ? ` (${levels.find((l) => l.index === activeLevel)?.height ?? ""}p)`
                            : ""}
                        </button>
                        {levels.map((level) => (
                          <button
                            key={level.index}
                            type="button"
                            role="menuitemradio"
                            aria-checked={selectedLevel === level.index}
                            onClick={() => {
                              selectQuality(level.index);
                              setShowSpeedMenu(false);
                            }}
                            className={`w-full text-left px-4 py-2 text-xs font-semibold transition-colors hover:bg-white/10 ${
                              selectedLevel === level.index ? "text-red-500" : "text-white"
                            }`}
                          >
                            {level.height}p
                          </button>
                        ))}
                        <div className="my-1 border-t border-white/10" />
                      </>
                    )}
                    <p className="px-4 pt-1 pb-1 text-[10px] uppercase tracking-wider text-white/50">
                      Speed
                    </p>
                    {SPEEDS.map((s) => (
                      <button
                        key={s}
                        type="button"
                        role="menuitemradio"
                        aria-checked={s === playbackRate}
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
                className="hidden sm:inline-flex hover:text-red-500 transition-colors p-1"
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

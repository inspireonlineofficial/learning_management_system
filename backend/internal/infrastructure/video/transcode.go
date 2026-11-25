package video

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// HLSSpec describes one rendition in the bitrate ladder. The defaults are
// tuned for a typical LMS lecture: 360p is the safety net for low-bandwidth
// mobile viewers, 720p is the default, 1080p rewards broadband users.
type HLSSpec struct {
	Name         string // folder name under hls/ (e.g. "v0", "v1")
	Width        int
	Height       int
	VideoBitrate string // e.g. "800k", "2500k"
	MaxBitrate   string // e.g. "900k", "2800k"
	BufSize      string // e.g. "1200k", "3500k"
	AudioBitrate string // e.g. "96k"
	Level        string // H.264 level, e.g. "3.1"
}

// DefaultLadder is the rendition set we transcode to. Ordered low → high so
// the master playlist is naturally sorted for ABR selection.
// DefaultLadder is the adaptive bitrate ladder (H.264 + AAC). Rungs taller
// than the source are dropped (see ladderForSourceHeight), so a 720p upload
// produces 240p/360p/480p/720p. Bitrates target classroom content (slides,
// screen recordings, talking heads) on mobile networks: the 240p/360p rungs
// keep playback going on slow 3G/4G connections common for students in
// Bangladesh, while 720p/1080p serve broadband.
var DefaultLadder = []HLSSpec{
	{Name: "v0", Width: 426, Height: 240, VideoBitrate: "300k", MaxBitrate: "350k", BufSize: "600k", AudioBitrate: "64k", Level: "3.0"},
	{Name: "v1", Width: 640, Height: 360, VideoBitrate: "600k", MaxBitrate: "700k", BufSize: "1200k", AudioBitrate: "96k", Level: "3.0"},
	{Name: "v2", Width: 854, Height: 480, VideoBitrate: "1000k", MaxBitrate: "1150k", BufSize: "2000k", AudioBitrate: "96k", Level: "3.1"},
	{Name: "v3", Width: 1280, Height: 720, VideoBitrate: "2200k", MaxBitrate: "2500k", BufSize: "4400k", AudioBitrate: "128k", Level: "3.1"},
	{Name: "v4", Width: 1920, Height: 1080, VideoBitrate: "4200k", MaxBitrate: "4800k", BufSize: "8400k", AudioBitrate: "128k", Level: "4.0"},
}

// SegmentSeconds is the HLS segment duration. Keyframes are forced on this
// boundary in every rendition so segments are aligned for seamless ABR
// switching.
const SegmentSeconds = 4

// TranscodeResult is what TranscodeToHLS returns to the caller. The caller is
// responsible for uploading the produced files to storage and updating the
// video row.
type TranscodeResult struct {
	// MasterManifest is the bytes of the master .m3u8. Already references the
	// per-rendition playlists via relative paths.
	MasterManifest []byte
	// Renditions maps rendition name → (playlist bytes, segment file names).
	Renditions map[string]Rendition
	// DurationSeconds is the source video duration, useful for populating the
	// duration_seconds column.
	DurationSeconds int
	// OutputDir is the temp directory holding the produced files. The caller
	// is responsible for cleanup. We return it so the upload step can walk
	// every produced file without the transcoder having to know storage.
	OutputDir string
}

// Rendition is one encoded variant of the source.
type Rendition struct {
	Playlist []byte
	Segments []SegmentFile
}

// SegmentFile is a single .ts segment plus its key in storage.
type SegmentFile struct {
	Key  string
	Path string
	Body io.ReadCloser
}

// TranscodeToHLS runs ffmpeg to produce a master HLS manifest and per-rendition
// playlists + segments under a temporary directory. Returns a TranscodeResult
// the caller can walk to upload everything to RustFS.
//
// The ffmpeg invocation uses one process per rendition (we could fan out
// concurrently but on a single upload worker that risks OOM on 1080p → keep it
// sequential). The master playlist is hand-written because ffmpeg's
// -master_pl_name is finicky about which extensions it recognises.
//
// The work happens on local disk so we can hand the transcoder a path instead
// of a pipe — pipes cap at ffmpeg's internal buffer and 2 GB files blow
// through that.
func TranscodeToHLS(ctx context.Context, sourcePath string, ladder []HLSSpec) (*TranscodeResult, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not available: %w", err)
	}
	if len(ladder) == 0 {
		ladder = DefaultLadder
	}

	outDir, err := os.MkdirTemp("", "hls-transcode-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	// Best-effort cleanup on error; success path is the caller's job.
	cleanup := func() { _ = os.RemoveAll(outDir) }

	duration, err := probeDuration(ctx, sourcePath)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("probe duration: %w", err)
	}

	// Do not upscale: renditions taller than the source only cost encode time
	// and storage and look no better. The lowest rung is always kept.
	srcW, srcH, sizeErr := probeVideoSize(ctx, sourcePath)
	if sizeErr == nil && srcH > 0 {
		ladder = ladderForSourceHeight(ladder, srcH)
		ladder = withSourceAspect(ladder, srcW, srcH)
	}

	hasAudio := probeHasAudio(ctx, sourcePath)
	if err := encodeLadder(ctx, sourcePath, outDir, ladder, hasAudio); err != nil {
		cleanup()
		return nil, err
	}
	renditions := make(map[string]Rendition, len(ladder))
	for _, r := range ladder {
		playlist, segs, err := collectRendition(outDir, r)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("collect %s: %w", r.Name, err)
		}
		renditions[r.Name] = Rendition{Playlist: playlist, Segments: segs}
	}

	master := buildMasterManifest(ladder, hasAudio)
	return &TranscodeResult{
		MasterManifest:  master,
		Renditions:      renditions,
		DurationSeconds: duration,
		OutputDir:       outDir,
	}, nil
}

// probeDuration pulls the container duration out of the source via ffmpeg's
// stderr -show_entries format=duration. We avoid ffprobe because we already
// require ffmpeg in PATH.
func probeDuration(ctx context.Context, path string) (int, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", path,
		"-hide_banner",
	)
	// ffmpeg prints "Duration: HH:MM:SS.xx" to stderr when run without -t/-ss.
	// We capture and parse rather than actually transcoding.
	out, _ := cmd.CombinedOutput()
	line := string(out)
	idx := strings.Index(line, "Duration: ")
	if idx < 0 {
		return 0, fmt.Errorf("could not find duration in ffmpeg output")
	}
	rest := line[idx+len("Duration: "):]
	end := strings.Index(rest, ",")
	if end < 0 {
		return 0, fmt.Errorf("malformed duration line: %q", rest)
	}
	hms := strings.TrimSpace(rest[:end])
	parts := strings.Split(hms, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("unexpected duration format: %q", hms)
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	sec, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, fmt.Errorf("parse seconds: %w", err)
	}
	total := h*3600 + m*60 + int(sec)
	return total, nil
}

// encodeRendition runs a single ffmpeg invocation for one bitrate variant.
// -hls_time 4 keeps segments short so the player can switch qualities
// quickly; the trade-off is slightly more .ts files but no perceptual quality
// difference.
// encodeLadder encodes every rendition in a single FFmpeg run: the source is
// decoded once and split, instead of being decoded once per rendition.
// Keyframes are forced every SegmentSeconds in all outputs (aligned GOPs),
// scene-cut keyframes are disabled, and constrained VBR (maxrate/bufsize)
// keeps segment sizes predictable for ABR.
func encodeLadder(ctx context.Context, src, outDir string, ladder []HLSSpec, hasAudio bool) error {
	args := buildLadderArgs(src, outDir, ladder, hasAudio)
	for _, r := range ladder {
		if err := os.MkdirAll(filepath.Join(outDir, r.Name), 0o755); err != nil {
			return err
		}
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w\n%s", err, tail(string(out), 4000))
	}
	return nil
}

func buildLadderArgs(src, outDir string, ladder []HLSSpec, hasAudio bool) []string {
	var filter strings.Builder
	fmt.Fprintf(&filter, "[0:v]split=%d", len(ladder))
	for i := range ladder {
		fmt.Fprintf(&filter, "[s%d]", i)
	}
	for i, r := range ladder {
		// Keep aspect ratio; -2 keeps the width even as H.264 requires.
		fmt.Fprintf(&filter, ";[s%d]scale=w=-2:h=%d:flags=bicubic,format=yuv420p[o%d]", i, r.Height, i)
	}

	args := []string{"-y", "-hide_banner", "-loglevel", "error", "-i", src, "-filter_complex", filter.String()}
	streamMap := make([]string, 0, len(ladder))
	for i, r := range ladder {
		args = append(args, "-map", fmt.Sprintf("[o%d]", i))
		args = append(args,
			fmt.Sprintf("-c:v:%d", i), "libx264",
			fmt.Sprintf("-profile:v:%d", i), "main",
			fmt.Sprintf("-level:v:%d", i), r.Level,
			fmt.Sprintf("-b:v:%d", i), r.VideoBitrate,
			fmt.Sprintf("-maxrate:v:%d", i), r.MaxBitrate,
			fmt.Sprintf("-bufsize:v:%d", i), r.BufSize,
		)
		entry := fmt.Sprintf("v:%d", i)
		if hasAudio {
			args = append(args, "-map", "0:a:0",
				fmt.Sprintf("-c:a:%d", i), "aac",
				fmt.Sprintf("-b:a:%d", i), r.AudioBitrate,
			)
			entry += fmt.Sprintf(",a:%d", i)
		}
		entry += ",name:" + r.Name
		streamMap = append(streamMap, entry)
	}
	args = append(args,
		"-preset", "veryfast",
		"-sc_threshold", "0",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", SegmentSeconds),
		"-ar", "48000", "-ac", "2",
		"-f", "hls",
		"-hls_time", strconv.Itoa(SegmentSeconds),
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_type", "mpegts",
		"-hls_segment_filename", filepath.Join(outDir, "%v", "seg_%03d.ts"),
		"-var_stream_map", strings.Join(streamMap, " "),
		filepath.Join(outDir, "%v", "playlist.m3u8"),
	)
	return args
}

var audioStreamPattern = regexp.MustCompile(`Stream #\d+:\d+.*: Audio:`)

// probeHasAudio reports whether the source has an audio stream (screen
// recordings are often silent; mapping a missing stream would fail).
func probeHasAudio(ctx context.Context, path string) bool {
	out, _ := exec.CommandContext(ctx, "ffmpeg", "-i", path, "-hide_banner").CombinedOutput()
	return audioStreamPattern.Match(out)
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// collectRendition reads the playlist bytes and walks the produced segment
// files, attaching their on-disk paths so the caller can upload each.
func collectRendition(outDir string, r HLSSpec) ([]byte, []SegmentFile, error) {
	playlist, err := os.ReadFile(filepath.Join(outDir, r.Name, "playlist.m3u8"))
	if err != nil {
		return nil, nil, err
	}
	entries, err := os.ReadDir(filepath.Join(outDir, r.Name))
	if err != nil {
		return nil, nil, err
	}
	var segs []SegmentFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ts") {
			continue
		}
		segs = append(segs, SegmentFile{
			Key:  fmt.Sprintf("hls/%s/%s", r.Name, e.Name()),
			Path: filepath.Join(outDir, r.Name, e.Name()),
		})
	}
	return playlist, segs, nil
}

// buildMasterManifest writes a #EXTM3U master that lists every rendition. The
// bandwidth hints are the VIDEO bitrate (not max); the player uses these to
// pick the right starting rendition.
func buildMasterManifest(ladder []HLSSpec, hasAudio bool) []byte {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:6\n")
	b.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	for _, r := range ladder {
		// bandwidth in bits/sec; strip trailing 'k' and multiply.
		videoBits := parseK(r.VideoBitrate)
		// Total bandwidth = video + audio (rough estimate).
		total := videoBits + parseK(r.AudioBitrate)
		codecs := avcCodec(r.Level)
		peak := parseK(r.MaxBitrate)
		if hasAudio {
			codecs += ",mp4a.40.2"
			peak += parseK(r.AudioBitrate)
		} else {
			total = videoBits
		}
		fmt.Fprintf(&b, "#EXT-X-STREAM-INF:BANDWIDTH=%d,AVERAGE-BANDWIDTH=%d,RESOLUTION=%dx%d,CODECS=\"%s\"\n",
			peak, total, r.Width, r.Height, codecs)
		fmt.Fprintf(&b, "%s/playlist.m3u8\n", r.Name)
	}
	return []byte(b.String())
}

// parseK converts "800k" → 800000, "2.5M" → 2500000. Defaults to 0 for garbage.
func parseK(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	mult := 1
	switch s[len(s)-1] {
	case 'k', 'K':
		mult = 1000
		s = s[:len(s)-1]
	case 'm', 'M':
		mult = 1000_000
		s = s[:len(s)-1]
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return int(v * float64(mult))
}

var videoSizePattern = regexp.MustCompile(`Video:.*?\b(\d{2,5})x(\d{2,5})\b`)

// probeVideoSize reads the source frame size from ffmpeg's stream info.
func probeVideoSize(ctx context.Context, path string) (int, int, error) {
	out, _ := exec.CommandContext(ctx, "ffmpeg", "-i", path, "-hide_banner").CombinedOutput()
	return parseVideoSize(string(out))
}

func parseVideoSize(ffmpegOutput string) (int, int, error) {
	m := videoSizePattern.FindStringSubmatch(ffmpegOutput)
	if m == nil {
		return 0, 0, fmt.Errorf("could not find video size in ffmpeg output")
	}
	w, _ := strconv.Atoi(m[1])
	h, err := strconv.Atoi(m[2])
	return w, h, err
}

// withSourceAspect sets each rung's advertised width from the source aspect
// ratio (FFmpeg scales with w=-2), so RESOLUTION in the master playlist
// matches the encoded frames for portrait and 4:3 sources too.
func withSourceAspect(ladder []HLSSpec, srcW, srcH int) []HLSSpec {
	if srcW <= 0 || srcH <= 0 {
		return ladder
	}
	out := make([]HLSSpec, len(ladder))
	for i, r := range ladder {
		w := int(float64(r.Height)*float64(srcW)/float64(srcH)/2+0.5) * 2
		r.Width = w
		out[i] = r
	}
	return out
}

// ladderForSourceHeight keeps the renditions that do not exceed the source
// height (allowing ~10% tolerance for odd sizes such as 1072p), and always at
// least the lowest one.
func ladderForSourceHeight(ladder []HLSSpec, sourceHeight int) []HLSSpec {
	if len(ladder) == 0 {
		return ladder
	}
	out := make([]HLSSpec, 0, len(ladder))
	for i, r := range ladder {
		if i == 0 || float64(r.Height) <= float64(sourceHeight)*1.1 {
			out = append(out, r)
		}
	}
	return out
}

// avcCodec returns the RFC 6381 codec string for H.264 Main profile at the
// given level (e.g. "3.1" -> avc1.4d401f).
func avcCodec(level string) string {
	codes := map[string]string{"3.0": "1e", "3.1": "1f", "4.0": "28", "4.1": "29"}
	if c, ok := codes[level]; ok {
		return "avc1.4d40" + c
	}
	return "avc1.4d401f"
}

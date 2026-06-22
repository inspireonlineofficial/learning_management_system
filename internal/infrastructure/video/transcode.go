package video

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// HLSSpec describes one rendition in the bitrate ladder. The defaults are
// tuned for a typical LMS lecture: 360p is the safety net for low-bandwidth
// mobile viewers, 720p is the default, 1080p rewards broadband users.
type HLSSpec struct {
	Name       string // folder name under hls/ (e.g. "v0", "v1")
	Width      int
	Height     int
	VideoBitrate string // e.g. "800k", "2500k"
	MaxBitrate   string // e.g. "900k", "2800k"
	BufSize      string // e.g. "1200k", "3500k"
	AudioBitrate string // e.g. "96k"
}

// DefaultLadder is the rendition set we transcode to. Ordered low → high so
// the master playlist is naturally sorted for ABR selection.
var DefaultLadder = []HLSSpec{
	{Name: "v0", Width: 640, Height: 360, VideoBitrate: "800k", MaxBitrate: "900k", BufSize: "1200k", AudioBitrate: "96k"},
	{Name: "v1", Width: 1280, Height: 720, VideoBitrate: "2500k", MaxBitrate: "2800k", BufSize: "3500k", AudioBitrate: "128k"},
	{Name: "v2", Width: 1920, Height: 1080, VideoBitrate: "5000k", MaxBitrate: "5500k", BufSize: "7000k", AudioBitrate: "192k"},
}

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

	renditions := make(map[string]Rendition, len(ladder))
	for _, r := range ladder {
		if err := encodeRendition(ctx, sourcePath, outDir, r); err != nil {
			cleanup()
			return nil, fmt.Errorf("encode %s: %w", r.Name, err)
		}
		playlist, segs, err := collectRendition(outDir, r)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("collect %s: %w", r.Name, err)
		}
		renditions[r.Name] = Rendition{Playlist: playlist, Segments: segs}
	}

	master := buildMasterManifest(ladder)
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
func encodeRendition(ctx context.Context, src, outDir string, r HLSSpec) error {
	playlistPath := filepath.Join(outDir, r.Name, "playlist.m3u8")
	segmentPattern := filepath.Join(outDir, r.Name, "seg_%03d.ts")
	if err := os.MkdirAll(filepath.Join(outDir, r.Name), 0o755); err != nil {
		return err
	}
	// We need a .m3u8 to be written alongside the segments; ffmpeg handles
	// the playlist when given -hls_segment_filename.
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", src,
		"-vf", fmt.Sprintf("scale=w=%d:h=%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", r.Width, r.Height, r.Width, r.Height),
		"-c:a", "aac",
		"-ar", "48000",
		"-c:v", "h264",
		"-profile:v", "main",
		"-crf", "20",
		"-sc_threshold", "0",
		"-g", "48",
		"-keyint_min", "48",
		"-hls_time", "4",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", segmentPattern,
		"-b:v", r.VideoBitrate,
		"-maxrate", r.MaxBitrate,
		"-bufsize", r.BufSize,
		"-b:a", r.AudioBitrate,
		playlistPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w\n%s", err, string(out))
	}
	return nil
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
func buildMasterManifest(ladder []HLSSpec) []byte {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:6\n")
	for _, r := range ladder {
		// bandwidth in bits/sec; strip trailing 'k' and multiply.
		videoBits := parseK(r.VideoBitrate)
		// Total bandwidth = video + audio (rough estimate).
		total := videoBits + parseK(r.AudioBitrate)
		fmt.Fprintf(&b, "#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,CODECS=\"avc1.4d401f,mp4a.40.2\"\n",
			total, r.Width, r.Height)
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
package video

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLadderForSourceHeight(t *testing.T) {
	cases := map[int][]string{
		240:  {"v0"},
		360:  {"v0", "v1"},
		480:  {"v0", "v1", "v2"},
		720:  {"v0", "v1", "v2", "v3"},
		1080: {"v0", "v1", "v2", "v3", "v4"},
		2160: {"v0", "v1", "v2", "v3", "v4"},
		100:  {"v0"},
	}
	for height, want := range cases {
		got := ladderForSourceHeight(DefaultLadder, height)
		names := make([]string, len(got))
		for i, r := range got {
			names[i] = r.Name
		}
		if strings.Join(names, ",") != strings.Join(want, ",") {
			t.Errorf("source %dp: got %v, want %v", height, names, want)
		}
	}
}

func TestParseVideoSizeAndAspect(t *testing.T) {
	w, h, err := parseVideoSize("Stream #0:0[0x1](und): Video: h264 (High) (avc1 / 0x31637661), yuv420p(progressive), 1280x720 [SAR 1:1 DAR 16:9], 2000 kb/s, 30 fps")
	if err != nil || w != 1280 || h != 720 {
		t.Fatalf("got %dx%d err=%v", w, h, err)
	}
	portrait := withSourceAspect(DefaultLadder[:2], 720, 1280)
	if portrait[0].Width != 136 || portrait[1].Width != 202 {
		t.Fatalf("portrait widths not derived from aspect: %+v", portrait)
	}
}

func TestMasterManifest(t *testing.T) {
	m := string(buildMasterManifest(DefaultLadder[:2], true))
	for _, want := range []string{"#EXT-X-INDEPENDENT-SEGMENTS", `CODECS="avc1.4d401e,mp4a.40.2"`, "v0/playlist.m3u8", "v1/playlist.m3u8"} {
		if !strings.Contains(m, want) {
			t.Errorf("master missing %q:\n%s", want, m)
		}
	}
	if strings.Contains(string(buildMasterManifest(DefaultLadder[:1], false)), "mp4a") {
		t.Errorf("silent sources must not advertise an audio codec")
	}
}

func TestLadderArgsAlignKeyframes(t *testing.T) {
	args := strings.Join(buildLadderArgs("in.mp4", "/out", DefaultLadder[:3], true), " ")
	for _, want := range []string{"split=3", "-force_key_frames expr:gte(t,n_forced*4)", "-sc_threshold 0", "-hls_time 4", "v:0,a:0,name:v0 v:1,a:1,name:v1 v:2,a:2,name:v2", "-hls_flags independent_segments"} {
		if !strings.Contains(args, want) {
			t.Errorf("ffmpeg args missing %q", want)
		}
	}
}

// Integration: runs the real encoder when ffmpeg is installed.
func TestTranscodeToHLSIntegration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src.mp4")
	gen := exec.Command("ffmpeg", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc=size=854x480:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440", "-t", "9", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generate source: %v %s", err, out)
	}
	res, err := TranscodeToHLS(context.Background(), src, DefaultLadder)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}
	defer os.RemoveAll(res.OutputDir)
	if len(res.Renditions) != 3 {
		t.Fatalf("480p source should yield 3 renditions, got %d", len(res.Renditions))
	}
	for name, r := range res.Renditions {
		if len(r.Segments) < 3 || !strings.Contains(string(r.Playlist), "#EXT-X-ENDLIST") {
			t.Errorf("%s: %d segments, playlist:\n%s", name, len(r.Segments), r.Playlist)
		}
		if strings.Contains(string(r.Playlist), "/") {
			t.Errorf("%s: playlist should reference segments relatively:\n%s", name, r.Playlist)
		}
	}
	if res.DurationSeconds < 8 {
		t.Errorf("duration not probed: %d", res.DurationSeconds)
	}
}

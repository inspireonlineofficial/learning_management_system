package video

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// buildAtom produces a single top-level MP4 atom. Used to compose synthetic
// ftyp/moov/mdat layouts in tests without depending on a real MP4 file.
func buildAtom(typ string, body []byte) []byte {
	out := make([]byte, 8+len(body))
	binary.BigEndian.PutUint32(out[0:4], uint32(8+len(body)))
	copy(out[4:8], typ)
	copy(out[8:], body)
	return out
}

func TestIsFastStartMP4_FastStart(t *testing.T) {
	// ftyp followed by moov followed by mdat — faststart.
	buf := buildAtom("ftyp", []byte("isom"))
	buf = append(buf, buildAtom("moov", []byte("body"))...)
	buf = append(buf, buildAtom("mdat", []byte("data"))...)
	ok, err := IsFastStartMP4(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected faststart = true for moov-before-mdat layout")
	}
}

func TestIsFastStartMP4_NotFastStart(t *testing.T) {
	// ftyp followed by mdat followed by moov — not faststart.
	buf := buildAtom("ftyp", []byte("isom"))
	buf = append(buf, buildAtom("mdat", []byte("data"))...)
	buf = append(buf, buildAtom("moov", []byte("body"))...)
	ok, err := IsFastStartMP4(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected faststart = false for mdat-before-moov layout")
	}
}

func TestIsFastStartMP4_RejectsNonMP4(t *testing.T) {
	// Anything without an ftyp atom should be rejected.
	buf := buildAtom("RIFF", []byte("data"))
	_, err := IsFastStartMP4(bytes.NewReader(buf))
	if err == nil {
		t.Fatal("expected error for non-MP4 file")
	}
}
func TestParseK(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"800k", 800_000},
		{"2.5M", 2_500_000},
		{"100", 100},
		{"", 0},
		{"garbage", 0},
	}
	for _, c := range cases {
		got := parseK(c.in)
		if got != c.want {
			t.Errorf("parseK(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestBuildMasterManifest(t *testing.T) {
	ladder := []HLSSpec{
		{Name: "v0", Width: 640, Height: 360, VideoBitrate: "800k", AudioBitrate: "96k"},
		{Name: "v1", Width: 1280, Height: 720, VideoBitrate: "2500k", AudioBitrate: "128k"},
	}
	out := buildMasterManifest(ladder)
	s := string(out)
	if !strings.HasPrefix(s, "#EXTM3U") {
		t.Errorf("missing master playlist header: %q", s)
	}
	if !strings.Contains(s, "v0/playlist.m3u8") {
		t.Errorf("expected v0 rendition in master: %q", s)
	}
	if !strings.Contains(s, "v1/playlist.m3u8") {
		t.Errorf("expected v1 rendition in master: %q", s)
	}
	if !strings.Contains(s, "RESOLUTION=1280x720") {
		t.Errorf("expected resolution tag: %q", s)
	}
	// Bandwidth is video+audio in bits/sec; v1 should be the larger value.
	v0idx := strings.Index(s, "v0/playlist")
	v1idx := strings.Index(s, "v1/playlist")
	if v0idx < 0 || v1idx < 0 || v1idx < v0idx {
		t.Fatalf("renditions out of order: %q", s)
	}
}

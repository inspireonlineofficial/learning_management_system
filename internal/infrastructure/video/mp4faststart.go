// Package video contains helpers for working with raw video bytes.
//
// The single most common cause of "video won't seek" in the LMS is an MP4
// file with the moov atom at the end of the file. Browsers can't compute
// the seek table without the moov, so they download the entire file
// before allowing scrub. iPhone recordings and many screen recorders
// produce this layout by default; re-muxing with ffmpeg -movflags
// +faststart moves the moov to the front.
//
// This package ships a tiny pure-Go detector so the upload service can
// reject non-faststart MP4s with a clear, actionable error rather than
// silently producing a broken player. We also expose a faststart
// re-muxer that runs only when the ffmpeg binary is available on the
// host. The re-muxer is intentionally a separate path so the upload
// flow stays fast in the common case (already-faststart MP4s) and
// gracefully degrades on hosts that don't ship ffmpeg.
package video

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// MP4 atom identifiers. Atoms are length-prefixed big-endian chunks
// inside an MP4 file; the first 4 bytes of the box are the size, the
// next 4 are the type (e.g. "moov", "mdat"). See ISO/IEC 14496-12.
const (
	atomFtyp = "ftyp"
	atomMoov = "moov"
	atomMdat = "mdat"
)

// IsFastStartMP4 returns true if the moov atom appears before the mdat
// atom in the file. This is a strong heuristic for "browser can seek
// without downloading the whole file first." The function only reads
// the top-level atoms (it does not descend into moov) so it is O(N) in
// the number of top-level atoms and finishes in a few KB of reads
// regardless of file size.
func IsFastStartMP4(reader io.Reader) (bool, error) {
	// We need random access into the reader, so wrap it in a SectionReader
	// anchored to the start of the stream.
	buf, err := io.ReadAll(io.LimitReader(reader, 8*1024*1024))
	if err != nil {
		return false, fmt.Errorf("read header: %w", err)
	}
	if len(buf) < 16 {
		return false, errors.New("file too small to be MP4")
	}
	// Top-level ftyp must be present.
	if !bytes.Equal(buf[4:8], []byte(atomFtyp)) {
		return false, errors.New("not an MP4 (missing ftyp)")
	}
	// Walk atoms.
	pos := 0
	sawMoov := false
	sawMdat := false
	for pos+8 <= len(buf) {
		size := int(binary.BigEndian.Uint32(buf[pos : pos+4]))
		typ := string(buf[pos+4 : pos+8])
		if size == 0 {
			// size==0 means "atom extends to end of file." We can't know
			// the order without reading the full file; bail.
			return sawMoov && !sawMdat, nil
		}
		if size < 8 {
			return false, fmt.Errorf("invalid atom size %d at offset %d", size, pos)
		}
		switch typ {
		case atomMoov:
			sawMoov = true
		case atomMdat:
			sawMdat = true
		}
		if sawMoov && !sawMdat {
			return true, nil
		}
		if sawMdat && !sawMoov {
			return false, nil
		}
		// Move to the next atom. We do not have the full file in memory, so
		// bail once we run out of buffer; for our purposes (detecting
		// faststart) the top of the file is what matters.
		next := pos + size
		if next > len(buf) {
			// We've consumed the whole header snapshot. If we haven't seen
			// both atoms, fall back to the heuristic that moov-before-mdat
			// is faststart: if we saw moov already without mdat, assume
			// faststart; if we saw mdat without moov, assume not.
			return sawMoov && !sawMdat, nil
		}
		pos = next
	}
	return sawMoov && !sawMdat, nil
}

// FastStartIfPossible returns a reader over the same bytes that has the
// moov atom moved to the front. If ffmpeg is not available it returns
// the input reader unchanged and reports ok=false. We use an external
// process for the re-mux because a pure-Go MP4 re-muxer is a few
// thousand lines of careful code and we want to keep the dependency
// surface small. The cost is one extra process per non-faststart MP4,
// which is acceptable for an LMS upload (cold path).
func FastStartIfPossible(ctx context.Context, input io.Reader) (io.Reader, bool, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return input, false, nil
	}
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", "pipe:0",
		"-c", "copy",
		"-movflags", "+faststart",
		"-f", "mp4",
		"pipe:1",
	)
	cmd.Stdin = input
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, fmt.Errorf("ffmpeg stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, false, fmt.Errorf("ffmpeg start: %w", err)
	}
	// We return the pipe; the caller is responsible for waiting on the
	// process. To make that easy we wrap so that Close() waits.
	return &cmdReadCloser{ReadCloser: out, cmd: cmd}, true, nil
}

type cmdReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (c *cmdReadCloser) Close() error {
	readErr := c.ReadCloser.Close()
	waitErr := c.cmd.Wait()
	if readErr != nil {
		return readErr
	}
	return waitErr
}

// Thumbnail returns a JPEG poster frame captured from the first viable
// timestamp of the input. The size argument is the long edge in pixels;
// the aspect ratio of the input is preserved. Returns ok=false (and no
// error) when ffmpeg is not available; the caller should treat that as
// "no thumbnail" rather than a hard failure.
func Thumbnail(ctx context.Context, input io.Reader, size int) (io.Reader, bool, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return input, false, nil
	}
	// -ss before -i is "fast seek": decodes only the keyframe we want. This
	// is what we want for poster generation; otherwise ffmpeg decodes the
	// whole file looking for second 1.
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", "1",
		"-i", "pipe:0",
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:-2", size),
		"-f", "image2",
		"-vcodec", "mjpeg",
		"pipe:1",
	)
	cmd.Stdin = input
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, fmt.Errorf("ffmpeg stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, false, fmt.Errorf("ffmpeg start: %w", err)
	}
	return &cmdReadCloser{ReadCloser: out, cmd: cmd}, true, nil
}
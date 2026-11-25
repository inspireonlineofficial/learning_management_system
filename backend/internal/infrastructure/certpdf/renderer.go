// Package certpdf renders course-completion certificates as a single-page PDF
// without third-party dependencies (PDF 1.4, standard Type1 fonts).
package certpdf

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode"

	"lms-backend/internal/domain/certificates"
)

// Renderer implements certificates.PDFRenderer.
type Renderer struct {
	// VerifyBaseURL is the public page used to verify a certificate, e.g.
	// "https://example.com/verify". Optional.
	VerifyBaseURL string
}

// NewRenderer creates a certificate renderer.
func NewRenderer(verifyBaseURL string) *Renderer {
	return &Renderer{VerifyBaseURL: strings.TrimRight(verifyBaseURL, "/")}
}

type textLine struct {
	font string // F1 = Helvetica, F2 = Helvetica-Bold, F3 = Times-Italic
	size float64
	y    float64
	text string
}

// RenderCertificate returns the PDF bytes for a certificate.
func (r *Renderer) RenderCertificate(ctx context.Context, cert *certificates.Certificate) ([]byte, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate is required")
	}
	const width, height = 842.0, 595.0 // A4 landscape (points)

	lines := []textLine{
		{"F2", 34, 470, "Certificate of Completion"},
		{"F1", 14, 425, "This certifies that"},
		{"F3", 30, 380, cert.StudentName},
		{"F1", 14, 340, "has successfully completed the course"},
		{"F2", 22, 300, cert.CourseTitle},
		{"F1", 13, 250, "Instructor: " + cert.InstructorName},
		{"F1", 13, 228, "Completed on " + cert.CompletionDate.Format("January 2, 2006")},
		{"F1", 9, 110, "Verification ID: " + cert.VerificationID},
	}
	if r.VerifyBaseURL != "" {
		lines = append(lines, textLine{"F1", 9, 95, "Verify at " + r.VerifyBaseURL + "/" + cert.VerificationID})
	}

	var content bytes.Buffer
	// Border
	content.WriteString("0.15 0.2 0.35 RG 3 w 30 30 782 535 re S 1 w 40 40 762 515 re S\n")
	for _, l := range lines {
		text := pdfSafe(l.text)
		x := (width - approxWidth(text, l.size, l.font)) / 2
		if x < 50 {
			x = 50
		}
		fmt.Fprintf(&content, "BT /%s %.1f Tf 0.1 0.12 0.2 rg %.2f %.2f Td (%s) Tj ET\n", l.font, l.size, x, l.y, text)
	}

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.0f %.0f] /Resources << /Font << /F1 5 0 R /F2 6 0 R /F3 7 0 R >> >> /Contents 4 0 R >>", width, height),
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", content.Len(), content.String()),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Times-Italic /Encoding /WinAnsiEncoding >>",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes(), nil
}

// pdfSafe escapes PDF string delimiters and replaces characters the standard
// fonts cannot show (WinAnsi covers Latin-1); other scripts become "?".
func pdfSafe(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '(' || r == ')' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 32 || unicode.IsControl(r):
			b.WriteByte(' ')
		case r < 256:
			b.WriteByte(byte(r))
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}

// approxWidth estimates rendered text width for centring (Helvetica average).
func approxWidth(s string, size float64, font string) float64 {
	factor := 0.5
	if font == "F2" {
		factor = 0.56
	}
	return float64(len(s)) * size * factor
}

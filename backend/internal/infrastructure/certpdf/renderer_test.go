package certpdf

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"lms-backend/internal/domain/certificates"
)

func TestRenderCertificateProducesPDF(t *testing.T) {
	pdf, err := NewRenderer("https://lms.example/verify").RenderCertificate(context.Background(), &certificates.Certificate{
		StudentName: "Ada (Countess) Lovelace", CourseTitle: "Go \\ Concurrency", InstructorName: "Rob",
		CompletionDate: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), VerificationID: "abc123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) || !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Fatalf("not a PDF")
	}
	s := string(pdf)
	for _, want := range []string{`Ada \(Countess\) Lovelace`, `Go \\ Concurrency`, "abc123", "https://lms.example/verify/abc123"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
}

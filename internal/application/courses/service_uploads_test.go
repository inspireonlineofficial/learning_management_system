package courses

import "testing"

func TestValidateUploadAcceptsBrowserDeclaredVideoWhenMagicIsGeneric(t *testing.T) {
	contentType, err := validateUpload(
		1024,
		maxVideoUploadBytes,
		"video/mp4",
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05},
		map[string]bool{
			"video/mp4":       true,
			"video/webm":      true,
			"video/quicktime": true,
		},
		true,
	)
	if err != nil {
		t.Fatalf("expected declared video fallback to pass, got %v", err)
	}
	if contentType != "video/mp4" {
		t.Fatalf("expected video/mp4, got %q", contentType)
	}
}

func TestValidateUploadRejectsDeclaredFallbackForNonVideoUploads(t *testing.T) {
	_, err := validateUpload(
		1024,
		maxFileUploadBytes,
		"image/png",
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05},
		map[string]bool{
			"application/pdf": true,
			"image/png":       true,
		},
		false,
	)
	if err == nil {
		t.Fatal("expected generic file magic to be rejected without declared fallback")
	}
}

func TestValidateUploadRejectsExecutablesBeforeDeclaredFallback(t *testing.T) {
	_, err := validateUpload(
		1024,
		maxVideoUploadBytes,
		"video/mp4",
		[]byte{'M', 'Z', 0x90, 0x00},
		map[string]bool{
			"video/mp4": true,
		},
		true,
	)
	if err == nil {
		t.Fatal("expected executable magic to be rejected")
	}
}

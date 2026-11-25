package slides

import "testing"

func TestValidateSlideMedia(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		declared string
		magic    []byte
		wantType string
		wantErr  bool
	}{
		{name: "accepts jpeg", size: 4, declared: "image/jpeg", magic: []byte{0xff, 0xd8, 0xff, 0xdb}, wantType: "image/jpeg"},
		{name: "accepts png", size: 8, declared: "image/png", magic: []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, wantType: "image/png"},
		{name: "rejects oversized file", size: maxSlideMediaBytes + 1, declared: "image/png", magic: []byte{0x89, 0x50, 0x4e, 0x47}, wantErr: true},
		{name: "rejects unsupported type", size: 4, declared: "application/pdf", magic: []byte("%PDF"), wantErr: true},
		{name: "rejects declared mismatch", size: 8, declared: "image/jpeg", magic: []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSlideMedia(tt.size, tt.declared, tt.magic)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSlideMedia() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.wantType {
				t.Fatalf("validateSlideMedia() = %q, want %q", got, tt.wantType)
			}
		})
	}
}

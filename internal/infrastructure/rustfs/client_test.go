package rustfs

import "testing"

func TestNormalizeEndpointStripsDigitalOceanBucketHost(t *testing.T) {
	got := normalizeEndpoint("https://inspirelms.sgp1.digitaloceanspaces.com")
	want := "https://sgp1.digitaloceanspaces.com"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeEndpointLeavesDigitalOceanRegionHost(t *testing.T) {
	got := normalizeEndpoint("https://sgp1.digitaloceanspaces.com")
	want := "https://sgp1.digitaloceanspaces.com"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeEndpointLeavesCustomEndpoint(t *testing.T) {
	got := normalizeEndpoint("https://storage.example.com")
	want := "https://storage.example.com"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

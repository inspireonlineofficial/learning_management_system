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

func TestResolveEndpointConfigUsesDigitalOceanSigningDefaults(t *testing.T) {
	got := resolveEndpointConfig("https://inspirelms.sgp1.digitaloceanspaces.com", "sgp1")
	if got.endpoint != "https://sgp1.digitaloceanspaces.com" {
		t.Fatalf("expected normalized endpoint, got %q", got.endpoint)
	}
	if got.region != "us-east-1" {
		t.Fatalf("expected us-east-1 signing region, got %q", got.region)
	}
	if got.forcePathStyle {
		t.Fatal("expected virtual-host style for DigitalOcean Spaces")
	}
}

func TestResolveEndpointConfigKeepsPathStyleForCustomEndpoint(t *testing.T) {
	got := resolveEndpointConfig("https://storage.example.com", "us-west-2")
	if got.endpoint != "https://storage.example.com" {
		t.Fatalf("expected custom endpoint to remain unchanged, got %q", got.endpoint)
	}
	if got.region != "us-west-2" {
		t.Fatalf("expected custom region to remain unchanged, got %q", got.region)
	}
	if !got.forcePathStyle {
		t.Fatal("expected path style for generic S3-compatible endpoints")
	}
}

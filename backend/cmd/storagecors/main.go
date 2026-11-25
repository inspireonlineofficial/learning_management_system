// Command storagecors shows and sets the CORS rule that lets the LMS
// frontend read private objects through signed URLs: the PDF.js book reader
// (range requests on the books bucket) and the HLS player (video bucket).
//
// It only adds or updates one rule with ID "lms-frontend"; other rules on
// the bucket are kept. Without -apply it prints what it would do.
//
//	go run ./cmd/storagecors -origins https://app.example.com
//	go run ./cmd/storagecors -origins https://app.example.com -apply
//
// Credentials and endpoint come from the same RUSTFS_* variables as the server.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"lms-backend/internal/infrastructure/rustfs"
)

func main() {
	origins := flag.String("origins", os.Getenv("FRONTEND_BASE_URL"), "comma-separated frontend origins, e.g. https://app.example.com")
	buckets := flag.String("buckets", envOr("RUSTFS_BOOKS_BUCKET", "lms-books")+","+envOr("RUSTFS_VIDEO_BUCKET", "lms-videos"), "comma-separated buckets")
	apply := flag.Bool("apply", false, "write the rule (default: dry run)")
	flag.Parse()

	allowed, err := rustfs.ValidateCORSOrigins(splitList(*origins))
	if err != nil {
		fail("%v", err)
	}
	client, err := rustfs.NewClient(os.Getenv("RUSTFS_ENDPOINT"), os.Getenv("RUSTFS_ACCESS_KEY"), os.Getenv("RUSTFS_SECRET_KEY"), envOr("RUSTFS_REGION", "us-east-1"))
	if err != nil {
		fail("storage client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, bucket := range splitList(*buckets) {
		current, err := client.GetBucketCORS(ctx, bucket)
		if err != nil {
			fail("%s: read CORS: %v", bucket, err)
		}
		fmt.Printf("%s: %d existing rule(s)\n", bucket, len(current))
		if !*apply {
			fmt.Printf("%s: would set rule %q for %s\n", bucket, rustfs.FrontendCORSRuleID, strings.Join(allowed, ", "))
			continue
		}
		changed, err := client.EnsureFrontendCORS(ctx, bucket, allowed)
		if err != nil {
			fail("%v", err)
		}
		after, _ := client.GetBucketCORS(ctx, bucket)
		fmt.Printf("%s: %s; now %d rule(s)\n", bucket, map[bool]string{true: "rule written", false: "rule already up to date"}[changed], len(after))
	}
	if !*apply {
		fmt.Println("dry run: add -apply to write")
	}
}

func splitList(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimRight(strings.TrimSpace(p), "/"); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "storagecors: "+format+"\n", args...)
	os.Exit(1)
}

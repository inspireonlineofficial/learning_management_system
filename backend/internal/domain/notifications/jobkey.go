package notifications

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type jobKeyCtx struct{}

// WithJobKey marks ctx as running the job with this ID.
func WithJobKey(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, jobKeyCtx{}, jobID)
}

// JobKey returns the ID of the job running in ctx, or "".
func JobKey(ctx context.Context) string {
	v, _ := ctx.Value(jobKeyCtx{}).(string)
	return v
}

// dedupeNamespace scopes the deterministic IDs below.
var dedupeNamespace = uuid.MustParse("0d1f6a52-5c3e-4c0b-9a51-2f7e9b4c8d10")

// DedupeID returns an ID that is the same every time the job running in ctx
// is retried, for the given parts (for example a recipient's user ID). Rows
// inserted with it and ON CONFLICT DO NOTHING are written once however many
// times the job runs. Outside a job it returns a fresh random ID.
func DedupeID(ctx context.Context, parts ...string) uuid.UUID {
	key := JobKey(ctx)
	if key == "" {
		return uuid.New()
	}
	return uuid.NewSHA1(dedupeNamespace, []byte(key+"|"+strings.Join(parts, "|")))
}

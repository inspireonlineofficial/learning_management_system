package assessments

import (
	"github.com/google/uuid"
)

// SubmissionFile represents a file attached to an assignment submission
type SubmissionFile struct {
	ID               uuid.UUID
	SubmissionID     uuid.UUID
	RustFSKey        string // NEVER exposed in API - use presigned URLs
	OriginalFilename string
	MimeType         string
	SizeBytes        int64
}

// IsWithinSizeLimit checks if the file is within the specified size limit
func (sf *SubmissionFile) IsWithinSizeLimit(maxSizeMB int) bool {
	maxBytes := int64(maxSizeMB) * 1024 * 1024
	return sf.SizeBytes <= maxBytes
}

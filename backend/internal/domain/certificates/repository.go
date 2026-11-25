package certificates

import (
	"context"

	"github.com/google/uuid"
)

// CertificateRepository defines the persistence port for the certificates context.
// Requirements: 18.1, 18.2
type CertificateRepository interface {
	// Create inserts a new certificate record.
	Create(ctx context.Context, cert *Certificate) error

	// FindByStudentAndCourse returns the certificate for a given student+course pair,
	// or nil if no certificate has been issued yet.
	FindByStudentAndCourse(ctx context.Context, studentID, courseID uuid.UUID) (*Certificate, error)

	// FindByVerificationID returns the certificate with the given verification_id,
	// used by the public verification endpoint.
	FindByVerificationID(ctx context.Context, verificationID string) (*Certificate, error)

	// UpdatePDFKey sets the pdf_rustfs_key after async PDF generation completes.
	UpdatePDFKey(ctx context.Context, id uuid.UUID, pdfKey string) error
}

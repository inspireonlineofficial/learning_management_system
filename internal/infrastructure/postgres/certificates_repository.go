package postgres

import (
	"context"
	"database/sql"

	domaincerts "lms-backend/internal/domain/certificates"

	"github.com/google/uuid"
)

// CertificateRepository implements domain/certificates.CertificateRepository.
type CertificateRepository struct {
	db *sql.DB
}

// NewCertificateRepository creates a new CertificateRepository.
func NewCertificateRepository(db *sql.DB) *CertificateRepository {
	return &CertificateRepository{db: db}
}

// Create inserts a new certificate record.
func (r *CertificateRepository) Create(ctx context.Context, cert *domaincerts.Certificate) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO certificates
			(id, student_id, course_id, verification_id, student_name, course_title, instructor_name, completion_date, pdf_rustfs_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		cert.ID,
		cert.StudentID,
		cert.CourseID,
		cert.VerificationID,
		cert.StudentName,
		cert.CourseTitle,
		cert.InstructorName,
		cert.CompletionDate,
		cert.PDFRustFSKey,
		cert.CreatedAt,
	)
	return err
}

// FindByStudentAndCourse returns the certificate for a given student+course pair.
func (r *CertificateRepository) FindByStudentAndCourse(ctx context.Context, studentID, courseID uuid.UUID) (*domaincerts.Certificate, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, course_id, verification_id, student_name, course_title, instructor_name, completion_date, pdf_rustfs_key, created_at
		FROM certificates
		WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID,
	)
	return scanCertificate(row)
}

// FindByVerificationID returns the certificate with the given verification_id.
func (r *CertificateRepository) FindByVerificationID(ctx context.Context, verificationID string) (*domaincerts.Certificate, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, course_id, verification_id, student_name, course_title, instructor_name, completion_date, pdf_rustfs_key, created_at
		FROM certificates
		WHERE verification_id = $1`,
		verificationID,
	)
	return scanCertificate(row)
}

// UpdatePDFKey sets the pdf_rustfs_key after async PDF generation completes.
func (r *CertificateRepository) UpdatePDFKey(ctx context.Context, id uuid.UUID, pdfKey string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE certificates SET pdf_rustfs_key = $1 WHERE id = $2`,
		pdfKey, id,
	)
	return err
}

// scanCertificate scans a single row into a Certificate.
func scanCertificate(row *sql.Row) (*domaincerts.Certificate, error) {
	cert := &domaincerts.Certificate{}
	err := row.Scan(
		&cert.ID,
		&cert.StudentID,
		&cert.CourseID,
		&cert.VerificationID,
		&cert.StudentName,
		&cert.CourseTitle,
		&cert.InstructorName,
		&cert.CompletionDate,
		&cert.PDFRustFSKey,
		&cert.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return cert, err
}

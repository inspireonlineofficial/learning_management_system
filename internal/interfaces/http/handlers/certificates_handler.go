package handlers

import (
	"net/http"

	"lms-backend/internal/application/certificates"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// CertificatesHandler handles HTTP requests for the certificates bounded context.
type CertificatesHandler struct {
	service certificates.Service
}

// NewCertificatesHandler creates a new CertificatesHandler.
func NewCertificatesHandler(service certificates.Service) *CertificatesHandler {
	return &CertificatesHandler{service: service}
}

// GetStudentCertificate handles GET /v1/student/certificates/:courseId
// Returns the student's certificate for a course, including a presigned PDF URL.
// Returns 404 CERTIFICATE_NOT_EARNED if the course is not yet complete.
// Requirements: 18.3, 18.5
//
// @Summary      Get student certificate
// @Description  Returns the authenticated student's certificate for a completed course, including a presigned PDF URL
// @Tags         certificates
// @Produce      json
// @Param        courseId  path  string  true  "Course ID"
// @Success      200  {object}  certificates.CertificateResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/student/certificates/{courseId} [get]
func (h *CertificatesHandler) GetStudentCertificate(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	cmd := certificates.GetStudentCertificateCommand{
		StudentID: userID,
		CourseID:  courseID,
	}

	result, err := h.service.GetStudentCertificate(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// VerifyCertificate handles GET /v1/certificates/verify/:verificationId (public)
// Returns validity and public fields for a given verification_id.
// Requirements: 18.4, 18.6
//
// @Summary      Verify certificate
// @Description  Public endpoint that verifies a certificate by its verification ID and returns its validity and public details
// @Tags         certificates
// @Produce      json
// @Param        verificationId  path  string  true  "Verification ID"
// @Success      200  {object}  certificates.VerifyCertificateResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /v1/certificates/verify/{verificationId} [get]
func (h *CertificatesHandler) VerifyCertificate(w http.ResponseWriter, r *http.Request) {
	verificationID := r.PathValue("verificationId")
	if verificationID == "" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "verification ID is required"))
		return
	}

	cmd := certificates.VerifyCertificateCommand{
		VerificationID: verificationID,
	}

	result, err := h.service.VerifyCertificate(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

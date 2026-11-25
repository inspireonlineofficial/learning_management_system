import { apiRequest } from "./client";
import { listMyEnrollments } from "./student";

export type Certificate = {
  id?: string;
  course_id: string;
  course_title: string;
  issued_at: string;
  certificate_url?: string;
  verification_code?: string;
};

export type CertificateVerification = {
  valid: boolean;
  verification_id: string;
  student_name?: string;
  course_title?: string;
  instructor_name?: string;
  completion_date?: string;
  issued_at?: string;
};

export const listCertificates = () =>
  listMyEnrollments({ status: "completed" }).then(async (enrollments) => {
    const items = await Promise.all(
      enrollments.data
        // Skip enrollments whose course was soft-deleted — those rows have
        // no certificate to fetch and the ID is gone from the public catalog.
        .filter((enrollment) => enrollment.course != null)
        .map((enrollment) => getCertificate(enrollment.course!.id).catch(() => null)),
    );
    return { items: items.filter((item): item is Certificate => Boolean(item)) };
  });
type CertificateResponse = Certificate & {
  pdf_url?: string | null;
  verification_id?: string;
  created_at?: string;
  completion_date?: string;
  student_name?: string;
  instructor_name?: string;
};

// The API returns pdf_url / verification_id / created_at; map them to the
// fields the UI renders (these were previously always empty, and "Download
// PDF" opened the JSON endpoint instead of a file).
export const getCertificate = (courseId: string): Promise<Certificate> =>
  apiRequest<CertificateResponse>(`/v1/student/certificates/${courseId}`, { auth: true }).then(
    (certificate): Certificate => ({
      ...certificate,
      course_id: certificate.course_id ?? courseId,
      issued_at:
        certificate.issued_at ?? certificate.completion_date ?? certificate.created_at ?? "",
      certificate_url: certificate.certificate_url ?? certificate.pdf_url ?? undefined,
      verification_code: certificate.verification_code ?? certificate.verification_id,
    }),
  );
export const downloadCertificatePdf = (courseId: string) =>
  getCertificate(courseId).then((certificate) => {
    if (!certificate.certificate_url) {
      throw new Error("The certificate PDF is being prepared. Please try again in a moment.");
    }
    return { url: certificate.certificate_url };
  });
export const verifyCertificate = (verificationId: string) =>
  apiRequest<CertificateVerification>(
    `/v1/certificates/verify/${encodeURIComponent(verificationId)}`,
  );

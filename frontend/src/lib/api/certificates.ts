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
export const getCertificate = (courseId: string) =>
  apiRequest<Certificate>(`/v1/student/certificates/${courseId}`, { auth: true }).then(
    (certificate) => ({
      ...certificate,
      course_id: certificate.course_id ?? courseId,
    }),
  );
export const downloadCertificatePdf = (courseId: string) =>
  getCertificate(courseId).then((certificate) => ({
    url: certificate.certificate_url ?? `/v1/student/certificates/${courseId}`,
  }));
export const verifyCertificate = (verificationId: string) =>
  apiRequest<CertificateVerification>(
    `/v1/certificates/verify/${encodeURIComponent(verificationId)}`,
  );

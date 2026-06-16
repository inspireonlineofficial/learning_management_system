import { apiRequest } from "./client";

export type Submission = {
  id: string;
  assignment_id: string;
  student: { id: string; full_name: string };
  submitted_at: string;
  status: "submitted" | "graded" | "returned";
  score?: number;
  files?: Array<{ name: string; url: string }>;
  text?: string;
};

type BackendSubmission = {
  id: string;
  assignment_id: string;
  student_id?: string;
  student?: { id: string; full_name: string };
  submitted_at?: string | null;
  created_at?: string;
  status: "submitted" | "graded" | "returned" | string;
  score?: number;
  text?: string;
  text_content?: string;
  files?: Array<{ name?: string; url?: string; original_filename?: string; download_url?: string }>;
  latest_grade?: { score?: number; feedback?: string };
};

export const listSubmissions = (assignmentId: string) =>
  apiRequest<{ items?: BackendSubmission[]; submissions?: BackendSubmission[] }>(
    `/v1/teacher/assignments/${assignmentId}/submissions`,
    {
      auth: true,
    },
  ).then((result) => ({
    items: (result.items ?? result.submissions ?? []).map(toSubmission),
  }));
export const getSubmission = (assignmentId: string, submissionId: string) =>
  apiRequest<BackendSubmission>(
    `/v1/teacher/assignments/${assignmentId}/submissions/${submissionId}`,
    {
      auth: true,
    },
  ).then(toSubmission);
export const gradeSubmission = (
  assignmentId: string,
  submissionId: string,
  input: { score: number; feedback?: string },
) =>
  apiRequest<{ score?: number; feedback?: string; graded_at?: string }>(
    `/v1/teacher/assignments/${assignmentId}/submissions/${submissionId}/grade`,
    { method: "POST", auth: true, body: input },
  ).then(() => ({ ok: true }));
export const submitAssignment = (
  assignmentId: string,
  input: { text?: string; file_urls?: string[] },
) =>
  apiRequest<Submission>(`/v1/assignments/${assignmentId}/submissions`, {
    method: "POST",
    auth: true,
    body: input,
  });

function toSubmission(submission: BackendSubmission): Submission {
  const studentId = submission.student?.id ?? submission.student_id ?? "";
  return {
    id: submission.id,
    assignment_id: submission.assignment_id,
    student: submission.student ?? {
      id: studentId,
      full_name: studentId ? `Student ${studentId.slice(0, 8)}` : "Student",
    },
    submitted_at: submission.submitted_at ?? submission.created_at ?? new Date(0).toISOString(),
    status:
      submission.status === "graded" || submission.latest_grade
        ? "graded"
        : submission.status === "returned"
          ? "returned"
          : "submitted",
    score: submission.score ?? submission.latest_grade?.score,
    files: submission.files?.map((file) => ({
      name: file.name ?? file.original_filename ?? "Attachment",
      url: file.url ?? file.download_url ?? "#",
    })),
    text: submission.text ?? submission.text_content ?? "",
  };
}

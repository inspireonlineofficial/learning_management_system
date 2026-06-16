import { apiRequest } from "./client";

export type AssignmentStatus =
  | "not_submitted"
  | "submitted"
  | "graded"
  | "revision_requested"
  | "late"
  | "missed";

export type AssignmentAttachment = {
  id: string;
  filename: string;
  url: string;
  size_bytes?: number;
  mime_type?: string;
};

export type Rubric = {
  criteria: { id: string; title: string; description?: string; points: number }[];
  total_points: number;
};

export type AssignmentSummary = {
  id: string;
  course_id: string;
  course_title?: string;
  title: string;
  due_at?: string | null;
  total_points: number;
  status: AssignmentStatus;
  grade?: number | null;
};

type BackendAssignment = {
  id: string;
  course_id: string;
  title: string;
  description?: string;
  due_at?: string | null;
  submission_type?: "text" | "file" | "both";
  max_file_size_mb?: number;
  allow_late_submission?: boolean;
  total_marks?: number;
  total_points?: number;
  submission?: {
    status?: AssignmentStatus;
    latest_grade?: { score?: number } | null;
  } | null;
};

export type Assignment = AssignmentSummary & {
  brief: string;
  instructions_html?: string;
  rubric?: Rubric;
  resources?: AssignmentAttachment[];
  allow_resubmission?: boolean;
  late_penalty_percent?: number;
  submission?: Submission | null;
};

export type Submission = {
  id: string;
  assignment_id: string;
  submitted_at: string;
  status: AssignmentStatus;
  text?: string;
  attachments: AssignmentAttachment[];
  grade?: number | null;
  feedback?: string;
  graded_at?: string | null;
  graded_by?: { id: string; full_name: string } | null;
  revision_requested?: boolean;
  resubmission_deadline?: string | null;
};

export function listCourseAssignments(courseId: string) {
  return apiRequest<{ assignments?: BackendAssignment[]; data?: BackendAssignment[] }>(
    `/v1/teacher/courses/${encodeURIComponent(courseId)}/assignments`,
    { auth: true },
  ).then((result) => ({
    data: (result.assignments ?? result.data ?? []).map(toAssignmentSummary),
  }));
}

export function listMyAssignments(params: { status?: string; page?: number; limit?: number } = {}) {
  return apiRequest<{
    assignments?: BackendAssignment[];
    data?: BackendAssignment[];
    meta: { page: number; limit: number; total: number; total_pages: number };
  }>("/v1/student/assignments", { auth: true, query: params }).then((result) => ({
    data: (result.assignments ?? result.data ?? []).map(toAssignmentSummary),
    meta: result.meta ?? {
      page: params.page ?? 1,
      limit: params.limit ?? 20,
      total: 0,
      total_pages: 1,
    },
  }));
}

export function getAssignment(id: string) {
  return apiRequest<Assignment>(`/v1/student/assignments/${encodeURIComponent(id)}`, {
    auth: true,
  });
}

export type SubmissionPayload = {
  text?: string;
  attachments?: { filename: string; url: string; size_bytes?: number; mime_type?: string }[];
};

export function submitAssignment(assignmentId: string, payload: SubmissionPayload) {
  return apiRequest<Submission>(`/v1/assignments/${encodeURIComponent(assignmentId)}/submissions`, {
    method: "POST",
    auth: true,
    body: payload,
  });
}

export function getSubmission(submissionId: string) {
  void submissionId;
  return Promise.reject(new Error("Open the assignment detail page to view submission feedback."));
}

export function resubmitAssignment(submissionId: string, payload: SubmissionPayload) {
  void payload;
  return getSubmission(submissionId);
}

export type TeacherAssignmentInput = {
  title: string;
  description: string;
  due_at: string;
  submission_type: "text" | "file" | "both";
  max_file_size_mb: number;
  allow_late_submission: boolean;
  total_marks: number;
};

export type TeacherAssignmentDetail = AssignmentSummary & {
  description: string;
  submission_type: "text" | "file" | "both";
  max_file_size_mb: number;
  allow_late_submission: boolean;
  total_marks: number;
};

export function createTeacherAssignment(courseId: string, input: TeacherAssignmentInput) {
  return apiRequest<BackendAssignment>(
    `/v1/teacher/courses/${encodeURIComponent(courseId)}/assignments`,
    {
      method: "POST",
      auth: true,
      body: input,
    },
  ).then((assignment) => ({
    ...toAssignmentSummary(assignment),
    brief: assignment.description ?? "",
    instructions_html: assignment.description ?? "",
    allow_resubmission: false,
    late_penalty_percent: 0,
  }));
}

export function getTeacherAssignment(assignmentId: string) {
  return apiRequest<BackendAssignment>(
    `/v1/teacher/assignments/${encodeURIComponent(assignmentId)}`,
    { auth: true },
  ).then(toTeacherAssignmentDetail);
}

export function updateTeacherAssignment(assignmentId: string, input: TeacherAssignmentInput) {
  return apiRequest<BackendAssignment>(
    `/v1/teacher/assignments/${encodeURIComponent(assignmentId)}`,
    {
      method: "PATCH",
      auth: true,
      body: input,
    },
  ).then(toTeacherAssignmentDetail);
}

function toAssignmentSummary(assignment: BackendAssignment): AssignmentSummary {
  return {
    id: assignment.id,
    course_id: assignment.course_id,
    title: assignment.title,
    due_at: assignment.due_at ?? null,
    total_points: assignment.total_points ?? assignment.total_marks ?? 0,
    status: assignment.submission?.status ?? "not_submitted",
    grade: assignment.submission?.latest_grade?.score ?? null,
  };
}

function toTeacherAssignmentDetail(assignment: BackendAssignment): TeacherAssignmentDetail {
  return {
    ...toAssignmentSummary(assignment),
    description: assignment.description ?? "",
    submission_type: assignment.submission_type ?? "both",
    max_file_size_mb: assignment.max_file_size_mb ?? 50,
    allow_late_submission: assignment.allow_late_submission ?? false,
    total_marks: assignment.total_marks ?? assignment.total_points ?? 0,
  };
}

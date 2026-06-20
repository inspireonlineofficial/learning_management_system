import { apiRequest } from "./client";

export type CourseAccessRequestStatus = "pending" | "approved" | "rejected";

export type CourseAccessRequest = {
  id: string;
  student_id: string;
  student_name: string;
  student_email: string;
  item_type: "course";
  item_id: string;
  item_title: string;
  item_subtitle?: string;
  status: CourseAccessRequestStatus;
  rejection_reason?: string;
  result_enrollment_id?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  created_at: string;
  updated_at: string;
};

export type CourseAccessRequestList = {
  data: CourseAccessRequest[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export function requestCourseAccess(courseId: string, note = "") {
  return apiRequest<CourseAccessRequest>("/v1/purchase-requests", {
    method: "POST",
    auth: true,
    body: {
      item_type: "course",
      item_id: courseId,
      file_name: note,
    },
  });
}

export function listMyCourseAccessRequests(
  params: { status?: CourseAccessRequestStatus; page?: number; limit?: number } = {},
) {
  return apiRequest<CourseAccessRequestList>("/v1/student/purchase-requests", {
    auth: true,
    query: { ...params, item_type: "course" },
  });
}

export function listAdminCourseAccessRequests(
  params: { status?: CourseAccessRequestStatus; page?: number; limit?: number } = {},
) {
  return apiRequest<CourseAccessRequestList>("/v1/admin/purchase-requests", {
    auth: true,
    query: { ...params, item_type: "course" },
  });
}

export function approveCourseAccessRequest(requestId: string) {
  return apiRequest<CourseAccessRequest>(
    `/v1/admin/purchase-requests/${encodeURIComponent(requestId)}/approve`,
    { method: "POST", auth: true },
  );
}

export function rejectCourseAccessRequest(requestId: string, reason: string) {
  return apiRequest<CourseAccessRequest>(
    `/v1/admin/purchase-requests/${encodeURIComponent(requestId)}/reject`,
    { method: "POST", auth: true, body: { reason } },
  );
}

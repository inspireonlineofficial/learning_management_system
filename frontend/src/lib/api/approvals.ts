import { apiRequest } from "./client";

export type ApprovalItem = {
  id: string;
  kind: "course_publish" | "book_purchase" | "refund" | "role_change";
  requester: { id: string; full_name: string };
  payload_summary: string;
  created_at: string;
};
export const listApprovals = () =>
  apiRequest<{ items: ApprovalItem[] }>("/v1/admin/approvals", { auth: true });
export const approve = (id: string, note?: string) =>
  apiRequest<{ ok: true }>(`/v1/admin/approvals/${id}/approve`, {
    method: "POST",
    auth: true,
    body: { note },
  });
export const reject = (id: string, note?: string) =>
  apiRequest<{ ok: true }>(`/v1/admin/approvals/${id}/reject`, {
    method: "POST",
    auth: true,
    body: { note },
  });

export const createPurchaseRequest = (input: {
  item_type: "book" | "course";
  item_id: string;
  note?: string;
}) =>
  apiRequest<{ id: string }>("/v1/student/requests", { method: "POST", auth: true, body: input });
export const listMyRequests = () =>
  apiRequest<{ items: ApprovalItem[] }>("/v1/student/requests", { auth: true });

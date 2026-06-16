import { apiRequest } from "./client";

export type RefundRequest = {
  id: string;
  order_id: string;
  requester: { id: string; full_name: string };
  reason: string;
  amount: number;
  currency: string;
  status: "pending" | "approved" | "rejected" | "processed";
  created_at: string;
};
export const listRefunds = () =>
  apiRequest<{ items: RefundRequest[] }>("/v1/admin/bookshop/refunds", {
    auth: true,
  });
export const decideRefund = (id: string, action: "approve" | "reject", note?: string) =>
  apiRequest<{ ok: true }>("/v1/admin/bookshop/refunds", {
    method: "POST",
    auth: true,
    body: { order_id: id, action, note },
  });

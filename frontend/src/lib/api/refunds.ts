import { apiRequest } from "./client";
import type { OrderRefund, PayoutMethod, RefundRequest } from "./bookshop";

/** A real order an admin can refund (or has refunded). */
export type RefundableOrder = {
  id: string;
  order_id: string;
  requester: { id: string; full_name: string; email?: string };
  book_id: string;
  book_title: string;
  format: "digital" | "physical";
  amount: number;
  currency: string;
  order_status: "placed" | "shipped" | "delivered" | "refunded";
  refundable: boolean;
  refund?: OrderRefund;
  pending_request_id?: string;
  created_at: string;
  updated_at: string;
};

export const listRefunds = () =>
  apiRequest<{ items: RefundableOrder[] }>("/v1/admin/bookshop/refunds", { auth: true });

export type PayoutInput = {
  payoutMethod: PayoutMethod;
  payoutReference?: string;
  /** Omit for the full order amount. */
  amount?: number;
  note?: string;
};

type RefundResult = {
  order_id: string;
  status: "refunded";
  stock_restored: boolean;
  access_revoked: boolean;
  refund?: OrderRefund;
};

const payoutBody = (p: PayoutInput) => ({
  payout_method: p.payoutMethod,
  payout_reference: p.payoutReference || undefined,
  amount: p.amount,
  note: p.note || undefined,
});

/**
 * Refunds an order and records how the money was returned. The LMS does not
 * move money (there is no payment gateway): send the refund first, then
 * record it here with its transaction ID. The idempotency key makes a double
 * click or a retried request apply once.
 */
export const issueRefund = (orderId: string, idempotencyKey: string, payout: PayoutInput) =>
  apiRequest<RefundResult>("/v1/admin/bookshop/refunds", {
    method: "POST",
    auth: true,
    headers: { "Idempotency-Key": idempotencyKey },
    body: { order_id: orderId, ...payoutBody(payout) },
  });

export const listRefundRequests = (status?: RefundRequest["status"]) =>
  apiRequest<{ data: RefundRequest[]; meta: { total: number } }>(
    "/v1/admin/bookshop/refund-requests",
    { auth: true, query: { status, limit: 100 } },
  );

export const approveRefundRequest = (requestId: string, payout: PayoutInput) =>
  apiRequest<RefundResult>(
    `/v1/admin/bookshop/refund-requests/${encodeURIComponent(requestId)}/approve`,
    { method: "POST", auth: true, body: payoutBody(payout) },
  );

export const rejectRefundRequest = (requestId: string, note: string) =>
  apiRequest<RefundRequest>(
    `/v1/admin/bookshop/refund-requests/${encodeURIComponent(requestId)}/reject`,
    { method: "POST", auth: true, body: { note } },
  );

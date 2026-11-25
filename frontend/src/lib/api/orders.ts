import { apiRequest } from "./client";
import type { BookOrder } from "./bookshop";

export type { BookOrder as Order } from "./bookshop";

type OrderList = {
  data: BookOrder[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export const listMyOrders = (query?: { page?: number; limit?: number }) =>
  apiRequest<OrderList>("/v1/student/bookshop/orders", { auth: true, query }).then((result) => ({
    items: result.data ?? [],
    meta: result.meta,
  }));

export const getMyOrder = (id: string) =>
  apiRequest<BookOrder>(`/v1/student/bookshop/orders/${encodeURIComponent(id)}`, { auth: true });

export const listAllOrders = (query?: { status?: string; page?: number; limit?: number }) =>
  apiRequest<OrderList>("/v1/admin/bookshop/orders", { auth: true, query }).then((result) => ({
    items: result.data ?? [],
    meta: result.meta,
  }));

/** Admin fulfilment: placed → shipped → delivered, or placed → cancelled. */
export const updateOrderStatus = (
  id: string,
  status: "shipped" | "delivered" | "cancelled",
  trackingNumber?: string,
) =>
  apiRequest<BookOrder>(`/v1/admin/bookshop/orders/${encodeURIComponent(id)}`, {
    method: "PATCH",
    auth: true,
    body: { status, tracking_number: trackingNumber },
  });

export const orderStatusLabel: Record<BookOrder["status"], string> = {
  placed: "Placed",
  shipped: "Shipped",
  delivered: "Delivered",
  refunded: "Refunded",
  cancelled: "Cancelled",
};

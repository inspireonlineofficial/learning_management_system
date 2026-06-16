import { apiRequest } from "./client";

export type OrderItem = { id: string; title: string; quantity: number; unit_price: number };
export type Order = {
  id: string;
  status: "pending" | "paid" | "placed" | "shipped" | "delivered" | "refunded" | "cancelled";
  items: OrderItem[];
  total: number;
  currency: string;
  created_at: string;
  shipping_address?: string;
};
export const listMyOrders = () =>
  apiRequest<{ items: Order[] }>("/v1/student/bookshop/orders", { auth: true });
export const getMyOrder = (id: string) =>
  apiRequest<Order>(`/v1/student/bookshop/orders/${id}`, { auth: true });
export const listAllOrders = (query?: { status?: string; q?: string }) =>
  apiRequest<{ items?: Order[]; data?: Order[] }>("/v1/admin/bookshop/orders", {
    auth: true,
    query,
  }).then((result) => ({ items: result.items ?? result.data ?? [] }));

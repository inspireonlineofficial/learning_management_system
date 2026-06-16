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

type BackendOrderItem = Partial<OrderItem> & {
  book?: { id?: string; title?: string };
  book_id?: string;
  unit_price_cents?: number;
};

type BackendOrder = Partial<Order> & {
  data?: unknown;
  amount?: number;
  total_cents?: number;
  items?: BackendOrderItem[];
};

export const listMyOrders = () =>
  apiRequest<{ items?: BackendOrder[]; data?: BackendOrder[] }>("/v1/student/bookshop/orders", {
    auth: true,
  }).then((result) => ({ items: (result.items ?? result.data ?? []).map(normalizeOrder) }));
export const getMyOrder = (id: string) =>
  apiRequest<BackendOrder>(`/v1/student/bookshop/orders/${id}`, { auth: true }).then(
    normalizeOrder,
  );
export const listAllOrders = (query?: { status?: string; q?: string }) =>
  apiRequest<{ items?: BackendOrder[]; data?: BackendOrder[] }>("/v1/admin/bookshop/orders", {
    auth: true,
    query,
  }).then((result) => ({ items: (result.items ?? result.data ?? []).map(normalizeOrder) }));

function normalizeOrder(order: BackendOrder): Order {
  const total =
    typeof order.total === "number"
      ? order.total
      : typeof order.total_cents === "number"
        ? order.total_cents / 100
        : typeof order.amount === "number"
          ? order.amount
          : 0;

  return {
    id: order.id ?? "order",
    status: order.status ?? "pending",
    items: (order.items ?? []).map((item, index) => ({
      id: item.id ?? item.book_id ?? item.book?.id ?? `item-${index}`,
      title: item.title ?? item.book?.title ?? "Book",
      quantity: item.quantity ?? 1,
      unit_price:
        typeof item.unit_price === "number"
          ? item.unit_price
          : typeof item.unit_price_cents === "number"
            ? item.unit_price_cents / 100
            : 0,
    })),
    total,
    currency: order.currency ?? "USD",
    created_at: order.created_at ?? new Date().toISOString(),
    shipping_address: order.shipping_address,
  };
}

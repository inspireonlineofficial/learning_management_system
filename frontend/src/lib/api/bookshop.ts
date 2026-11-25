import { apiRequest } from "./client";

export type BookSummary = {
  id: string;
  title: string;
  author: string;
  cover_url?: string | null;
  price_cents: number;
  currency: string;
  category?: string;
  rating?: number | null;
  is_free?: boolean;
  in_library?: boolean;
  physical_stock?: number;
  format?: "physical" | "digital" | "both";
};

type BackendBook = {
  id: string;
  title: string;
  author: string;
  subject?: string;
  class_grade?: string;
  description?: string;
  format?: string;
  price: number;
  currency: string;
  cover_url?: string | null;
  physical_stock?: number;
  is_active?: boolean;
  has_preview?: boolean;
  has_digital_file?: boolean;
};

function toSummary(book: BackendBook): BookSummary {
  return {
    id: book.id,
    title: book.title,
    author: book.author,
    cover_url: book.cover_url,
    price_cents: Math.round((book.price ?? 0) * 100),
    currency: book.currency || "BDT",
    category: book.subject || book.class_grade,
    is_free: (book.price ?? 0) === 0,
    physical_stock: book.physical_stock,
    format: book.format as BookSummary["format"],
  };
}

/** Whether a book can be bought as a digital edition / a printed copy. */
export const sellsDigital = (book: Pick<BookSummary, "format">) =>
  book.format === "digital" || book.format === "both";
export const sellsPrinted = (book: Pick<BookSummary, "format">) =>
  book.format === "physical" || book.format === "both";

export type BookDetail = BookSummary & {
  description?: string;
  subject?: string;
  class_grade?: string;
  format?: "physical" | "digital" | "both";
  physical_stock?: number;
  is_active?: boolean;
  publisher?: string;
  published_at?: string;
  pages?: number;
  language?: string;
  isbn?: string;
  table_of_contents?: { id: string; title: string; page?: number }[];
  preview_url?: string | null;
  has_preview?: boolean;
  has_digital_file?: boolean;
};

export type Paginated<T> = {
  data: T[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export function listBooks(
  params: {
    search?: string;
    category?: string;
    sort?: "newest" | "price_asc" | "price_desc" | "title";
    page?: number;
    limit?: number;
  } = {},
) {
  const query = {
    search: params.search,
    subject: params.category,
    sort: params.sort,
    page: params.page,
    limit: params.limit,
  };
  return apiRequest<{ data: BackendBook[]; meta: Paginated<BookSummary>["meta"] }>(
    "/v1/bookshop/books",
    { query },
  ).then((result) => ({
    data: result.data.map(toSummary),
    meta: result.meta,
  }));
}

function toDetail(book: BackendBook): BookDetail {
  return {
    ...toSummary(book),
    description: book.description,
    subject: book.subject,
    class_grade: book.class_grade,
    format: book.format as BookDetail["format"],
    physical_stock: book.physical_stock,
    is_active: book.is_active,
    has_preview: book.has_preview,
    has_digital_file: book.has_digital_file,
  };
}

export function getBook(bookId: string): Promise<BookDetail> {
  return apiRequest<BackendBook>(`/v1/bookshop/books/${encodeURIComponent(bookId)}`).then(toDetail);
}

/** Admin view of a book, including hidden ones. */
export function getAdminBook(bookId: string): Promise<BookDetail> {
  return apiRequest<BackendBook>(`/v1/admin/bookshop/books/${encodeURIComponent(bookId)}`, {
    auth: true,
  }).then(toDetail);
}

/** Whether the signed-in student already owns the digital edition. */
export function getBookOwnership(bookId: string) {
  return apiRequest<{ book_id: string; in_library: boolean; order_id?: string }>(
    `/v1/student/bookshop/books/${encodeURIComponent(bookId)}/status`,
    { auth: true },
  );
}

/** Short-lived signed URL for the free sample pages. */
export function getBookPreview(bookId: string) {
  return apiRequest<{ preview_url: string; expires_in: number }>(
    `/v1/bookshop/books/${encodeURIComponent(bookId)}/preview`,
    { auth: true },
  );
}

export type ShippingAddress = {
  name: string;
  phone: string;
  line1: string;
  line2?: string;
  city: string;
  postal_code?: string;
};

/** Free books: add the digital edition to the library or order a printed copy. */
export function orderFreeBook(
  bookId: string,
  format: "digital" | "physical",
  shippingAddress?: ShippingAddress,
) {
  return apiRequest<BookOrder>("/v1/bookshop/orders", {
    method: "POST",
    auth: true,
    headers: { "Idempotency-Key": newIdempotencyKey() },
    body: { book_id: bookId, format, shipping_address: shippingAddress },
  });
}

export type BookPurchaseRequest = {
  id: string;
  item_id: string;
  item_title: string;
  item_subtitle?: string;
  book_format?: "digital" | "physical";
  shipping_address?: ShippingAddress;
  file_name?: string;
  status: "pending" | "approved" | "rejected";
  rejection_reason?: string;
  result_order_id?: string;
  created_at: string;
  updated_at: string;
};

/** Paid books: ask an admin to confirm the payment and approve the purchase. */
export function requestBookPurchase(input: {
  bookId: string;
  format: "digital" | "physical";
  shippingAddress?: ShippingAddress;
  note?: string;
}) {
  return apiRequest<BookPurchaseRequest>("/v1/purchase-requests", {
    method: "POST",
    auth: true,
    body: {
      item_type: "book",
      item_id: input.bookId,
      format: input.format,
      shipping_address: input.format === "physical" ? input.shippingAddress : undefined,
      file_name: input.note ?? "",
    },
  });
}

export function listMyBookRequests(params: { page?: number; limit?: number } = {}) {
  return apiRequest<{
    data: BookPurchaseRequest[];
    meta: { page: number; limit: number; total: number; total_pages: number };
  }>("/v1/student/purchase-requests", {
    auth: true,
    query: { ...params, item_type: "book" },
  });
}

export const EMPTY_ADDRESS: ShippingAddress = {
  name: "",
  phone: "",
  line1: "",
  line2: "",
  city: "",
  postal_code: "",
};

export function formatAddress(a?: ShippingAddress | null) {
  if (!a) return "—";
  return [a.name, a.line1, a.line2, a.city, a.postal_code, a.phone].filter(Boolean).join(", ");
}

export function newIdempotencyKey() {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export type CartItem = {
  /** "<book_id>:<format>" */
  id: string;
  book: BookSummary;
  format: "digital" | "physical";
  quantity: 1;
  unit_price_cents: number;
  free: boolean;
  available: boolean;
  problem?: string;
};

export type Cart = {
  id: string;
  items: CartItem[];
  subtotal_cents: number;
  total_cents: number;
  currency: string;
  needs_address: boolean;
  has_paid_items: boolean;
};

type BackendCart = Omit<Cart, "items"> & {
  items: Array<Omit<CartItem, "book"> & { book: BackendBook }>;
};

const toCart = (cart: BackendCart): Cart => ({
  ...cart,
  items: (cart.items ?? []).map((item) => ({ ...item, book: toSummary(item.book) })),
});

export function getCart() {
  return apiRequest<BackendCart>("/v1/bookshop/cart", { auth: true }).then(toCart);
}

export function addToCart(bookId: string, format: "digital" | "physical") {
  return apiRequest<BackendCart>("/v1/bookshop/cart/items", {
    method: "POST",
    auth: true,
    body: { book_id: bookId, format },
  }).then(toCart);
}

export function removeCartItem(itemId: string) {
  return apiRequest<BackendCart>(`/v1/bookshop/cart/items/${encodeURIComponent(itemId)}`, {
    method: "DELETE",
    auth: true,
  }).then(toCart);
}

export type CartCheckoutResult = {
  orders: BookOrder[];
  requests: BookPurchaseRequest[];
  failed: { item_id: string; title: string; error: string }[];
  cart: Cart;
};

/** Free items become orders (all or none); paid items become purchase requests. */
export function checkoutCart(input: {
  shippingAddress?: ShippingAddress;
  paymentReference?: string;
}) {
  return apiRequest<Omit<CartCheckoutResult, "cart"> & { cart: BackendCart }>(
    "/v1/bookshop/cart/checkout",
    {
      method: "POST",
      auth: true,
      body: { shipping_address: input.shippingAddress, payment_reference: input.paymentReference },
    },
  ).then((r) => ({ ...r, cart: toCart(r.cart) }));
}

export type PayoutMethod = "bkash" | "nagad" | "rocket" | "bank" | "cash" | "other" | "none";

export type OrderRefund = {
  amount: number;
  currency: string;
  payout_method: PayoutMethod;
  payout_reference?: string;
  note?: string;
  refunded_at: string;
};

export type RefundRequest = {
  id: string;
  order_id: string;
  student_id: string;
  student_name?: string;
  student_email?: string;
  book_title: string;
  format: "digital" | "physical";
  amount: number;
  currency: string;
  order_status: string;
  reason: string;
  status: "pending" | "approved" | "rejected";
  admin_note?: string;
  reviewed_at?: string;
  created_at: string;
};

export function requestRefund(orderId: string, reason: string) {
  return apiRequest<RefundRequest>(
    `/v1/student/bookshop/orders/${encodeURIComponent(orderId)}/refund-request`,
    { method: "POST", auth: true, body: { reason } },
  );
}

export const payoutMethodLabel: Record<PayoutMethod, string> = {
  bkash: "bKash",
  nagad: "Nagad",
  rocket: "Rocket",
  bank: "Bank transfer",
  cash: "Cash",
  other: "Other",
  none: "No payment (free order)",
};

export type BookOrder = {
  id: string;
  book_id: string;
  book_title: string;
  format: "digital" | "physical";
  amount: number;
  currency: string;
  status: "placed" | "shipped" | "delivered" | "refunded" | "cancelled";
  tracking_number?: string | null;
  shipping_address?: ShippingAddress;
  shipped_at?: string;
  delivered_at?: string;
  refund_info?: {
    eligible: boolean;
    deadline?: string;
    blocked_reason?: string;
    request?: RefundRequest;
    refund?: OrderRefund;
  };
  student_name?: string;
  student_email?: string;
  created_at: string;
  updated_at: string;
};

export type LibraryBook = BookSummary & {
  order_id?: string;
  purchased_at?: string;
  last_page_read?: number;
  reading_enabled?: boolean;
};

export function listMyLibrary(params: { page?: number; limit?: number } = {}) {
  return apiRequest<{
    data: Array<
      BackendBook & {
        order_id?: string;
        purchased_at?: string;
        last_page_read?: number;
        reading_enabled?: boolean;
      }
    >;
    meta: Paginated<BookSummary>["meta"];
  }>("/v1/student/bookshop/library", { auth: true, query: params }).then((result) => ({
    data: result.data.map((book): LibraryBook => ({
      ...toSummary(book),
      in_library: true,
      order_id: book.order_id,
      purchased_at: book.purchased_at,
      last_page_read: book.last_page_read,
      reading_enabled: book.reading_enabled,
    })),
    meta: result.meta,
  }));
}

export type ReaderAccess = {
  book_id: string;
  /** Signed URL for the PDF; valid for `expires_in` seconds. */
  access_url: string;
  expires_in: number;
  last_page_read: number;
};

export function getReaderAccess(bookId: string) {
  return apiRequest<ReaderAccess>(
    `/v1/student/bookshop/reader/${encodeURIComponent(bookId)}/access`,
    { auth: true },
  );
}

export function saveReadingPosition(bookId: string, page: number) {
  return apiRequest<{ last_page_read: number }>(
    `/v1/student/bookshop/reader/${encodeURIComponent(bookId)}/bookmark`,
    {
      method: "POST",
      auth: true,
      body: { last_page_read: Math.max(0, Math.floor(page)) },
    },
  );
}

export function formatPrice(cents: number, currency = "BDT") {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency,
  }).format(cents / 100);
}

export type AdminBookInput = {
  title: string;
  author: string;
  subject?: string;
  class_grade?: string;
  description?: string;
  format?: "physical" | "digital" | "both";
  price?: number;
  currency?: string;
  physical_stock?: number;
  is_active?: boolean;
};

export function listAdminBooks(params: { page?: number; limit?: number } = {}) {
  return apiRequest<{ data: BackendBook[]; meta: Paginated<BookSummary>["meta"] }>(
    "/v1/admin/bookshop/books",
    { auth: true, query: params },
  ).then((result) => ({
    data: result.data.map(toSummary),
    meta: result.meta,
  }));
}

export function createAdminBook(input: AdminBookInput) {
  return apiRequest<BackendBook>("/v1/admin/bookshop/books", {
    method: "POST",
    auth: true,
    body: input,
  }).then(toSummary);
}

export function updateAdminBook(bookId: string, input: Partial<AdminBookInput>) {
  return apiRequest<BackendBook>(`/v1/admin/bookshop/books/${encodeURIComponent(bookId)}`, {
    method: "PATCH",
    auth: true,
    body: input,
  }).then(toSummary);
}

/** Uploads the full digital edition ("file"), the sample ("preview") or the cover. */
export function uploadBookAsset(bookId: string, kind: "file" | "preview" | "cover", file: File) {
  const formData = new FormData();
  formData.append(kind === "cover" ? "cover" : "file", file);
  return apiRequest<BackendBook>(`/v1/admin/bookshop/books/${encodeURIComponent(bookId)}/${kind}`, {
    method: "POST",
    auth: true,
    body: formData,
  }).then(toDetail);
}

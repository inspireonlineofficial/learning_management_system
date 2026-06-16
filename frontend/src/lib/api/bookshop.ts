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
};

function toSummary(book: BackendBook): BookSummary {
  return {
    id: book.id,
    title: book.title,
    author: book.author,
    cover_url: book.cover_url,
    price_cents: Math.round((book.price ?? 0) * 100),
    currency: book.currency ?? "USD",
    category: book.subject ?? book.class_grade,
    is_free: (book.price ?? 0) === 0,
    physical_stock: (book as BackendBook & { physical_stock?: number }).physical_stock,
    format: book.format as BookSummary["format"],
  };
}

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
};

export type Paginated<T> = {
  data: T[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export function listBooks(
  params: {
    search?: string;
    category?: string;
    sort?: "popular" | "newest" | "price_asc" | "price_desc";
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

export function getBook(bookId: string) {
  return apiRequest<BackendBook>(`/v1/bookshop/books/${encodeURIComponent(bookId)}`).then(
    (book) =>
      ({
        ...toSummary(book),
        description: book.description,
        subject: book.subject,
        class_grade: book.class_grade,
        format: book.format as BookDetail["format"],
        physical_stock: (book as BackendBook & { physical_stock?: number }).physical_stock,
        is_active: (book as BackendBook & { is_active?: boolean }).is_active,
        language: "English",
      }) satisfies BookDetail,
  );
}

export type CartItem = {
  id: string;
  book: BookSummary;
  quantity: number;
  unit_price_cents: number;
};

export type Cart = {
  id: string;
  items: CartItem[];
  subtotal_cents: number;
  total_cents: number;
  currency: string;
};

export function getCart() {
  return apiRequest<Cart>("/v1/bookshop/cart", { auth: true }).then(normalizeCart);
}

export function addToCart(bookId: string, quantity = 1) {
  return apiRequest<Cart>("/v1/bookshop/cart/items", {
    method: "POST",
    auth: true,
    body: { book_id: bookId, quantity },
  }).then(normalizeCart);
}

export function updateCartItem(itemId: string, quantity: number) {
  return apiRequest<Cart>(`/v1/bookshop/cart/items/${encodeURIComponent(itemId)}`, {
    method: "PATCH",
    auth: true,
    body: { quantity },
  }).then(normalizeCart);
}

export function removeCartItem(itemId: string) {
  return apiRequest<Cart>(`/v1/bookshop/cart/items/${encodeURIComponent(itemId)}`, {
    method: "DELETE",
    auth: true,
  }).then(normalizeCart);
}

function normalizeCart(cart: Cart): Cart {
  return {
    ...cart,
    items: cart.items.map((item) => ({
      ...item,
      book: {
        ...item.book,
        price_cents: item.book.price_cents ?? item.unit_price_cents,
      },
    })),
  };
}

function toCart(items: CartItem[]): Cart {
  const subtotal = items.reduce((sum, item) => sum + item.unit_price_cents * item.quantity, 0);
  return {
    id: "checkout",
    items,
    subtotal_cents: subtotal,
    total_cents: subtotal,
    currency: items[0]?.book.currency ?? "USD",
  };
}

export function checkout() {
  return getCart().then((cart) => {
    const items = cart.items;
    return apiRequest<{
      orders: Array<{
        id: string;
        amount: number;
        currency: string;
        status: Order["status"];
        created_at: string;
      }>;
    }>("/v1/bookshop/checkout", {
      method: "POST",
      auth: true,
      body: {
        items: items.map((item) => ({
          book_id: item.book.id,
          format: "digital",
        })),
      },
    }).then((result) => {
      const firstOrder = result.orders[0];
      return {
        order: {
          id: firstOrder?.id ?? "checkout",
          status: firstOrder?.status ?? "pending",
          total_cents: result.orders.reduce(
            (sum, order) => sum + Math.round(order.amount * 100),
            0,
          ),
          currency: firstOrder?.currency ?? toCart(items).currency,
          created_at: firstOrder?.created_at ?? new Date().toISOString(),
          items,
        },
      };
    });
  });
}

export type Order = {
  id: string;
  status:
    | "pending"
    | "paid"
    | "failed"
    | "placed"
    | "shipped"
    | "delivered"
    | "refunded"
    | "cancelled";
  total_cents: number;
  currency: string;
  created_at: string;
  items: CartItem[];
  receipt_url?: string;
};

export function listMyOrders(params: { page?: number; limit?: number } = {}) {
  return apiRequest<{
    data: Array<{
      id: string;
      book_id: string;
      amount: number;
      currency: string;
      status: Order["status"];
      created_at: string;
    }>;
    meta: Paginated<Order>["meta"];
  }>("/v1/student/bookshop/orders", { auth: true, query: params }).then((result) => ({
    data: result.data.map((order) => ({
      id: order.id,
      status: order.status,
      total_cents: Math.round(order.amount * 100),
      currency: order.currency,
      created_at: order.created_at,
      items: [],
    })),
    meta: result.meta,
  }));
}

export function listMyLibrary(params: { page?: number; limit?: number } = {}) {
  return listBooks(params);
}

export type BookContent = {
  book: BookDetail;
  chapters: { id: string; title: string; body_html: string; position: number }[];
  bookmark?: { chapter_id: string; position?: number } | null;
};

export function readBook(bookId: string) {
  return apiRequest<{ access_url: string; last_page_read: number }>(
    `/v1/student/bookshop/reader/${encodeURIComponent(bookId)}/access`,
    {
      auth: true,
    },
  ).then(async (access) => ({
    book: await getBook(bookId),
    chapters: [
      {
        id: "access",
        title: "Digital book",
        body_html: `<p><a href="${access.access_url}" target="_blank" rel="noreferrer">Open the digital book</a></p>`,
        position: access.last_page_read,
      },
    ],
    bookmark: { chapter_id: "access", position: access.last_page_read },
  }));
}

export function setBookmark(bookId: string, chapterId: string, position?: number) {
  void chapterId;
  return apiRequest<{ ok: true }>(
    `/v1/student/bookshop/reader/${encodeURIComponent(bookId)}/bookmark`,
    {
      method: "POST",
      auth: true,
      body: { last_page_read: position ?? 0 },
    },
  );
}

export function formatPrice(cents: number, currency = "USD") {
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
  digital_file_rustfs_key?: string;
  preview_rustfs_key?: string;
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

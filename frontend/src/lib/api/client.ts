import {
  clearStoredSession,
  getStoredSession,
  setStoredSession,
  type Session,
} from "@/lib/session";

export const API_BASE_URL = (() => {
  if (typeof window !== "undefined") {
    const host = window.location.hostname;
    const isLocal =
      host === "localhost" ||
      host === "127.0.0.1" ||
      host.endsWith(".local") ||
      host.endsWith(".internal");
    if (isLocal) {
      return (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "http://localhost:8080";
    }
    return window.location.origin;
  }
  return (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "http://localhost:8080";
})();

export class ApiError extends Error {
  status: number;
  code: string;
  details: unknown;
  constructor(
    message: string,
    options: { status?: number; code?: string; details?: unknown } = {},
  ) {
    super(message);
    this.name = "ApiError";
    this.status = options.status ?? 500;
    this.code = options.code ?? "REQUEST_FAILED";
    this.details = options.details ?? null;
  }
}

export function buildApiUrl(path: string, query?: Record<string, unknown>): string {
  const url = new URL(path, API_BASE_URL);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === undefined || value === null || value === "") continue;
      url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

type RequestOptions = {
  method?: string;
  body?: unknown;
  auth?: boolean;
  query?: Record<string, unknown>;
  headers?: Record<string, string>;
  signal?: AbortSignal;
  /** Keep the request alive while the page unloads (progress beacons). */
  keepalive?: boolean;
};

function extractRoleFromJwt(token: string): Session["user"]["role"] | null {
  try {
    const parts = token.split(".");
    if (parts.length < 2) return null;
    const base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, "=");
    const payload = JSON.parse(atob(padded)) as { role?: string };
    if (payload.role === "student" || payload.role === "teacher" || payload.role === "admin") {
      return payload.role;
    }
    return null;
  } catch {
    return null;
  }
}

let refreshPromise: Promise<Session | null> | null = null;

export async function refreshSession(): Promise<Session | null> {
  if (refreshPromise) return refreshPromise;
  const current = getStoredSession();
  if (!current?.refreshToken) return null;

  refreshPromise = fetch(buildApiUrl("/v1/auth/refresh"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: current.refreshToken }),
  })
    .then(async (response) => {
      if (!response.ok) {
        clearStoredSession();
        return null;
      }
      const data = (await response.json()) as {
        access_token: string;
        refresh_token?: string;
      };
      const jwtRole = extractRoleFromJwt(data.access_token);
      const next: Session = {
        ...current,
        accessToken: data.access_token,
        refreshToken: data.refresh_token ?? current.refreshToken,
        user: jwtRole ? { ...current.user, role: jwtRole } : current.user,
      };
      setStoredSession(next);
      return next;
    })
    .catch(() => {
      clearStoredSession();
      return null;
    })
    .finally(() => {
      refreshPromise = null;
    });

  return refreshPromise;
}

async function parseResponse(response: Response): Promise<unknown> {
  if (response.status === 204) return null;
  const ct = response.headers.get("content-type") ?? "";
  if (ct.includes("application/json")) return response.json();
  return response.text();
}

export async function apiRequest<T = unknown>(
  path: string,
  options: RequestOptions = {},
  _retry = false,
): Promise<T> {
  const { method = "GET", body, auth = false, query, headers = {}, signal, keepalive } = options;
  const init: RequestInit = {
    method,
    headers: {
      Accept: "application/json",
      ...headers,
    },
    signal,
    keepalive,
  };
  if (body instanceof FormData) {
    init.body = body;
  } else if (body !== undefined) {
    init.headers = {
      ...init.headers,
      "Content-Type": "application/json",
    };
    init.body = JSON.stringify(body);
  }

  if (auth) {
    const session = getStoredSession();
    if (session?.accessToken) {
      (init.headers as Record<string, string>)["Authorization"] = `Bearer ${session.accessToken}`;
    }
  }

  const response = await fetch(buildApiUrl(path, query), init);

  if (response.status === 401 && auth && !_retry) {
    const next = await refreshSession();
    if (next) return apiRequest<T>(path, options, true);
    clearStoredSession();
  }

  const data = await parseResponse(response);

  if (!response.ok) {
    // The API wraps errors as {"error": {"code", "message", "details"}};
    // flat {"code", "message"} bodies are accepted too. Reading only the flat
    // shape dropped every server message and error code.
    const raw = (data ?? {}) as {
      message?: string;
      code?: string;
      details?: unknown;
      error?: { message?: string; code?: string; details?: unknown };
    };
    const payload = raw.error && typeof raw.error === "object" ? raw.error : raw;
    throw new ApiError(payload.message ?? response.statusText ?? "Request failed", {
      status: response.status,
      code: payload.code,
      details: payload.details,
    });
  }

  return data as T;
}

/** Resolve API-relative media paths (/v1/...) against the API host. */
export function resolveAssetUrl(url: string): string {
  if (!url) return url;
  if (/^(https?:|data:|blob:)/i.test(url)) return url;
  if (url.startsWith("/v1/")) return `${API_BASE_URL}${url}`;
  return url;
}

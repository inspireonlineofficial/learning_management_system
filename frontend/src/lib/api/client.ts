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
};

let refreshPromise: Promise<Session | null> | null = null;

async function refreshSession(): Promise<Session | null> {
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
      const next: Session = {
        ...current,
        accessToken: data.access_token,
        refreshToken: data.refresh_token ?? current.refreshToken,
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
  const { method = "GET", body, auth = false, query, headers = {}, signal } = options;
  const init: RequestInit = {
    method,
    headers: {
      Accept: "application/json",
      ...headers,
    },
    signal,
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
    const payload = (data ?? {}) as { message?: string; code?: string; details?: unknown };
    throw new ApiError(payload.message ?? response.statusText ?? "Request failed", {
      status: response.status,
      code: payload.code,
      details: payload.details,
    });
  }

  return data as T;
}

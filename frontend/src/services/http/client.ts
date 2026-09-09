const API = "/api/v1";
const TOKEN_KEY = "ash.auth.token";
const REFRESH_KEY = "ash.auth.refresh";
const SPACE_KEY = "ash.space.id";

export class ApiError extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }
}

export async function api<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(opts.headers as Record<string, string>),
  };
  const token = getAuthToken();
  const spaceId = getCurrentSpaceId();
  if (token) headers.Authorization = `Bearer ${token}`;
  if (spaceId) headers["X-ASH-Space-ID"] = spaceId;
  const res = await fetch(API + path, {
    headers,
    ...opts,
  });
  const text = await res.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }
  if (!res.ok) {
    const err = data as { error?: { code?: string; message?: string } };
    const code = err?.error?.code || "REQUEST_FAILED";
    if (res.status === 401) {
      await handleAuthUnauthorized(code);
    }
    throw new ApiError(code, err?.error?.message || res.statusText);
  }
  return data as T;
}

async function handleAuthUnauthorized(code: string) {
  const { isAuthFailureCode, fetchConsoleAuthRequired } = await import(
    "@/modules/platform/auth/consoleGate"
  );
  if (!isAuthFailureCode(code)) return;
  clearAuthSession();
  const required = await fetchConsoleAuthRequired();
  if (!required) return;
  if (typeof window === "undefined") return;
  const path = window.location.pathname.replace(/\/ui\/?/, "/") || "/";
  if (path.includes("login")) return;
  const base = window.location.pathname.startsWith("/ui") ? "/ui" : "";
  window.location.assign(`${base}/login`);
}

export function getAuthToken() {
  return localStorage.getItem(TOKEN_KEY) || "";
}

export function getRefreshToken() {
  return localStorage.getItem(REFRESH_KEY) || "";
}

export function getCurrentSpaceId() {
  return localStorage.getItem(SPACE_KEY) || "local";
}

export function setAuthSession(token: string, spaceId: string, refreshToken?: string) {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  if (refreshToken) localStorage.setItem(REFRESH_KEY, refreshToken);
  localStorage.setItem(SPACE_KEY, spaceId || "local");
}

export function clearAuthSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
  localStorage.removeItem(SPACE_KEY);
}

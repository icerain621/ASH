const API = "/api/v1";
const TOKEN_KEY = "ash.auth.token";
const SPACE_KEY = "ash.space.id";
export const AUTH_SESSION_CHANGED = "ash.auth.session.changed";

export class ApiError extends Error {
  code: string;
  status: number;
  constructor(code: string, message: string, status = 0) {
    super(message);
    this.code = code;
    this.status = status;
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
    const apiError = new ApiError(
      err?.error?.code || "REQUEST_FAILED",
      err?.error?.message || res.statusText,
      res.status,
    );
    if (res.status === 401) {
      clearAuthSession();
    }
    throw apiError;
  }
  return data as T;
}

export function getAuthToken() {
  return localStorage.getItem(TOKEN_KEY) || "";
}

export function getCurrentSpaceId() {
  return localStorage.getItem(SPACE_KEY) || "local";
}

export function setAuthSession(token: string, spaceId: string) {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(SPACE_KEY, spaceId || "local");
  notifyAuthSessionChanged();
}

export function setCurrentSpaceId(spaceId: string) {
  localStorage.setItem(SPACE_KEY, spaceId || "local");
  notifyAuthSessionChanged();
}

export function getAuthSession() {
  return {
    token: getAuthToken(),
    spaceId: getCurrentSpaceId(),
  };
}

export function clearAuthSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(SPACE_KEY);
  notifyAuthSessionChanged();
}

function notifyAuthSessionChanged() {
  window.dispatchEvent(new Event(AUTH_SESSION_CHANGED));
}

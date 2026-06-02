const API = "/api/v1";

export class ApiError extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }
}

export async function api<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(API + path, {
    headers: { "Content-Type": "application/json", ...(opts.headers as Record<string, string>) },
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
    throw new ApiError(
      err?.error?.code || "REQUEST_FAILED",
      err?.error?.message || res.statusText,
    );
  }
  return data as T;
}

import { getReadyz, type ReadyzResponse } from "@/modules/health/api/health.api";
import { getAuthToken } from "@/services/http/client";

let cached: Promise<boolean> | null = null;

/** Reset cache (tests). */
export function resetConsoleAuthGateCache() {
  cached = null;
}

export function isConsoleAuthRequiredFlag(readyz: Pick<ReadyzResponse, "consoleAuthRequired">): boolean {
  return Boolean(readyz.consoleAuthRequired);
}

export async function fetchConsoleAuthRequired(): Promise<boolean> {
  if (!cached) {
    cached = getReadyz()
      .then((r) => isConsoleAuthRequiredFlag(r))
      .catch(() => false);
  }
  return cached;
}

/** Pure helper for route gate decisions. */
export function shouldRedirectToLogin(opts: {
  consoleAuthRequired: boolean;
  token: string;
  pathname: string;
}): boolean {
  if (!opts.consoleAuthRequired) return false;
  if (opts.token) return false;
  const path = opts.pathname.replace(/\/+$/, "") || "/";
  return path !== "/login";
}

export function shouldRedirectToLoginNow(pathname: string): Promise<boolean> {
  return fetchConsoleAuthRequired().then((required) =>
    shouldRedirectToLogin({
      consoleAuthRequired: required,
      token: getAuthToken(),
      pathname,
    }),
  );
}

export function isAuthFailureCode(code: string): boolean {
  if (!code) return false;
  if (code === "UNAUTHORIZED" || code === "REQUEST_FAILED") return true;
  return code.startsWith("AUTH_");
}

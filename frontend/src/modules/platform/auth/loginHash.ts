/** Parse OIDC / password redirect hash: #token=…&refreshToken=…&spaceId=… */
export function parseLoginHash(hash: string): { token: string; spaceId: string; refreshToken?: string } | null {
  const raw = hash.startsWith("#") ? hash.slice(1) : hash;
  if (!raw.trim()) return null;
  const params = new URLSearchParams(raw);
  const token = params.get("token")?.trim() || "";
  if (!token) return null;
  const spaceId = params.get("spaceId")?.trim() || "local";
  const refreshToken = params.get("refreshToken")?.trim() || undefined;
  return { token, spaceId, refreshToken };
}

export function oidcLoginHref(): string {
  return "/api/v1/auth/oidc/login?ui=1";
}

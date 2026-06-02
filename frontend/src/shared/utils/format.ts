export function fmtTime(ms?: number | null): string {
  if (!ms) return "—";
  return new Date(ms).toLocaleString();
}

export function shortId(id?: string): string {
  if (!id || id.length < 16) return id || "";
  return id.slice(0, 14) + "…";
}

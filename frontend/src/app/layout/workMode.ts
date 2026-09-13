/** Console work mode: Agent use vs review governance (v5 / DSH-aligned IA). */

export type WorkMode = "agent" | "review";

export const WORK_MODE_STORAGE_KEY = "ash.console.workMode";

export function workModeFromPath(pathname: string): WorkMode | null {
  const p = (pathname.replace(/\/+$/, "") || "/").toLowerCase();
  if (p === "/quest" || p.endsWith("/quest") || p.includes("/quest/")) return "agent";
  if (p === "/reviews" || p.endsWith("/reviews") || p.includes("/reviews/") || p.includes("/m/reviews")) {
    return "review";
  }
  return null;
}

export function persistWorkMode(mode: WorkMode): void {
  try {
    localStorage.setItem(WORK_MODE_STORAGE_KEY, mode);
  } catch {
    /* ignore quota / private mode */
  }
}

export function readPersistedWorkMode(): WorkMode | null {
  try {
    const v = localStorage.getItem(WORK_MODE_STORAGE_KEY);
    if (v === "agent" || v === "review") return v;
  } catch {
    /* ignore */
  }
  return null;
}

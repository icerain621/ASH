/** Console work mode: Agent · 记忆 · 评审管控 (three-pillar IA). */

export type WorkMode = "agent" | "memory" | "review";

export const WORK_MODE_STORAGE_KEY = "ash.console.workMode";

export function workModeFromPath(pathname: string): WorkMode | null {
  const p = (pathname.replace(/\/+$/, "") || "/").toLowerCase();
  if (p === "/quest" || p.endsWith("/quest") || p.includes("/quest/") || p === "/agent" || p.endsWith("/agent") || p.includes("/agent/")) {
    return "agent";
  }
  if (p === "/memory" || p.endsWith("/memory") || p.includes("/memory/") || p === "/knowledge" || p.endsWith("/knowledge") || p.includes("/knowledge/")) {
    return "memory";
  }
  if (
    p === "/reviews" ||
    p.endsWith("/reviews") ||
    p.includes("/reviews/") ||
    p.includes("/m/reviews") ||
    p === "/metrics" ||
    p.endsWith("/metrics") ||
    p.includes("/metrics/") ||
    p === "/observability" ||
    p.endsWith("/observability") ||
    p.includes("/observability/")
  ) {
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
    if (v === "agent" || v === "memory" || v === "review") return v;
  } catch {
    /* ignore */
  }
  return null;
}

import type { PermissionMode } from "./api/session.api";

export const PERMISSION_OPTIONS: { value: PermissionMode; label: string }[] = [
  { value: "read-only", label: "询问审批" },
  { value: "workspace-write", label: "自动审批" },
  { value: "full", label: "完全访问" },
];

const LABEL_BY_VALUE = Object.fromEntries(
  PERMISSION_OPTIONS.map((o) => [o.value, o.label]),
) as Record<PermissionMode, string>;

/** Short strip text aligned with prototype chip labels (审批：询问). */
export const PERMISSION_STRIP_LABEL: Record<PermissionMode, string> = {
  "read-only": "询问",
  "workspace-write": "自动",
  full: "完全访问",
};

export const FULL_ACCESS_CONFIRM_MESSAGE =
  "切换到「完全访问」？此模式非 ASH 默认。危险工具可跳过逐步批准，仍受场景策略约束。确认继续？";

export function resolvePermissionMode(raw?: string | null): PermissionMode {
  if (raw === "workspace-write" || raw === "full") return raw;
  return "read-only";
}

export function permissionOptionLabel(mode: PermissionMode): string {
  return LABEL_BY_VALUE[mode] ?? LABEL_BY_VALUE["read-only"];
}

export function permissionStripLabel(mode: PermissionMode): string {
  return PERMISSION_STRIP_LABEL[mode] ?? PERMISSION_STRIP_LABEL["read-only"];
}

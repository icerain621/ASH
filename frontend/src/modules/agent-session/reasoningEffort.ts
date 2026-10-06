export const REASONING_EFFORTS = [
  { id: "low", label: "Low" },
  { id: "medium", label: "Medium" },
  { id: "high", label: "High" },
  { id: "max", label: "Max" },
] as const;

export type ReasoningEffortId = (typeof REASONING_EFFORTS)[number]["id"];

export const DEFAULT_REASONING_EFFORT: ReasoningEffortId = "high";

const storageKey = (sessionId: string) => `ash.agentChat.reasoningEffort.${sessionId}`;

export function isReasoningEffortId(raw: string): raw is ReasoningEffortId {
  return REASONING_EFFORTS.some((item) => item.id === raw);
}

export function reasoningEffortLabel(id: ReasoningEffortId): string {
  return REASONING_EFFORTS.find((item) => item.id === id)?.label ?? id;
}

/** Display label for an event-payload effort id; unknown values stay off the projection. */
export function reasoningEffortCaption(raw: string | null | undefined): string {
  const id = (raw ?? "").trim();
  if (!isReasoningEffortId(id)) return "";
  return reasoningEffortLabel(id);
}

export function readReasoningEffort(sessionId: string | null | undefined): ReasoningEffortId {
  if (!sessionId) return DEFAULT_REASONING_EFFORT;
  try {
    const raw = localStorage.getItem(storageKey(sessionId));
    if (raw && isReasoningEffortId(raw)) return raw;
  } catch {
    /* quota / private mode */
  }
  return DEFAULT_REASONING_EFFORT;
}

export function writeReasoningEffort(sessionId: string, effort: ReasoningEffortId): void {
  try {
    localStorage.setItem(storageKey(sessionId), effort);
  } catch {
    /* ignore */
  }
}

export const COMPOSER_PLACEHOLDER_DEFAULT =
  "发消息或创建任务，/ 调用指令";

export const COMPOSER_PLACEHOLDER_STEER =
  "打断并续写，Enter 续写，Alt+Enter 排队";

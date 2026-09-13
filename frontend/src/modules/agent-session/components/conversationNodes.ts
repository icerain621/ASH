import type { SessionEventEnvelope } from "../api/session.api";
import { eventVisibility } from "../api/session.api";

export type ConversationNodeKind = "session.turn" | "gate.waiting_approval" | "session.intent" | "default";

export type ConversationNodeDescriptor = {
  kind: ConversationNodeKind;
  title: string;
  summary: string;
  visibility: string;
};

/** Events shown in the thin thread (hide audit-only). */
export function isThreadVisibleEvent(ev: SessionEventEnvelope): boolean {
  return eventVisibility(ev) !== "audit";
}

function payloadRecord(payload: unknown): Record<string, unknown> {
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    return payload as Record<string, unknown>;
  }
  if (typeof payload === "string") {
    try {
      const parsed = JSON.parse(payload) as unknown;
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>;
      }
    } catch {
      /* ignore */
    }
  }
  return {};
}

function str(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

/** Registry: map event type → thin ConversationNode descriptor (no private scoring). */
export function resolveConversationNode(ev: SessionEventEnvelope): ConversationNodeDescriptor {
  const visibility = eventVisibility(ev);
  const p = payloadRecord(ev.payload);
  switch (ev.type) {
    case "session.turn":
      return {
        kind: "session.turn",
        title: "用户意图",
        summary: str(p.prompt) || "(empty prompt)",
        visibility,
      };
    case "gate.waiting_approval":
      return {
        kind: "gate.waiting_approval",
        title: "等待审批",
        summary: str(p.reason) || str(p.gate) || "waiting_approval",
        visibility,
      };
    case "session.intent":
      return {
        kind: "session.intent",
        title: `意图 · ${str(p.action) || "—"}`,
        summary: str(p.reason) || str(p.actorId) || ev.type,
        visibility,
      };
    default:
      return {
        kind: "default",
        title: ev.type,
        summary: Object.keys(p).length ? JSON.stringify(p).slice(0, 160) : ev.severity || "",
        visibility,
      };
  }
}

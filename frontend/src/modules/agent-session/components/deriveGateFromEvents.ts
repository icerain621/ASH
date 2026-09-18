import type { SessionEventEnvelope } from "../api/session.api";

function payloadReason(payload: unknown): string {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return "";
  const p = payload as Record<string, unknown>;
  const v = p.reason ?? p.gate ?? p.message;
  return typeof v === "string" ? v : "";
}

function payloadTool(payload: unknown): string {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return "";
  const p = payload as Record<string, unknown>;
  return typeof p.tool === "string" ? p.tool.trim() : "";
}

/** Latest open gate from session events (DSH: approval stays in chat column). */
export function deriveGateFromEvents(events: SessionEventEnvelope[]): {
  waiting: boolean;
  reason: string;
  tool: string;
} {
  let waiting = false;
  let reason = "";
  let tool = "";
  for (const ev of events) {
    const type = ev.type || "";
    if (type === "gate.waiting_approval") {
      waiting = true;
      reason = payloadReason(ev.payload) || "waiting_approval";
      tool = payloadTool(ev.payload) || tool;
      continue;
    }
    if (
      type === "gate.approved" ||
      type === "gate.rejected" ||
      type === "gate.decision" ||
      type === "run.finished" ||
      type === "run.failed" ||
      type === "run.canceled"
    ) {
      waiting = false;
      continue;
    }
    if (type === "session.intent") {
      const p =
        ev.payload && typeof ev.payload === "object" && !Array.isArray(ev.payload)
          ? (ev.payload as Record<string, unknown>)
          : {};
      const action = typeof p.action === "string" ? p.action.toLowerCase() : "";
      if (
        action === "approve" ||
        action === "allow_once" ||
        action === "allow_session" ||
        action === "reject" ||
        action === "deny" ||
        action === "cancel" ||
        action === "cancel_run"
      ) {
        waiting = false;
      }
    }
  }
  return { waiting, reason, tool };
}

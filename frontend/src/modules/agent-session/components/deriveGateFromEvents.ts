import type { SessionEventEnvelope } from "../api/session.api";

function payloadReason(payload: unknown): string {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return "";
  const p = payload as Record<string, unknown>;
  const v = p.reason ?? p.gate ?? p.message;
  return typeof v === "string" ? v : "";
}

/** Latest open gate from session events (DSH: approval stays in chat column). */
export function deriveGateFromEvents(events: SessionEventEnvelope[]): {
  waiting: boolean;
  reason: string;
} {
  let waiting = false;
  let reason = "";
  for (const ev of events) {
    const type = ev.type || "";
    if (type === "gate.waiting_approval") {
      waiting = true;
      reason = payloadReason(ev.payload) || "waiting_approval";
      continue;
    }
    if (
      type === "gate.approved" ||
      type === "gate.rejected" ||
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
      if (action === "approve" || action === "reject" || action === "cancel") {
        waiting = false;
      }
    }
  }
  return { waiting, reason };
}

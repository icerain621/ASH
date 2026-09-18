import { describe, expect, it } from "vitest";
import type { SessionEventEnvelope } from "../api/session.api";
import { deriveGateFromEvents } from "./deriveGateFromEvents";

function ev(partial: Partial<SessionEventEnvelope> & { type: string }): SessionEventEnvelope {
  return {
    id: partial.id ?? "evt",
    runId: partial.runId ?? "run_1",
    seq: partial.seq ?? 1,
    ts: partial.ts ?? 1,
    severity: partial.severity ?? "info",
    type: partial.type,
    payload: partial.payload,
  };
}

describe("deriveGateFromEvents", () => {
  it("opens on gate.waiting_approval and closes on approve intent", () => {
    expect(
      deriveGateFromEvents([
        ev({ type: "gate.waiting_approval", payload: { reason: "need review", tool: "bash" } }),
      ]),
    ).toEqual({ waiting: true, reason: "need review", tool: "bash" });

    expect(
      deriveGateFromEvents([
        ev({ seq: 1, type: "gate.waiting_approval", payload: { reason: "need review", tool: "bash" } }),
        ev({ seq: 2, type: "session.intent", payload: { action: "approve", scope: "once" } }),
      ]),
    ).toEqual({ waiting: false, reason: "need review", tool: "bash" });
  });

  it("closes on gate.decision and allow_session intent", () => {
    expect(
      deriveGateFromEvents([
        ev({ seq: 1, type: "gate.waiting_approval", payload: { reason: "tool gate", tool: "bash" } }),
        ev({ seq: 2, type: "gate.decision", payload: { scope: "session", tool: "bash" } }),
      ]),
    ).toEqual({ waiting: false, reason: "tool gate", tool: "bash" });
  });
});

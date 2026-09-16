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
        ev({ type: "gate.waiting_approval", payload: { reason: "need review" } }),
      ]),
    ).toEqual({ waiting: true, reason: "need review" });

    expect(
      deriveGateFromEvents([
        ev({ seq: 1, type: "gate.waiting_approval", payload: { reason: "need review" } }),
        ev({ seq: 2, type: "session.intent", payload: { action: "approve" } }),
      ]),
    ).toEqual({ waiting: false, reason: "need review" });
  });
});

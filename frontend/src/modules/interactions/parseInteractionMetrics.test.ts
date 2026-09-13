import { describe, expect, it } from "vitest";
import { parseInteractionMetrics } from "./parseInteractionMetrics";

describe("parseInteractionMetrics", () => {
  it("parses seal / mismatch / memory_link counters", () => {
    const text = `
# HELP
ash_interaction_thread_sealed_total 3
ash_interaction_replay_mismatch_total 1
ash_memory_link_total{type="hit_used"} 4
ash_memory_link_total{type="context_ref"} 2
ash_run_total{status="started"} 9
`;
    const s = parseInteractionMetrics(text);
    expect(s.sealedTotal).toBe(3);
    expect(s.replayMismatchTotal).toBe(1);
    expect(s.memoryLinkTotal).toBe(6);
    expect(s.memoryLinksByType).toEqual({ hit_used: 4, context_ref: 2 });
  });

  it("returns zeros for empty text", () => {
    expect(parseInteractionMetrics("")).toEqual({
      sealedTotal: 0,
      replayMismatchTotal: 0,
      memoryLinksByType: {},
      memoryLinkTotal: 0,
    });
  });
});

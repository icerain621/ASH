import { describe, expect, it } from "vitest";
import { isTemplateBindable, type AgentTemplateEntry, type MemoryEvalCounts } from "./agentCatalog.api";

describe("isTemplateBindable", () => {
  it("allows approved templates only", () => {
    const approved: AgentTemplateEntry = {
      status: "approved",
      manifest: { id: "tpl.react", version: "1.0.0" },
    };
    const candidate: AgentTemplateEntry = {
      status: "candidate",
      manifest: { id: "tpl.custom", version: "1.0.0" },
    };
    expect(isTemplateBindable(approved)).toBe(true);
    expect(isTemplateBindable(candidate)).toBe(false);
  });
});

describe("MemoryEvalCounts shape", () => {
  it("carries three F05 counters", () => {
    const counts: MemoryEvalCounts = { remembered: 2, usedCorrectly: 1, wrongUpdate: 0 };
    expect(counts.remembered + counts.usedCorrectly + counts.wrongUpdate).toBe(3);
  });
});

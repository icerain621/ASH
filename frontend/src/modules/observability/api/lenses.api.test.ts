import { describe, expect, it } from "vitest";
import { observeDeepLink, packAccessLabel } from "./lenses.api";

describe("observeDeepLink", () => {
  it("keeps stable deep-link ids for H06", () => {
    expect(observeDeepLink("global")).toBe("/ui/observe?lens=global");
    expect(observeDeepLink("agent", { run: "run_x" })).toBe("/ui/observe?lens=agent&run=run_x");
    expect(observeDeepLink("memory", { id: "mem_y" })).toBe("/ui/observe?lens=memory&id=mem_y");
  });
});

describe("packAccessLabel", () => {
  it("marks missing pack as 未接入 (I03)", () => {
    expect(packAccessLabel("missing")).toBe("未接入");
    expect(packAccessLabel(undefined)).toBe("未接入");
    expect(packAccessLabel("ready")).toBe("已接入");
  });
});

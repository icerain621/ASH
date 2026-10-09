import { describe, expect, it } from "vitest";
import { observeDeepLink } from "./lenses.api";

describe("observeDeepLink", () => {
  it("keeps stable deep-link ids for H06", () => {
    expect(observeDeepLink("global")).toBe("/ui/observe?lens=global");
    expect(observeDeepLink("agent", { run: "run_x" })).toBe("/ui/observe?lens=agent&run=run_x");
    expect(observeDeepLink("memory", { id: "mem_y" })).toBe("/ui/observe?lens=memory&id=mem_y");
  });
});

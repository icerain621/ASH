import { describe, expect, it } from "vitest";
import { workModeFromPath } from "./workMode";

describe("workModeFromPath", () => {
  it("maps quest to agent", () => {
    expect(workModeFromPath("/quest")).toBe("agent");
    expect(workModeFromPath("/ui/quest")).toBe("agent");
  });
  it("maps reviews to review", () => {
    expect(workModeFromPath("/reviews")).toBe("review");
    expect(workModeFromPath("/m/reviews")).toBe("review");
  });
  it("returns null for other routes", () => {
    expect(workModeFromPath("/runs")).toBeNull();
    expect(workModeFromPath("/memory")).toBeNull();
  });
});

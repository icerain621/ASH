import { describe, expect, it } from "vitest";
import { eventVisibility } from "./session.api";

describe("eventVisibility", () => {
  it("defaults session.turn to model_visible", () => {
    expect(eventVisibility({ type: "session.turn" })).toBe("model_visible");
  });
  it("respects explicit ui_only", () => {
    expect(eventVisibility({ type: "session.turn", visibility: "ui_only" })).toBe("ui_only");
  });
  it("maps gate.waiting_approval", () => {
    expect(eventVisibility({ type: "gate.waiting_approval" })).toBe("ui_only");
  });
});

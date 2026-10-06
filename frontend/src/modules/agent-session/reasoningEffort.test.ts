import { describe, expect, it, vi } from "vitest";
import {
  COMPOSER_PLACEHOLDER_DEFAULT,
  DEFAULT_REASONING_EFFORT,
  readReasoningEffort,
  reasoningEffortLabel,
  reasoningEffortCaption,
  writeReasoningEffort,
} from "./reasoningEffort";

describe("reasoningEffort", () => {
  it("captions known effort ids for projection", () => {
    expect(reasoningEffortCaption("max")).toBe("Max");
    expect(reasoningEffortCaption("high")).toBe("High");
    expect(reasoningEffortCaption("")).toBe("");
    expect(reasoningEffortCaption("nope")).toBe("");
  });

  it("defaults to High and round-trips per session", () => {
    const store = new Map<string, string>();
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => {
        store.set(key, value);
      },
    });
    expect(readReasoningEffort("sess_a")).toBe(DEFAULT_REASONING_EFFORT);
    expect(reasoningEffortLabel(DEFAULT_REASONING_EFFORT)).toBe("High");
    writeReasoningEffort("sess_a", "max");
    expect(readReasoningEffort("sess_a")).toBe("max");
    expect(readReasoningEffort("sess_b")).toBe("high");
    expect(COMPOSER_PLACEHOLDER_DEFAULT).toMatch(/发消息或创建任务/);
    vi.unstubAllGlobals();
  });
});

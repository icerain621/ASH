import { afterEach, describe, expect, it } from "vitest";
import {
  WORK_MODE_STORAGE_KEY,
  persistWorkMode,
  readPersistedWorkMode,
  workModeFromPath,
} from "./workMode";

describe("workModeFromPath", () => {
  it("maps quest and agent to agent", () => {
    expect(workModeFromPath("/quest")).toBe("agent");
    expect(workModeFromPath("/ui/quest")).toBe("agent");
    expect(workModeFromPath("/agent")).toBe("agent");
  });

  it("maps memory and knowledge to memory", () => {
    expect(workModeFromPath("/memory")).toBe("memory");
    expect(workModeFromPath("/knowledge")).toBe("memory");
  });

  it("maps reviews and observability paths to review", () => {
    expect(workModeFromPath("/reviews")).toBe("review");
    expect(workModeFromPath("/m/reviews")).toBe("review");
    expect(workModeFromPath("/metrics")).toBe("review");
    expect(workModeFromPath("/observability")).toBe("review");
  });

  it("returns null for more/account routes", () => {
    expect(workModeFromPath("/runs")).toBeNull();
    expect(workModeFromPath("/automation")).toBeNull();
    expect(workModeFromPath("/space")).toBeNull();
    expect(workModeFromPath("/login")).toBeNull();
  });
});

describe("persistWorkMode / readPersistedWorkMode", () => {
  afterEach(() => {
    localStorage.removeItem(WORK_MODE_STORAGE_KEY);
  });

  it("persists and reads three modes", () => {
    persistWorkMode("agent");
    expect(readPersistedWorkMode()).toBe("agent");
    persistWorkMode("memory");
    expect(readPersistedWorkMode()).toBe("memory");
    persistWorkMode("review");
    expect(readPersistedWorkMode()).toBe("review");
  });

  it("ignores invalid stored values", () => {
    localStorage.setItem(WORK_MODE_STORAGE_KEY, "ops");
    expect(readPersistedWorkMode()).toBeNull();
  });
});

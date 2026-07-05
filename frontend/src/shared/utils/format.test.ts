import { describe, expect, it } from "vitest";
import { fmtTime, shortId } from "./format";

describe("fmtTime", () => {
  it("returns em dash for empty values", () => {
    expect(fmtTime()).toBe("—");
    expect(fmtTime(null)).toBe("—");
    expect(fmtTime(0)).toBe("—");
  });

  it("formats epoch ms", () => {
    const text = fmtTime(1_700_000_000_000);
    expect(text).not.toBe("—");
    expect(text).toMatch(/2023/);
  });
});

describe("shortId", () => {
  it("returns empty for missing id", () => {
    expect(shortId()).toBe("");
  });

  it("truncates long ids", () => {
    const id = "run-0123456789abcdef";
    expect(shortId(id)).toBe("run-0123456789…");
  });

  it("keeps short ids unchanged", () => {
    expect(shortId("run-1")).toBe("run-1");
  });
});

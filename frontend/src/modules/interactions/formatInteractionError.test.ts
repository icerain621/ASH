import { describe, expect, it } from "vitest";
import { formatInteractionError } from "./formatInteractionError";

describe("formatInteractionError", () => {
  it("localizes run / thread not found", () => {
    expect(formatInteractionError("run not found")).toBe("未找到该 Run，请检查 Run ID");
    expect(formatInteractionError("Run Not Found")).toBe("未找到该 Run，请检查 Run ID");
    expect(formatInteractionError("thread not found")).toBe("未找到该 Thread");
  });

  it("passes through other messages", () => {
    expect(formatInteractionError("permission denied")).toBe("permission denied");
    expect(formatInteractionError("")).toBe("");
  });
});

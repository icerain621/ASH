import { describe, expect, it } from "vitest";
import { reorderSessionIds } from "./reorderSessionIds";

describe("reorderSessionIds", () => {
  it("moves an id before another", () => {
    expect(reorderSessionIds(["a", "b", "c"], "c", "a")).toEqual(["c", "a", "b"]);
  });

  it("is a no-op when ids missing", () => {
    expect(reorderSessionIds(["a", "b"], "x", "a")).toEqual(["a", "b"]);
  });
});

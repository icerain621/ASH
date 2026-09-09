import { beforeEach, describe, expect, it } from "vitest";
import { ApiError, clearAuthSession, getCurrentSpaceId, setAuthSession } from "./client";

describe("ApiError", () => {
  it("exposes code and message", () => {
    const err = new ApiError("SPACE_ACCESS_DENIED", "forbidden");
    expect(err.code).toBe("SPACE_ACCESS_DENIED");
    expect(err.message).toBe("forbidden");
  });
});

describe("auth session", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("defaults space to local", () => {
    expect(getCurrentSpaceId()).toBe("local");
  });

  it("persists token and space", () => {
    setAuthSession("tok-abc", "team-a");
    expect(localStorage.getItem("ash.auth.token")).toBe("tok-abc");
    expect(getCurrentSpaceId()).toBe("team-a");
  });

  it("clears auth session", () => {
    setAuthSession("tok-abc", "team-a", "rt");
    clearAuthSession();
    expect(localStorage.getItem("ash.auth.token")).toBeNull();
    expect(localStorage.getItem("ash.auth.refresh")).toBeNull();
    expect(getCurrentSpaceId()).toBe("local");
  });
});

import { describe, expect, it } from "vitest";
import { oidcLoginHref, parseLoginHash } from "@/modules/platform/auth/loginHash";

describe("parseLoginHash", () => {
  it("reads token and spaceId from hash", () => {
    expect(parseLoginHash("#token=abc&spaceId=space_1")).toEqual({ token: "abc", spaceId: "space_1" });
  });

  it("defaults spaceId to local", () => {
    expect(parseLoginHash("token=xyz")).toEqual({ token: "xyz", spaceId: "local" });
  });

  it("returns null without token", () => {
    expect(parseLoginHash("#spaceId=only")).toBeNull();
    expect(parseLoginHash("")).toBeNull();
  });
});

describe("oidcLoginHref", () => {
  it("includes ui=1", () => {
    expect(oidcLoginHref()).toBe("/api/v1/auth/oidc/login?ui=1");
  });
});

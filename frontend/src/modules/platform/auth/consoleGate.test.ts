import { beforeEach, describe, expect, it } from "vitest";
import {
  isAuthFailureCode,
  isConsoleAuthRequiredFlag,
  resetConsoleAuthGateCache,
  shouldRedirectToLogin,
} from "./consoleGate";

describe("consoleGate", () => {
  beforeEach(() => {
    resetConsoleAuthGateCache();
  });

  it("reads consoleAuthRequired from readyz", () => {
    expect(isConsoleAuthRequiredFlag({})).toBe(false);
    expect(isConsoleAuthRequiredFlag({ consoleAuthRequired: true })).toBe(true);
  });

  it("redirects when gate on, no token, not login", () => {
    expect(
      shouldRedirectToLogin({ consoleAuthRequired: true, token: "", pathname: "/runs" }),
    ).toBe(true);
    expect(
      shouldRedirectToLogin({ consoleAuthRequired: true, token: "", pathname: "/login" }),
    ).toBe(false);
    expect(
      shouldRedirectToLogin({ consoleAuthRequired: true, token: "t", pathname: "/runs" }),
    ).toBe(false);
    expect(
      shouldRedirectToLogin({ consoleAuthRequired: false, token: "", pathname: "/runs" }),
    ).toBe(false);
  });

  it("classifies auth failure codes", () => {
    expect(isAuthFailureCode("UNAUTHORIZED")).toBe(true);
    expect(isAuthFailureCode("AUTH_TOKEN_REPLAY")).toBe(true);
    expect(isAuthFailureCode("SPACE_ACCESS_DENIED")).toBe(false);
  });
});

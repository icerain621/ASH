import { describe, expect, it } from "vitest";
import {
  agentModeTagline,
  ASH_CHAT_BG_URL,
  resolveAgentMode,
} from "./agentModeLabels";

describe("agentModeLabels", () => {
  it("defaults to coding", () => {
    expect(resolveAgentMode(undefined)).toBe("coding");
    expect(resolveAgentMode("nope")).toBe("coding");
  });

  it("tagline follows mode", () => {
    expect(agentModeTagline("coding")).toMatch(/交付编排/);
    expect(agentModeTagline("general")).toMatch(/通用助手/);
  });

  it("uses vite base for bg asset", () => {
    expect(ASH_CHAT_BG_URL).toMatch(/\/ui\/ash-chat-bg\.png$/);
  });
});

import { describe, expect, it } from "vitest";
import { projectHooksFromBodyJson } from "./hooksPolicy";

describe("projectHooksFromBodyJson", () => {
  it("returns null when hooks key is absent", () => {
    expect(projectHooksFromBodyJson('{"toolApprovalPresets":{}}')).toBeNull();
  });

  it("projects rules from bodyJson.hooks", () => {
    const body = JSON.stringify({
      hooks: {
        version: "ash.hooks.v1",
        rules: [{ event: "PreToolUse", tool: "bash", action: "deny", reason: "blocked" }],
      },
    });
    const proj = projectHooksFromBodyJson(body);
    expect(proj?.version).toBe("ash.hooks.v1");
    expect(proj?.rules).toHaveLength(1);
    expect(proj?.rules[0]?.action).toBe("deny");
  });
});

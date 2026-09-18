export type HooksRuleView = {
  event?: string;
  tool?: string;
  risk?: string;
  action?: string;
  reason?: string;
};

export type SpaceHooksProjection = {
  version?: string;
  rules: HooksRuleView[];
  parseError?: string;
};

/** Read-only projection of SpacePolicy bodyJson.hooks (ash.hooks.v1). */
export function projectHooksFromBodyJson(bodyJson?: string): SpaceHooksProjection | null {
  const raw = (bodyJson || "").trim();
  if (!raw) return null;
  try {
    const root = JSON.parse(raw) as unknown;
    if (!root || typeof root !== "object" || Array.isArray(root)) {
      return { rules: [], parseError: "bodyJson 非对象" };
    }
    const hooks = (root as Record<string, unknown>).hooks;
    if (hooks == null) return null;
    if (typeof hooks !== "object" || Array.isArray(hooks)) {
      return { rules: [], parseError: "hooks 非对象" };
    }
    const h = hooks as Record<string, unknown>;
    const version = typeof h.version === "string" ? h.version : undefined;
    const rulesRaw = h.rules;
    if (!Array.isArray(rulesRaw)) {
      return { version, rules: [], parseError: "rules 非数组" };
    }
    const rules: HooksRuleView[] = rulesRaw.map((item) => {
      if (!item || typeof item !== "object" || Array.isArray(item)) return {};
      const r = item as Record<string, unknown>;
      return {
        event: typeof r.event === "string" ? r.event : undefined,
        tool: typeof r.tool === "string" ? r.tool : undefined,
        risk: typeof r.risk === "string" ? r.risk : undefined,
        action: typeof r.action === "string" ? r.action : undefined,
        reason: typeof r.reason === "string" ? r.reason : undefined,
      };
    });
    return { version, rules };
  } catch {
    return { rules: [], parseError: "bodyJson 解析失败" };
  }
}

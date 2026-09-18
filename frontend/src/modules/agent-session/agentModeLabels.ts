export type AgentMode = "coding" | "general";

export const AGENT_MODE_OPTIONS: { value: AgentMode; label: string }[] = [
  { value: "coding", label: "⟨/⟩ 编程" },
  { value: "general", label: "◎ 通用" },
];

const TAGLINE: Record<AgentMode, string> = {
  coding: "交付编排从这里开始 · 可审计 · 双核治理",
  general: "通用助手模式 · 仍受 SpacePolicy / 审批门禁约束",
};

export function resolveAgentMode(raw?: string | null): AgentMode {
  return raw === "general" ? "general" : "coding";
}

export function agentModeTagline(mode: AgentMode): string {
  return TAGLINE[mode];
}

/** Public asset URLs (Vite base `/ui/`). */
export const ASH_CHAT_BG_URL = `${import.meta.env.BASE_URL}ash-chat-bg.png`;
export const ASH_ICON_URL = `${import.meta.env.BASE_URL}ash-icon.png`;

export const EMPTY_HERO_DISMISS_KEY = "ash.agent.emptyHero.dismissed";

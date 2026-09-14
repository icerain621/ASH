import type { SessionEventEnvelope } from "../api/session.api";
import { eventVisibility } from "../api/session.api";

export type ConversationNodeKind =
  | "session.turn"
  | "assistant"
  | "gate.waiting_approval"
  | "session.intent"
  | "tool.called"
  | "tool.result"
  | "tool"
  | "step"
  | "default";

export type ConversationNodeDescriptor = {
  kind: ConversationNodeKind;
  title: string;
  summary: string;
  visibility: string;
};

/** Events shown in the thin thread (hide audit-only). */
export function isThreadVisibleEvent(ev: SessionEventEnvelope): boolean {
  return eventVisibility(ev) !== "audit";
}

function payloadRecord(payload: unknown): Record<string, unknown> {
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    return payload as Record<string, unknown>;
  }
  if (typeof payload === "string") {
    try {
      const parsed = JSON.parse(payload) as unknown;
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>;
      }
    } catch {
      /* ignore */
    }
  }
  return {};
}

function str(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

function compactJSON(p: Record<string, unknown>, max = 160): string {
  if (!Object.keys(p).length) return "";
  try {
    return JSON.stringify(p).slice(0, max);
  } catch {
    return "";
  }
}

/** Registry: map event type → thin ConversationNode descriptor (no private scoring). */
export function resolveConversationNode(ev: SessionEventEnvelope): ConversationNodeDescriptor {
  const visibility = eventVisibility(ev);
  const p = payloadRecord(ev.payload);
  const type = ev.type || "";

  switch (type) {
    case "session.turn":
      return {
        kind: "session.turn",
        title: "用户意图",
        summary: str(p.prompt) || "(empty prompt)",
        visibility,
      };
    case "assistant.delta":
    case "assistant.message":
      return {
        kind: "assistant",
        title: "助手",
        summary: str(p.text) || "",
        visibility,
      };
    case "gate.waiting_approval":
      return {
        kind: "gate.waiting_approval",
        title: "等待审批",
        summary: str(p.reason) || str(p.gate) || "waiting_approval",
        visibility,
      };
    case "session.intent":
      return {
        kind: "session.intent",
        title: `意图 · ${str(p.action) || "—"}`,
        summary: str(p.reason) || str(p.actorId) || type,
        visibility,
      };
    case "tool.called":
      return {
        kind: "tool.called",
        title: `工具调用 · ${str(p.name) || str(p.tool) || str(p.toolName) || "tool"}`,
        summary: str(p.input) || str(p.args) || compactJSON(p) || type,
        visibility,
      };
    case "tool.result":
      return {
        kind: "tool.result",
        title: `工具结果 · ${str(p.name) || str(p.tool) || str(p.toolName) || "tool"}`,
        summary: str(p.output) || str(p.result) || str(p.error) || compactJSON(p) || type,
        visibility,
      };
    default:
      break;
  }

  if (type.startsWith("tool.")) {
    return {
      kind: "tool",
      title: `工具 · ${type.replace(/^tool\./, "")}`,
      summary: str(p.name) || str(p.message) || compactJSON(p) || type,
      visibility,
    };
  }
  if (type.startsWith("step.")) {
    const phase = type.replace(/^step\./, "");
    return {
      kind: "step",
      title: `步骤 · ${phase}`,
      summary: str(p.name) || str(p.step) || str(p.message) || compactJSON(p) || type,
      visibility,
    };
  }

  return {
    kind: "default",
    title: type,
    summary: Object.keys(p).length ? compactJSON(p) : ev.severity || "",
    visibility,
  };
}

export type ChatBubbleRole = "user" | "assistant" | "gate" | "tool" | "other";

export type MergedChatBubble = {
  id: string;
  role: ChatBubbleRole;
  type: string;
  kind: string;
  title: string;
  summary: string;
  streaming?: boolean;
  turnId?: string;
  payload?: unknown;
};

function bubbleRole(type: string, kind: string): ChatBubbleRole {
  if (type === "session.turn" || kind === "session.turn") return "user";
  if (type === "assistant.delta" || type === "assistant.message" || kind === "assistant") {
    return "assistant";
  }
  if (type === "gate.waiting_approval" || kind === "gate.waiting_approval") return "gate";
  if (
    kind === "tool.called" ||
    kind === "tool.result" ||
    kind === "tool" ||
    kind === "step" ||
    type.startsWith("tool.") ||
    type.startsWith("step.")
  ) {
    return "tool";
  }
  return "other";
}

/**
 * Merge assistant.delta by turnId into one live bubble; assistant.message replaces/completes it.
 * Other visible events become their own bubbles.
 */
export function mergeAssistantBubbles(events: SessionEventEnvelope[]): MergedChatBubble[] {
  const visible = events.filter(isThreadVisibleEvent);
  const out: MergedChatBubble[] = [];
  /** Index of the open streaming assistant bubble for a turnId. */
  const streamingIdx = new Map<string, number>();

  for (const item of visible) {
    const type = item.type || "";
    const node = resolveConversationNode(item);
    const p = payloadRecord(item.payload);
    const turnId = str(p.turnId);
    const id = item.id || `${item.seq}-${item.type}`;

    if (type === "assistant.delta") {
      const text = str(p.text);
      const existing = turnId ? streamingIdx.get(turnId) : undefined;
      if (existing != null && out[existing]) {
        out[existing] = {
          ...out[existing],
          summary: out[existing].summary + text,
          streaming: true,
          type: "assistant.delta",
          payload: item.payload,
        };
      } else {
        const idx = out.length;
        out.push({
          id: turnId ? `assistant:${turnId}` : id,
          role: "assistant",
          type: "assistant.delta",
          kind: "assistant",
          title: "助手",
          summary: text,
          streaming: true,
          turnId: turnId || undefined,
          payload: item.payload,
        });
        if (turnId) streamingIdx.set(turnId, idx);
      }
      continue;
    }

    if (type === "assistant.message") {
      const text = str(p.text);
      const existing = turnId ? streamingIdx.get(turnId) : undefined;
      const bubble: MergedChatBubble = {
        id: turnId ? `assistant:${turnId}` : id,
        role: "assistant",
        type: "assistant.message",
        kind: "assistant",
        title: "助手",
        summary: text,
        streaming: false,
        turnId: turnId || undefined,
        payload: item.payload,
      };
      if (existing != null && out[existing]) {
        out[existing] = bubble;
        streamingIdx.delete(turnId);
      } else {
        out.push(bubble);
      }
      continue;
    }

    out.push({
      id,
      role: bubbleRole(type, node.kind),
      type,
      kind: node.kind,
      title: node.title,
      summary: node.summary,
      payload: item.payload,
    });
  }

  return out;
}

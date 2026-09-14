import { api } from "@/services/http/client";

/** GV01 thin agent session client (FE stub; ConversationNode UI lands GV02). */

export type EventVisibility = "model_visible" | "ui_only" | "audit" | string;

export type SessionEventEnvelope = {
  id: string;
  traceId?: string;
  runId: string;
  seq: number;
  ts: number;
  type: string;
  severity: string;
  visibility?: EventVisibility;
  payload?: unknown;
};

export type AgentSessionView = {
  id: string;
  spaceId: string;
  status: string;
  title?: string;
  goal?: string;
  planId?: string;
  runId?: string;
  streamUrl?: string;
  workspaceId?: string;
  meta?: Record<string, unknown>;
  turns?: Array<{ id: string; prompt: string; createdAt: number }>;
  replies?: Array<{
    turnId: string;
    text: string;
    source?: string;
    stopped?: boolean;
    chunks?: string[];
    createdAt?: number;
  }>;
  createdAt?: number;
  updatedAt?: number;
};

export type SessionIntentAction = "prompt" | "approve" | "cancel" | "stop" | "reject";

export type AgentSessionListResponse = {
  items: AgentSessionView[];
};

export async function listAgentSessions(opts?: {
  spaceId?: string;
  limit?: number;
  includeClosed?: boolean;
}): Promise<AgentSessionListResponse> {
  const q = new URLSearchParams();
  if (opts?.spaceId) q.set("spaceId", opts.spaceId);
  if (opts?.limit != null) q.set("limit", String(opts.limit));
  if (opts?.includeClosed) q.set("includeClosed", "1");
  const suffix = q.toString() ? `?${q}` : "";
  return api<AgentSessionListResponse>(`/agents/sessions${suffix}`);
}

export async function createAgentSession(body: {
  runId?: string;
  goal?: string;
  spaceId?: string;
  providerKind?: string;
  workspaceId?: string;
}): Promise<AgentSessionView> {
  return api<AgentSessionView>("/agents/sessions", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export async function getAgentSession(sessionId: string): Promise<AgentSessionView> {
  return api<AgentSessionView>(`/agents/sessions/${encodeURIComponent(sessionId)}`);
}

export async function patchAgentSession(
  sessionId: string,
  body: { title: string },
): Promise<AgentSessionView> {
  return api<AgentSessionView>(`/agents/sessions/${encodeURIComponent(sessionId)}`, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export async function closeAgentSession(sessionId: string): Promise<AgentSessionView> {
  return api<AgentSessionView>(`/agents/sessions/${encodeURIComponent(sessionId)}`, {
    method: "DELETE",
  });
}

export async function submitSessionIntent(
  sessionId: string,
  body: { action: SessionIntentAction; prompt?: string; reason?: string; actorId?: string },
): Promise<AgentSessionView> {
  return api<AgentSessionView>(`/agents/sessions/${encodeURIComponent(sessionId)}/actions`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export async function listSessionEvents(
  sessionId: string,
  opts?: { afterSeq?: number; limit?: number },
): Promise<{ sessionId: string; runId?: string; streamUrl?: string; items: SessionEventEnvelope[] }> {
  const q = new URLSearchParams();
  if (opts?.afterSeq != null) q.set("afterSeq", String(opts.afterSeq));
  if (opts?.limit != null) q.set("limit", String(opts.limit));
  const suffix = q.toString() ? `?${q}` : "";
  return api(`/agents/sessions/${encodeURIComponent(sessionId)}/events${suffix}`);
}

/** UI helper: treat missing visibility as model_visible (legacy). */
export function eventVisibility(ev: { type?: string; visibility?: string }): EventVisibility {
  const v = (ev.visibility || "").trim();
  if (v === "model_visible" || v === "ui_only" || v === "audit") return v;
  if ((ev.type || "").startsWith("metric.") || (ev.type || "").startsWith("score.") || (ev.type || "").startsWith("audit.")) {
    return "audit";
  }
  if ((ev.type || "").startsWith("ui.") || ev.type === "gate.waiting_approval") return "ui_only";
  if ((ev.type || "").startsWith("tool.") || (ev.type || "").startsWith("step.")) return "ui_only";
  return "model_visible";
}

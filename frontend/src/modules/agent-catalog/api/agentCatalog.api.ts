import { api } from "@/services/http/client";

export type AgentTemplateManifest = {
  id: string;
  version: string;
  loop?: string;
  tools?: string[];
  compaction?: string;
  sandbox?: string;
  maxTurns?: number;
  memory?: string[];
  planMemory?: string[];
};

export type AgentTemplateEntry = {
  status: "approved" | "candidate" | string;
  manifest: AgentTemplateManifest;
};

export type AgentComponent = {
  id: string;
  kind: string;
  status: string;
};

export type AgentReviewItem = {
  id: string;
  kind: string;
  targetId: string;
  status: string;
  reason?: string;
  created?: number;
};

export type PassKResult = {
  k: number;
  target: string;
  allPassed: boolean;
  clusters?: { name: string; count: number }[];
};

export type MemoryEvalCounts = {
  remembered: number;
  usedCorrectly: number;
  wrongUpdate: number;
};

export function listAgentTemplates() {
  return api<{ items: AgentTemplateEntry[] }>("/agent-templates");
}

export function submitAgentTemplateCandidate(body: AgentTemplateManifest) {
  return api<AgentTemplateEntry>("/agent-templates/candidates", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function listAgentComponents() {
  return api<{ items: AgentComponent[] }>("/agent-components");
}

export function registerAgentComponent(body: { id: string; kind: string; status?: string }) {
  return api<AgentComponent>("/agent-components", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function listAgentReviews(kind?: string) {
  const q = kind ? `?kind=${encodeURIComponent(kind)}` : "";
  return api<{ items: AgentReviewItem[] }>(`/agent-reviews${q}`);
}

export function decideAgentReview(reviewId: string, decision: "approve" | "reject", reason?: string) {
  return api<AgentReviewItem>(`/agent-reviews/${encodeURIComponent(reviewId)}/decide`, {
    method: "POST",
    body: JSON.stringify({ decision, reason }),
  });
}

export function runPassK(target: string, k = 2) {
  return api<PassKResult>("/passk/run", {
    method: "POST",
    body: JSON.stringify({ target, k }),
  });
}

export function getMemoryEvalCounts() {
  return api<MemoryEvalCounts>("/memory-eval");
}

/** G03: only approved / built-in templates may bind to a production run. */
export function isTemplateBindable(entry: AgentTemplateEntry): boolean {
  return entry.status === "approved";
}

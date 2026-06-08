import { api } from "@/services/http/client";

export type MemoryEdge = {
  id: string;
  fromId: string;
  toId: string;
  kind: string;
  confidence: number;
  reason?: string;
};

export type GovernanceHint = {
  memoryId: string;
  kind: string;
  title?: string;
  status?: string;
  reason?: string;
};

export type GovernanceHints = {
  duplicates?: GovernanceHint[];
  conflicts?: GovernanceHint[];
};

export type MemoryRecord = {
  id: string;
  layer: string;
  title: string;
  body?: string;
  status: string;
  ttlDays?: number;
  confidence?: number;
  dedupeKey?: string;
  edges?: MemoryEdge[];
  evidence?: { id: string; kind: string; ref: string }[];
};

export function listCandidates(params: {
  limit?: number;
  status?: string;
  expiring?: boolean;
  reviewDue?: boolean;
} = {}) {
  const search = new URLSearchParams();
  search.set("limit", String(params.limit ?? 50));
  if (params.status) search.set("status", params.status);
  if (params.expiring) search.set("expiring", "true");
  if (params.reviewDue) search.set("reviewDue", "true");
  return api<{ items: MemoryRecord[] }>(`/memory/candidates?${search.toString()}`);
}

export function getMemoryRecord(recordId: string) {
  return api<MemoryRecord>(`/memory/records/${recordId}`);
}

export function createCandidate(body: Record<string, unknown>) {
  return api<{ candidateId: string; governance?: GovernanceHints }>("/memory/candidates", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function reviewCandidate(candidateId: string, body: Record<string, unknown>) {
  return api<{ ok: boolean; status: string }>(`/memory/candidates/${candidateId}/review`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function queryMemory(body: { text: string; layers?: string[]; topK?: number; scope?: Record<string, string> }) {
  return api<{ items: MemoryRecord[] }>("/memory/query", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function applyMemoryRetention(body: { dryRun?: boolean }) {
  return api<{ spaceId: string; matched: number; archived: number; reviewRequired: number; decayed: number; dryRun: boolean }>(
    "/memory/retention/apply",
    {
      method: "POST",
      body: JSON.stringify(body),
    },
  );
}

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
  dedupeKey?: string;
  edges?: MemoryEdge[];
  evidence?: { id: string; kind: string; ref: string }[];
};

export function listCandidates(limit = 50) {
  return api<{ items: MemoryRecord[] }>(`/memory/candidates?limit=${limit}`);
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

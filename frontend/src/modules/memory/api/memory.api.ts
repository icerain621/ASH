import { api } from "@/services/http/client";

export type MemoryCandidate = {
  id: string;
  layer: string;
  title: string;
  status: string;
};

export function listCandidates(limit = 50) {
  return api<{ items: MemoryCandidate[] }>(`/memory/candidates?limit=${limit}`);
}

export function createCandidate(body: Record<string, unknown>) {
  return api<{ candidateId: string }>("/memory/candidates", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function reviewCandidate(candidateId: string, body: Record<string, unknown>) {
  return api<{ ok: boolean }>(`/memory/candidates/${candidateId}/review`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

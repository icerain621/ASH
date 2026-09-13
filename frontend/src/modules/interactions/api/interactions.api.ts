import { api } from "@/services/http/client";

/** GV04 interaction Timeline / MemoryLink client. */

export type InteractionThread = {
  id: string;
  spaceId: string;
  sessionId?: string;
  runId: string;
  kind: string;
  status: string;
  digest?: string;
  headSeq?: number;
};

export type InteractionMemoryLink = {
  id: string;
  spaceId: string;
  sessionId: string;
  threadId: string;
  runId: string;
  eventSeq: number;
  memoryId: string;
  linkType: "hit_used" | "context_ref" | "citation" | "candidate_out" | string;
  digest?: string;
};

export type InteractionFoldResult = {
  sessionId?: string;
  threadId: string;
  runId: string;
  spaceId: string;
  nodes: Array<{ id: string; seq: number; ts: number; type: string; visibility: string; payload?: unknown }>;
  links: InteractionMemoryLink[];
  digest: string;
  headSeq: number;
};

export async function getInteractionByRun(runId: string) {
  return api<{ runId: string; thread: InteractionThread }>(
    `/interactions/by-run/${encodeURIComponent(runId)}`,
  );
}

export async function listInteractionSessionThreads(sessionId: string) {
  return api<{ sessionId: string; items: InteractionThread[] }>(
    `/interactions/sessions/${encodeURIComponent(sessionId)}/threads`,
  );
}

export async function getInteractionThread(threadId: string) {
  return api<InteractionFoldResult>(`/interactions/threads/${encodeURIComponent(threadId)}`);
}

export async function listInteractionMemoryLinks(threadId: string) {
  return api<{ threadId: string; items: InteractionMemoryLink[] }>(
    `/interactions/threads/${encodeURIComponent(threadId)}/memory-links`,
  );
}

export async function ensureInteractionThread(body: { runId: string; sessionId?: string; spaceId?: string }) {
  return api<InteractionThread>("/interactions/threads/ensure", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export type InteractionReplayResult = {
  threadId: string;
  ok: boolean;
  digest: string;
  sealedDigest?: string;
  mismatchSeq?: number;
  status: string;
  nodes: InteractionFoldResult["nodes"];
  links: InteractionMemoryLink[];
};

export type InteractionCompareResult = {
  leftThreadId: string;
  rightThreadId: string;
  leftDigest: string;
  rightDigest: string;
  nodesAdded: string[];
  nodesRemoved: string[];
  nodesChanged: string[];
  linksAdded: string[];
  linksRemoved: string[];
};

export async function sealInteractionThread(threadId: string) {
  return api<InteractionThread>(`/interactions/threads/${encodeURIComponent(threadId)}/seal`, {
    method: "POST",
  });
}

export async function replayInteractionThread(threadId: string) {
  return api<InteractionReplayResult>(`/interactions/threads/${encodeURIComponent(threadId)}/replay`, {
    method: "POST",
  });
}

export async function compareInteractionThreads(left: string, right: string) {
  return api<InteractionCompareResult>("/interactions/compare", {
    method: "POST",
    body: JSON.stringify({ left, right }),
  });
}

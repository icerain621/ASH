import { api } from "@/services/http/client";

export type ObserveLens = "global" | "agent" | "memory";

export type CountItem = { id: string; count: number };

export type GlobalLensView = {
  generatedAt: number;
  runningRuns: number;
  failedClusters: CountItem[];
  templateUsage: CountItem[];
  gateRejects: number;
  reviewPending: number;
  doctorChip: string;
};

export type LensEvent = {
  type: string;
  lens: ObserveLens;
  runId?: string;
  ts: number;
  payload?: Record<string, unknown>;
  redacted?: boolean;
};

export type AgentLensView = {
  runId: string;
  traceId: string;
  events: LensEvent[];
  waterfall?: { runId: string; spans?: unknown[] };
};

export type MemoryStage = { stage: string; at?: number; detail?: string; eventId?: string };

export type MemoryLensView = {
  memoryId: string;
  status: string;
  title: string;
  stages: MemoryStage[];
  events: LensEvent[];
  redacted: boolean;
};

export function getGlobalLens(spaceId?: string) {
  const q = spaceId ? `?spaceId=${encodeURIComponent(spaceId)}` : "";
  return api<GlobalLensView>(`/observability/lenses/global${q}`);
}

export function getAgentLens(runId: string) {
  return api<AgentLensView>(`/observability/lenses/agent?runId=${encodeURIComponent(runId)}`);
}

export function getMemoryLens(id: string) {
  return api<MemoryLensView>(`/observability/lenses/memory?id=${encodeURIComponent(id)}`);
}

/** Console deep links (I05 / I09 / H06). */
export function observeDeepLink(lens: ObserveLens, opts?: { run?: string; id?: string }) {
  const q = new URLSearchParams({ lens });
  if (opts?.run) q.set("run", opts.run);
  if (opts?.id) q.set("id", opts.id);
  return `/ui/observe?${q.toString()}`;
}

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

/** I03: ready when context.packed exists; missing → console「未接入」. */
export type PackAccess = "ready" | "missing";

export type AgentLensView = {
  runId: string;
  traceId: string;
  packAccess: PackAccess;
  events: LensEvent[];
  waterfall?: { runId: string; spans?: unknown[] };
};

export function packAccessLabel(access: PackAccess | undefined): string {
  return access === "ready" ? "已接入" : "未接入";
}

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

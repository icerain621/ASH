import { api } from "@/services/http/client";

export type ScenarioRef = { name: string; scenarioVersion: string };

export type RunSummary = {
  runId: string;
  traceId: string;
  scenario: ScenarioRef;
  policyProfile: string;
  status: string;
  startedAt: number;
  finishedAt?: number;
  recovered?: boolean;
};

export type CreateRunRequest = {
  scenario: ScenarioRef;
  inputs: Record<string, unknown>;
  policyProfile?: string;
  repo?: { root?: string; revision?: string; branch?: string };
};

export type ReplayRequest = {
  mode: "exact" | "latest_memory";
  overrides?: Record<string, unknown>;
};

export function listRuns(limit = 30) {
  return api<{ items: RunSummary[] }>(`/runs?limit=${limit}`);
}

export function getRun(runId: string) {
  return api<RunSummary>(`/runs/${runId}`);
}

export function createRun(body: CreateRunRequest) {
  return api<{ runId: string; traceId: string }>("/runs", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function resumeRun(runId: string) {
  return api<{ runId: string; traceId: string; status: string }>(`/runs/${runId}/resume`, {
    method: "POST",
    body: "{}",
  });
}

export function replayRun(runId: string, body: ReplayRequest) {
  return api<{ runId: string; traceId: string }>(`/runs/${runId}/replay`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getRunArtifacts(runId: string) {
  return api<unknown>(`/runs/${runId}/artifacts`);
}

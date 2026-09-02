import { api } from "@/services/http/client";

export type WakerDuty = {
  id: string;
  spaceId: string;
  kind: string;
  enabled: boolean;
  intervalMs: number;
  nextRunAt: string;
  updatedAt?: string;
};

export type WakerDutyRun = {
  id: string;
  dutyId: string;
  spaceId?: string;
  kind: string;
  status: string;
  matched: number;
  flagged: number;
  canceled: number;
  summary: string;
  startedAt: string;
  finishedAt?: string;
};

export type WakerStatus = {
  duties: WakerDuty[];
  recentRuns: WakerDutyRun[];
  allowCancel: boolean;
  interval?: string;
  intervalMs?: number;
};

export type WakerQueueItem = {
  runId: string;
  spaceId: string;
  status: string;
  reason: string;
  kind?: string;
  ageMs?: number;
  updatedAt?: number;
};

export type WakerQueue = {
  items: WakerQueueItem[];
  count: number;
  maxAge?: string;
  maxAgeMs?: number;
  inspectedAt?: string;
};

export type WakerDutiesResponse = {
  duties: WakerDuty[];
};

export type WakerSweepRequest = {
  spaceId?: string;
  dryRun?: boolean;
  action?: string;
  confirm?: string;
  maxAge?: string;
  limit?: number;
};

export type WakerSweepResponse = {
  ok: boolean;
  dryRun: boolean;
  action: string;
  matched: number;
  flagged: number;
  canceled?: number;
  runIds?: string[];
  maxAge: string;
  summary?: string;
};

function qs(params: Record<string, string | number | undefined>) {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== "") search.set(key, String(value));
  });
  const suffix = search.toString();
  return suffix ? `?${suffix}` : "";
}

export function getWakerStatus(spaceId?: string, recent?: number) {
  return api<WakerStatus>(`/waker/status${qs({ spaceId, recent })}`);
}

export function getWakerQueue(params?: { spaceId?: string; limit?: number; maxAge?: string }) {
  return api<WakerQueue>(`/waker/queue${qs({ spaceId: params?.spaceId, limit: params?.limit, maxAge: params?.maxAge })}`);
}

export function listWakerDuties(spaceId?: string) {
  return api<WakerDutiesResponse>(`/waker/duties${qs({ spaceId })}`);
}

export function postWakerSweep(body: WakerSweepRequest) {
  return api<WakerSweepResponse>("/waker/sweep", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function postWakerDutyRun(id: string, body?: { dryRun?: boolean }) {
  return api<WakerSweepResponse>(`/waker/duties/${encodeURIComponent(id)}/run`, {
    method: "POST",
    body: JSON.stringify(body ?? {}),
  });
}

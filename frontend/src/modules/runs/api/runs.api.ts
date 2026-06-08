import { api } from "@/services/http/client";

export type ScenarioRef = { name: string; scenarioVersion: string };

export type RunSummary = {
  runId: string;
  traceId: string;
  scenario: ScenarioRef;
  policyProfile: string;
  status: string;
  spaceId?: string;
  actorRole?: string;
  startedAt: number;
  finishedAt?: number;
  recovered?: boolean;
};

export type CreateRunRequest = {
  scenario: ScenarioRef;
  inputs: Record<string, unknown>;
  policyProfile?: string;
  actorRole?: string;
  spaceId?: string;
  repo?: { root?: string; revision?: string; branch?: string };
};

export type CreateRunResponse = {
  runId: string;
  traceId: string;
  status?: string;
  executionError?: string;
};

export type ReplayRequest = {
  mode: "exact" | "latest_memory";
  overrides?: Record<string, unknown>;
};

export type ApproveRequest = {
  actorId?: string;
  reason?: string;
};

export type TimelineItem = {
  seq?: number;
  ts?: number;
  type: string;
  severity?: string;
  stepId?: string;
  status?: string;
  payload?: unknown;
};

export type ToolCall = {
  id: string;
  stepId?: string;
  tool: string;
  risk: string;
  status: string;
  durationMs?: number;
  error?: string;
};

export type AgentTask = {
  id: string;
  stepId?: string;
  adapter: string;
  agentId?: string;
  execGoTaskId?: string;
  status: string;
  durationMs?: number;
  errorCode?: string;
  errorMessage?: string;
  stdoutSummary?: string;
  stderrSummary?: string;
};

export type QualityMetric = {
  id: string;
  runId: string;
  spaceId: string;
  name: string;
  value: number;
  unit?: string;
  createdAt: string;
};

export type WaterfallSpan = {
  id: string;
  parentId?: string;
  runId: string;
  type: string;
  name: string;
  status: string;
  startTs?: number;
  endTs?: number;
  durationMs?: number;
  attributes?: Record<string, unknown>;
};

export type FailureAttribution = {
  type: string;
  ref: string;
  code?: string;
  message?: string;
};

export type WaterfallMetric = {
  name: string;
  value: number;
  unit?: string;
};

export type RunWaterfall = {
  runId: string;
  traceId: string;
  status: string;
  generatedAt: number;
  spans: WaterfallSpan[];
  failures?: FailureAttribution[];
  metrics?: WaterfallMetric[];
};

export type ArtifactItem = {
  type: string;
  name: string;
  uri: string;
  digest: string;
  contentType?: string;
  producer?: {
    stepId?: string;
    role?: string;
    eventRange?: string;
    agentTaskId?: string;
    evidenceRefs?: string[];
  };
  sizeBytes?: number;
};

export type RunArtifactsResponse = {
  manifest?: {
    artifacts?: ArtifactItem[];
  };
  artifacts?: ArtifactItem[];
};

export type ArtifactAccessResponse = {
  runId: string;
  name: string;
  uri: string;
  signedUrl: string;
  expiresAt: number;
  digest: string;
  contentType?: string;
  sizeBytes?: number;
};

export type Checkpoint = {
  id: string;
  runId: string;
  stepId: string;
  snapshotDigest: string;
  uri?: string;
  storeKey?: string;
  contentType?: string;
  sizeBytes?: number;
  strategy?: string;
  createdAt?: string;
};

export type CheckpointAccessResponse = {
  runId: string;
  checkpointId: string;
  stepId: string;
  uri: string;
  signedUrl: string;
  expiresAt: number;
  snapshotDigest: string;
  contentType?: string;
  sizeBytes?: number;
};

type RawRecord = Record<string, unknown>;

function itemsFrom<T>(res: { items?: T[]; Items?: T[] }) {
  return res.items ?? res.Items ?? [];
}

function str(raw: RawRecord, camel: string, pascal: string) {
  const value = raw[camel] ?? raw[pascal];
  return typeof value === "string" ? value : "";
}

function num(raw: RawRecord, camel: string, pascal: string) {
  const value = raw[camel] ?? raw[pascal];
  return typeof value === "number" ? value : 0;
}

function normalizeToolCall(raw: RawRecord): ToolCall {
  return {
    id: str(raw, "id", "ID"),
    stepId: str(raw, "stepId", "StepID"),
    tool: str(raw, "tool", "Tool"),
    risk: str(raw, "risk", "Risk"),
    status: str(raw, "status", "Status"),
    durationMs: num(raw, "durationMs", "DurationMs"),
    error: str(raw, "error", "Error"),
  };
}

function normalizeAgentTask(raw: RawRecord): AgentTask {
  return {
    id: str(raw, "id", "ID"),
    stepId: str(raw, "stepId", "StepID"),
    adapter: str(raw, "adapter", "Adapter"),
    agentId: str(raw, "agentId", "AgentID"),
    execGoTaskId: str(raw, "execGoTaskId", "ExecGoTaskID"),
    status: str(raw, "status", "Status"),
    durationMs: num(raw, "durationMs", "DurationMs"),
    errorCode: str(raw, "errorCode", "ErrorCode"),
    errorMessage: str(raw, "errorMessage", "ErrorMessage"),
    stdoutSummary: str(raw, "stdoutSummary", "StdoutSummary"),
    stderrSummary: str(raw, "stderrSummary", "StderrSummary"),
  };
}

function normalizeQualityMetric(raw: RawRecord): QualityMetric {
  return {
    id: str(raw, "id", "ID"),
    runId: str(raw, "runId", "RunID"),
    spaceId: str(raw, "spaceId", "SpaceID"),
    name: str(raw, "name", "Name"),
    value: num(raw, "value", "Value"),
    unit: str(raw, "unit", "Unit"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
  };
}

function normalizeCheckpoint(raw: RawRecord): Checkpoint {
  return {
    id: str(raw, "id", "ID"),
    runId: str(raw, "runId", "RunID"),
    stepId: str(raw, "stepId", "StepID"),
    snapshotDigest: str(raw, "snapshotDigest", "SnapshotDigest"),
    uri: str(raw, "uri", "URI"),
    storeKey: str(raw, "storeKey", "StoreKey"),
    contentType: str(raw, "contentType", "ContentType"),
    sizeBytes: num(raw, "sizeBytes", "SizeBytes"),
    strategy: str(raw, "strategy", "Strategy"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
  };
}

export function listRuns(limit = 30) {
  return api<{ items: RunSummary[] }>(`/runs?limit=${limit}`);
}

export function getRun(runId: string) {
  return api<RunSummary>(`/runs/${runId}`);
}

export function createRun(body: CreateRunRequest) {
  return api<CreateRunResponse>("/runs", {
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

export function cancelRun(runId: string) {
  return api<{ runId: string; status: string }>(`/runs/${runId}/cancel`, {
    method: "POST",
    body: "{}",
  });
}

export function approveRun(runId: string, body: ApproveRequest) {
  return api<{ runId: string; ok: boolean }>(`/runs/${runId}/approve`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getRunArtifacts(runId: string) {
  return api<RunArtifactsResponse>(`/runs/${runId}/artifacts`);
}

export function getRunArtifactAccess(runId: string, artifactName: string, ttlSeconds = 900) {
  return api<ArtifactAccessResponse>(
    `/runs/${runId}/artifacts/${encodeURIComponent(artifactName)}/access?ttlSeconds=${ttlSeconds}`,
  );
}

export function getRunCheckpoints(runId: string) {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/runs/${runId}/checkpoints`).then((res) => ({
    items: itemsFrom(res).map(normalizeCheckpoint),
  }));
}

export function getRunCheckpointAccess(runId: string, checkpointId: string, ttlSeconds = 900) {
  return api<CheckpointAccessResponse>(
    `/runs/${runId}/checkpoints/${encodeURIComponent(checkpointId)}/access?ttlSeconds=${ttlSeconds}`,
  );
}

export function getRunTimeline(runId: string) {
  return api<{ items: TimelineItem[] }>(`/runs/${runId}/timeline`);
}

export function getRunToolCalls(runId: string) {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/runs/${runId}/tool-calls`).then((res) => ({
    items: itemsFrom(res).map(normalizeToolCall),
  }));
}

export function getRunAgentTasks(runId: string) {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/runs/${runId}/agent-tasks`).then((res) => ({
    items: itemsFrom(res).map(normalizeAgentTask),
  }));
}

export function getRunQualityMetrics(runId: string) {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/runs/${runId}/quality-metrics`).then((res) => ({
    items: itemsFrom(res).map(normalizeQualityMetric),
  }));
}

export function getRunWaterfall(runId: string) {
  return api<RunWaterfall>(`/observability/waterfall/${runId}`);
}

export type ProvenanceLink = {
  kind: string;
  ref: string;
};

export type RunProvenance = {
  runId: string;
  traceId: string;
  scenario: ScenarioRef;
  status: string;
  toolCalls: number;
  agentTasks: number;
  artifacts: number;
  events: number;
  modelUsage: number;
  links: ProvenanceLink[];
};

function normalizeProvenance(raw: RawRecord): RunProvenance {
  const scenario = (raw.scenario ?? raw.Scenario) as RawRecord | undefined;
  const rawLinks = raw.links ?? raw.Links;
  const linksRaw = Array.isArray(rawLinks) ? rawLinks : [];
  return {
    runId: str(raw, "runId", "RunID"),
    traceId: str(raw, "traceId", "TraceID"),
    scenario: {
      name: str(scenario ?? {}, "name", "Name"),
      scenarioVersion: str(scenario ?? {}, "scenarioVersion", "ScenarioVersion"),
    },
    status: str(raw, "status", "Status"),
    toolCalls: num(raw, "toolCalls", "ToolCalls"),
    agentTasks: num(raw, "agentTasks", "AgentTasks"),
    artifacts: num(raw, "artifacts", "Artifacts"),
    events: num(raw, "events", "Events"),
    modelUsage: num(raw, "modelUsage", "ModelUsage"),
    links: linksRaw.map((item) => {
      const link = (item ?? {}) as RawRecord;
      return {
        kind: str(link, "kind", "Kind"),
        ref: str(link, "ref", "Ref"),
      };
    }),
  };
}

export function getRunProvenance(runId: string) {
  return api<RawRecord>(`/runs/${runId}/provenance`).then(normalizeProvenance);
}

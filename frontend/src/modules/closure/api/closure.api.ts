import { api } from "@/services/http/client";

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

function bool(raw: RawRecord, camel: string, pascal: string) {
  const value = raw[camel] ?? raw[pascal];
  return typeof value === "boolean" ? value : false;
}

function arr(raw: RawRecord, camel: string, pascal: string) {
  const value = raw[camel] ?? raw[pascal];
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : [];
}

function qs(params: Record<string, string | number | boolean | undefined>) {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== "") search.set(key, String(value));
  });
  const suffix = search.toString();
  return suffix ? `?${suffix}` : "";
}

export type RepoConnection = {
  id: string;
  provider: string;
  owner: string;
  repo: string;
  status: string;
  defaultBranch: string;
  lastSyncAt?: string;
};

export type CIRun = {
  id: string;
  connectionId: string;
  providerRunId: string;
  workflow: string;
  status: string;
  conclusion: string;
  attempt: number;
  branch: string;
  commitSha: string;
  runUrl: string;
  createdAt?: string;
};

export type CIJob = {
  id: string;
  ciRunId: string;
  providerJobId: string;
  name: string;
  status: string;
  conclusion: string;
  logDigest?: string;
};

export type CIDiagnosis = {
  id: string;
  connectionId: string;
  runId: string;
  jobId: string;
  status: string;
  rootCause: string;
  fixSuggestions: string[];
  evidenceRefs: string[];
  confidence: number;
  adopted: boolean;
  decisionStatus: string;
  decisionReason?: string;
  logDigest?: string;
  createdAt?: string;
};

export type Feedback = {
  id: string;
  spaceId: string;
  targetType: string;
  targetId: string;
  rating: number;
  category: string;
  status: string;
  severity: string;
  source: string;
  comment: string;
  actorId: string;
  createdAt?: string;
  updatedAt?: string;
};

export type AlertRule = {
  id: string;
  name: string;
  metric: string;
  condition: string;
  threshold: number;
  windowMinutes: number;
  severity: string;
  enabled: boolean;
  description?: string;
};

export type AlertEvent = {
  id: string;
  ruleName: string;
  severity: string;
  status: string;
  targetType: string;
  targetId: string;
  message: string;
  evidenceRefsJson?: string;
  triggeredAt?: string;
};

export type AlertEvaluation = {
  spaceId: string;
  evaluatedAt: string;
  results: Array<{
    ruleId: string;
    ruleName: string;
    metric: string;
    status: string;
    message: string;
    value: number;
    threshold: number;
    evidenceRefs: string[];
  }>;
  events: AlertEvent[];
};

export type TraceView = {
  traceId: string;
  runs: RawRecord[];
  events: RawRecord[];
  toolCalls: RawRecord[];
  agentTasks: RawRecord[];
  auditLogs: RawRecord[];
};

export type ReleaseRecord = {
  id: string;
  version: string;
  title: string;
  status: string;
  gateStatus: string;
  canaryStrategy?: string;
  createdAt?: string;
};

export type ReleaseChecklistItem = {
  id: string;
  itemKey: string;
  label: string;
  status: string;
  evidenceRef?: string;
};

export type ReleaseGateResult = {
  id: string;
  gateKey: string;
  status: string;
  message: string;
  evidenceRefsJson?: string;
};

export type ReleaseGateEvaluation = {
  releaseId: string;
  overall: string;
  results: ReleaseGateResult[];
  evidenceRefs: string[];
  evaluatedAt: string;
};

function normalizeFeedback(raw: RawRecord): Feedback {
  return {
    id: str(raw, "id", "ID"),
    spaceId: str(raw, "spaceId", "SpaceID"),
    targetType: str(raw, "targetType", "TargetType"),
    targetId: str(raw, "targetId", "TargetID"),
    rating: num(raw, "rating", "Rating"),
    category: str(raw, "category", "Category") || "general",
    status: str(raw, "status", "Status") || "open",
    severity: str(raw, "severity", "Severity") || "normal",
    source: str(raw, "source", "Source") || "ui",
    comment: str(raw, "comment", "Comment"),
    actorId: str(raw, "actorId", "ActorID"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
    updatedAt: str(raw, "updatedAt", "UpdatedAt"),
  };
}

function normalizeDiagnosis(raw: RawRecord): CIDiagnosis {
  return {
    id: str(raw, "id", "ID"),
    connectionId: str(raw, "connectionId", "ConnectionID"),
    runId: str(raw, "runId", "RunID"),
    jobId: str(raw, "jobId", "JobID"),
    status: str(raw, "status", "Status"),
    rootCause: str(raw, "rootCause", "RootCause"),
    fixSuggestions: arr(raw, "fixSuggestions", "FixSuggestions"),
    evidenceRefs: arr(raw, "evidenceRefs", "EvidenceRefs"),
    confidence: num(raw, "confidence", "Confidence"),
    adopted: bool(raw, "adopted", "Adopted"),
    decisionStatus: str(raw, "decisionStatus", "DecisionStatus") || "pending",
    decisionReason: str(raw, "decisionReason", "DecisionReason"),
    logDigest: str(raw, "logDigest", "LogDigest"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
  };
}

export function listRepoConnections() {
  return api<{ items?: RepoConnection[]; Items?: RepoConnection[] }>("/repo/connections").then((res) => ({
    items: itemsFrom(res),
  }));
}

export function listCIRuns(params: { connectionId?: string; sync?: boolean; limit?: number } = {}) {
  return api<{ items?: CIRun[]; Items?: CIRun[] }>(`/ci/runs${qs(params)}`).then((res) => ({ items: itemsFrom(res) }));
}

export function listCIJobs(params: { runId?: string; sync?: boolean; limit?: number } = {}) {
  return api<{ items?: CIJob[]; Items?: CIJob[] }>(`/ci/jobs${qs(params)}`).then((res) => ({ items: itemsFrom(res) }));
}

export function listCIDiagnoses(params: {
  connectionId?: string;
  runId?: string;
  jobId?: string;
  decisionStatus?: string;
  limit?: number;
} = {}) {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/ci/diagnoses${qs(params)}`).then((res) => ({
    items: itemsFrom(res).map(normalizeDiagnosis),
  }));
}

export function diagnoseCIFailure(body: { connectionId?: string; runId?: string; jobId?: string; logText?: string }) {
  return api<RawRecord>("/ci/failures/diagnose", {
    method: "POST",
    body: JSON.stringify(body),
  }).then(normalizeDiagnosis);
}

export function adoptCIDiagnosis(id: string, reason: string) {
  return api<RawRecord>(`/ci/diagnoses/${encodeURIComponent(id)}/adopt`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  }).then(normalizeDiagnosis);
}

export function dismissCIDiagnosis(id: string, reason: string) {
  return api<RawRecord>(`/ci/diagnoses/${encodeURIComponent(id)}/dismiss`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  }).then(normalizeDiagnosis);
}

export function listFeedback(params: { targetType?: string; category?: string; rating?: number; status?: string; severity?: string; limit?: number } = {}) {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/feedback${qs(params)}`).then((res) => ({
    items: itemsFrom(res).map(normalizeFeedback),
  }));
}

export function createFeedback(body: Partial<Feedback> & { targetType: string; targetId: string }) {
  return api<RawRecord>("/feedback", { method: "POST", body: JSON.stringify(body) }).then(normalizeFeedback);
}

export function updateFeedback(id: string, body: Pick<Partial<Feedback>, "category" | "status" | "severity" | "comment">) {
  return api<RawRecord>(`/feedback/${encodeURIComponent(id)}`, { method: "PATCH", body: JSON.stringify(body) }).then(normalizeFeedback);
}

export function listAlerts(params: { status?: string; limit?: number } = {}) {
  return api<{ items?: AlertEvent[]; Items?: AlertEvent[] }>(`/observability/alerts${qs(params)}`).then((res) => ({
    items: itemsFrom(res),
  }));
}

export function listAlertRules() {
  return api<{ items?: AlertRule[]; Items?: AlertRule[] }>("/observability/alert-rules").then((res) => ({ items: itemsFrom(res) }));
}

export function putAlertRules(items: AlertRule[]) {
  return api<{ items?: AlertRule[]; Items?: AlertRule[] }>("/observability/alert-rules", {
    method: "PUT",
    body: JSON.stringify({ items }),
  }).then((res) => ({ items: itemsFrom(res) }));
}

export function evaluateAlerts() {
  return api<AlertEvaluation>("/observability/alerts/evaluate", { method: "POST" });
}

export function getTrace(traceId: string) {
  return api<TraceView>(`/observability/trace/${encodeURIComponent(traceId)}`);
}

export async function getPrometheusText() {
  const res = await fetch("/metrics");
  if (!res.ok) throw new Error(res.statusText);
  return res.text();
}

export function listReleases(params: { limit?: number } = {}) {
  return api<{ items?: ReleaseRecord[]; Items?: ReleaseRecord[] }>(`/releases${qs(params)}`).then((res) => ({ items: itemsFrom(res) }));
}

export function createRelease(body: { version: string; title?: string; canaryStrategy?: string }) {
  return api<ReleaseRecord>("/releases", { method: "POST", body: JSON.stringify(body) });
}

export function getReleaseChecklist(releaseId: string) {
  return api<{ items?: ReleaseChecklistItem[]; Items?: ReleaseChecklistItem[] }>(
    `/releases/${encodeURIComponent(releaseId)}/checklist`,
  ).then((res) => ({ items: itemsFrom(res) }));
}

export function patchReleaseChecklist(releaseId: string, items: Array<{ id?: string; itemKey?: string; status?: string; evidenceRef?: string }>) {
  return api<{ items?: ReleaseChecklistItem[]; Items?: ReleaseChecklistItem[] }>(
    `/releases/${encodeURIComponent(releaseId)}/checklist`,
    { method: "PATCH", body: JSON.stringify({ items }) },
  ).then((res) => ({ items: itemsFrom(res) }));
}

export function evaluateReleaseGate(releaseId: string) {
  return api<ReleaseGateEvaluation>(`/releases/${encodeURIComponent(releaseId)}/gate`, { method: "POST" });
}

export function createRollbackDrill(releaseId: string, body: { scenario: string; status?: string; durationMs?: number; evidenceRefs?: string[]; notes?: string }) {
  return api<RawRecord>(`/releases/${encodeURIComponent(releaseId)}/rollback-drills`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

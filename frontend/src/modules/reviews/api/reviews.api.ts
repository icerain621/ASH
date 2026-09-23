import { api } from "@/services/http/client";

export type ReviewItem = {
  id: string;
  queue: "memory" | "orchestration" | "appeal";
  targetType: string;
  targetId: string;
  title: string;
  summary?: string;
  diff?: string;
  status: string;
  spaceId: string;
  createdAt: number;
  assigneeId?: string;
  slaBreach?: boolean;
  ageHours?: number;
};

export type AssignPayload = {
  assigneeId: string;
  actorId?: string;
};

export type ScenarioPatch = {
  id: string;
  spaceId: string;
  scenarioName: string;
  fromVersion?: string;
  toVersion?: string;
  title: string;
  diffText: string;
  status: string;
  createdAt: number;
};

export type HarnessProfile = {
  id: string;
  spaceId: string;
  name: string;
  version: number;
  status: string;
  spec?: { sandbox?: { defaultMode?: string }; provider?: { kind?: string } };
};

export function listReviewsQueue(queue = "all", limit = 50) {
  const q = new URLSearchParams({ queue, limit: String(limit) });
  return api<{ items: ReviewItem[]; queue?: string }>(`/reviews/queue?${q}`);
}

export function decideReview(
  reviewId: string,
  body: {
    decision: string;
    reason: string;
    policyProfile?: string;
    rubric?: { correctness: number; safety: number; citable: number; efficiency: number };
  },
) {
  return api(`/reviews/${encodeURIComponent(reviewId)}/decide`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function createScoreAppeal(scoreEventId: string, body: { reason: string; actorId?: string }) {
  return api<ReviewItem>(`/scores/${encodeURIComponent(scoreEventId)}/appeal`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function assignReview(reviewId: string, body: AssignPayload) {
  return api<{ ok: boolean; reviewId: string; assigneeId: string }>(
    `/reviews/${encodeURIComponent(reviewId)}/assign`,
    { method: "POST", body: JSON.stringify(body) },
  );
}

export function listScenarioPatches(status = "") {
  const q = status ? `?status=${encodeURIComponent(status)}` : "";
  return api<{ items: ScenarioPatch[] }>(`/scenario-patches${q}`);
}

export function createScenarioPatch(body: {
  scenarioName: string;
  fromVersion?: string;
  toVersion?: string;
  title: string;
  diffText: string;
}) {
  return api<ScenarioPatch>("/scenario-patches", { method: "POST", body: JSON.stringify(body) });
}

export function submitScenarioPatchReview(patchId: string) {
  return api<ScenarioPatch>(`/scenario-patches/${patchId}/submit-review`, { method: "POST" });
}

export function listHarnessProfiles(status = "", name = "") {
  const q = new URLSearchParams();
  if (status) q.set("status", status);
  if (name) q.set("name", name);
  const qs = q.toString();
  return api<{ items: HarnessProfile[]; sandboxBackends?: string[] }>(`/harness/profiles${qs ? `?${qs}` : ""}`);
}

export function loadActiveHarnessProfile(name = "default") {
  return api<{ profile: HarnessProfile }>(`/harness/profiles/active?name=${encodeURIComponent(name)}`);
}

export function createHarnessProfile(body: { name: string; spec: Record<string, unknown> }) {
  return api<HarnessProfile>("/harness/profiles", { method: "POST", body: JSON.stringify(body) });
}

export function submitHarnessReview(profileId: string) {
  return api<HarnessProfile>(`/harness/profiles/${profileId}/submit-review`, { method: "POST" });
}

export function promoteHarnessProfile(profileId: string) {
  return api<HarnessProfile>(`/harness/profiles/${profileId}/promote`, { method: "POST" });
}

export function rollbackHarnessProfile(profileId: string) {
  return api<HarnessProfile>(`/harness/profiles/${profileId}/rollback`, { method: "POST" });
}

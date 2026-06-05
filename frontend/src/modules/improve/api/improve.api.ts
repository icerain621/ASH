import { api } from "@/services/http/client";

export type ArtifactCompare = {
  baselineRunId: string;
  experimentRunId: string;
  matched: number;
  changed: number;
  missing: number;
  byType?: Record<string, string>;
};

export type ImproveProposal = {
  id: string;
  title: string;
  description?: string;
  baselineRunId: string;
  experimentRunId?: string;
  status: string;
  changeSummary?: string;
  canaryPercent: number;
  compare?: ArtifactCompare;
};

export function listImproveProposals(limit = 20) {
  return api<{ items: ImproveProposal[] }>(`/improve/proposals?limit=${limit}`);
}

export function createImproveProposal(body: {
  title: string;
  baselineRunId: string;
  description?: string;
  changeSummary?: string;
}) {
  return api<ImproveProposal>("/improve/proposals", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function startImproveExperiment(proposalId: string) {
  return api<{ proposalId: string; experimentRunId: string; compare?: ArtifactCompare }>(
    `/improve/proposals/${proposalId}/experiment`,
    { method: "POST", body: "{}" },
  );
}

export function startImproveCanary(proposalId: string, percent: number) {
  return api<{ ok: boolean; status: string }>(`/improve/proposals/${proposalId}/canary`, {
    method: "POST",
    body: JSON.stringify({ percent }),
  });
}

export function promoteImproveProposal(proposalId: string) {
  return api<{ ok: boolean; status: string }>(`/improve/proposals/${proposalId}/promote`, {
    method: "POST",
    body: "{}",
  });
}

export function rollbackImproveProposal(proposalId: string) {
  return api<{ ok: boolean; status: string }>(`/improve/proposals/${proposalId}/rollback`, {
    method: "POST",
    body: "{}",
  });
}

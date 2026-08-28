import { api } from "@/services/http/client";

export type BoardItem = {
  id: string;
  kind: "plan" | "run" | string;
  title: string;
  status: string;
  column: string;
  scenario?: string;
  runId?: string;
  planId?: string;
  spaceId: string;
  updatedAt: number;
};

export type QuestBoard = {
  columns: Record<string, BoardItem[]>;
};

export type DiffLine = {
  kind: string;
  text: string;
  oldNo?: number;
  newNo?: number;
  index: number;
};

export type DiffHunk = {
  header: string;
  oldStart: number;
  newStart: number;
  lines: DiffLine[];
};

export type DiffFile = {
  path: string;
  hunks: DiffHunk[];
};

export type RunDiff = {
  runId: string;
  raw: string;
  files: DiffFile[];
  contextRefs?: string[];
};

export type DiffComment = {
  id: string;
  runId: string;
  filePath: string;
  lineIndex: number;
  side: string;
  body: string;
  createdBy?: string;
  createdAt: number;
};

export function getQuestBoard(limit = 80) {
  return api<QuestBoard>(`/quest/board?limit=${limit}`);
}

export function getRunDiff(runId: string) {
  return api<RunDiff>(`/runs/${runId}/diff`);
}

export function listDiffComments(runId: string) {
  return api<{ items: DiffComment[] }>(`/runs/${runId}/diff/comments`);
}

export function createDiffComment(
  runId: string,
  body: { filePath: string; lineIndex: number; side?: string; body: string },
) {
  return api<DiffComment>(`/runs/${runId}/diff/comments`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function rateRunStep(runId: string, stepId: string, body: { rating: number; comment?: string }) {
  return api(`/runs/${runId}/steps/${encodeURIComponent(stepId)}/rate`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

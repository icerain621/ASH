import { api } from "@/services/http/client";

export type AgentWorkspaceView = {
  id: string;
  spaceId: string;
  title: string;
  repoRoot?: string;
  sessionIds: string[];
  createdAt?: number;
  updatedAt?: number;
  status: string;
};

export type AgentWorkspaceListResponse = {
  items: AgentWorkspaceView[];
};

export async function listAgentWorkspaces(opts?: {
  spaceId?: string;
  limit?: number;
}): Promise<AgentWorkspaceListResponse> {
  const q = new URLSearchParams();
  if (opts?.spaceId) q.set("spaceId", opts.spaceId);
  if (opts?.limit != null) q.set("limit", String(opts.limit));
  const suffix = q.toString() ? `?${q}` : "";
  return api<AgentWorkspaceListResponse>(`/agent-workspaces${suffix}`);
}

export async function createAgentWorkspace(body: {
  title: string;
  repoRoot?: string;
  spaceId?: string;
}): Promise<AgentWorkspaceView> {
  return api<AgentWorkspaceView>("/agent-workspaces", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export async function patchAgentWorkspace(
  workspaceId: string,
  body: { title?: string; repoRoot?: string; sessionIds?: string[] },
): Promise<AgentWorkspaceView> {
  return api<AgentWorkspaceView>(`/agent-workspaces/${encodeURIComponent(workspaceId)}`, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export async function attachAgentWorkspaceSession(
  workspaceId: string,
  sessionId: string,
): Promise<AgentWorkspaceView> {
  return api<AgentWorkspaceView>(
    `/agent-workspaces/${encodeURIComponent(workspaceId)}/sessions`,
    {
      method: "POST",
      body: JSON.stringify({ sessionId }),
    },
  );
}

export async function closeAgentWorkspace(workspaceId: string): Promise<AgentWorkspaceView> {
  return api<AgentWorkspaceView>(`/agent-workspaces/${encodeURIComponent(workspaceId)}`, {
    method: "DELETE",
  });
}

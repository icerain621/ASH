import { api } from "@/services/http/client";

export type RegistryAsset = {
  id: string;
  spaceId: string;
  kind: string;
  refId?: string;
  name: string;
  status: string;
  createdAt: number;
};

export function listAgentAssets(status = "", limit = 50) {
  const q = new URLSearchParams({ limit: String(limit) });
  if (status) q.set("status", status);
  return api<{ items: RegistryAsset[] }>(`/agents/assets?${q}`);
}

export function createAgentAsset(body: { kind: string; name: string; refId?: string; status?: string }) {
  return api<RegistryAsset>("/agents/assets", { method: "POST", body: JSON.stringify(body) });
}

export function patchAgentAssetStatus(id: string, status: string) {
  return api<RegistryAsset>(`/agents/assets/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });
}

export function listMemoryAssets(status = "", limit = 50) {
  const q = new URLSearchParams({ limit: String(limit) });
  if (status) q.set("status", status);
  return api<{ items: RegistryAsset[] }>(`/memory/assets?${q}`);
}

export function createMemoryAsset(body: { kind: string; name: string; refId?: string; status?: string }) {
  return api<RegistryAsset>("/memory/assets", { method: "POST", body: JSON.stringify(body) });
}

export function patchMemoryAssetStatus(id: string, status: string) {
  return api<RegistryAsset>(`/memory/assets/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });
}

export type SpacePolicyResponse = {
  pack: {
    spaceId: string;
    citationMode: string;
    multiSign: boolean;
    reviewSlaHours: number;
    bodyJson?: string;
  };
  effective: {
    spaceId: string;
    spaceKind: string;
    citationMode: string;
    multiSign: boolean;
    reviewSlaHours: number;
    sources: string[];
  };
};

export function getSpacePolicy(spaceId: string) {
  return api<SpacePolicyResponse>(`/spaces/${encodeURIComponent(spaceId)}/policy`);
}

export function putSpacePolicy(
  spaceId: string,
  body: {
    citationMode?: string;
    multiSign?: boolean;
    reviewSlaHours?: number;
    bodyJson?: string;
  },
) {
  return api<SpacePolicyResponse>(`/spaces/${encodeURIComponent(spaceId)}/policy`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

import { api } from "@/services/http/client";

export type SkillItem = {
  id: string;
  name: string;
  description: string;
  license?: string;
  path: string;
  relPath: string;
  body?: string;
  contextRef: string;
};

export type SkillsList = {
  items: SkillItem[];
  repoRoot: string;
};

export function listSkills(repoRoot = ".") {
  const q = new URLSearchParams({ repoRoot });
  return api<SkillsList>(`/skills?${q}`);
}

export function getSkill(skillId: string, repoRoot = ".") {
  const q = new URLSearchParams({ repoRoot });
  return api<SkillItem>(`/skills/${encodeURIComponent(skillId)}?${q}`);
}

export type SkillPackVerifyResult = {
  ok: boolean;
  name?: string;
  version?: string;
  publisher?: string;
  digest?: string;
  message?: string;
};

export type SkillPackInstallResult = {
  ok: boolean;
  name: string;
  version: string;
  publisher: string;
  path: string;
  repoRoot: string;
};

export function verifySkillPack(body: {
  repoRoot?: string;
  spaceId?: string;
  packPath?: string;
  packBase64?: string;
  signature: string;
}) {
  return api<SkillPackVerifyResult>("/skills/packs/verify", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function installSkillPack(body: {
  repoRoot?: string;
  spaceId?: string;
  packPath?: string;
  packBase64?: string;
  signature: string;
}) {
  return api<SkillPackInstallResult>("/skills/packs/install", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

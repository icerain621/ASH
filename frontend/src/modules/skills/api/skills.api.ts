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

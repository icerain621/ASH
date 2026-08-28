import { api } from "@/services/http/client";

export type RepoProfile = {
  id: string;
  repoRoot: string;
  languages: string[];
  modules: string[];
  testCommands: string[];
  markers?: Record<string, boolean>;
  summary: string;
  contextRef: string;
};

export type WikiPage = {
  id: string;
  title: string;
  body: string;
  layer?: string;
  source: string;
  memoryId?: string;
  contextRef: string;
  tags?: string[];
};

export type WikiListResponse = {
  items: WikiPage[];
  query?: string;
};

export function getRepoProfile(repoRoot = ".") {
  const q = new URLSearchParams({ repoRoot });
  return api<RepoProfile>(`/repos/profile?${q}`);
}

export function listWikiPages(opts: { repoRoot?: string; q?: string; limit?: number } = {}) {
  const q = new URLSearchParams();
  if (opts.repoRoot) q.set("repoRoot", opts.repoRoot);
  if (opts.q) q.set("q", opts.q);
  if (opts.limit) q.set("limit", String(opts.limit));
  const qs = q.toString();
  return api<WikiListResponse>(`/wiki/pages${qs ? `?${qs}` : ""}`);
}

export function getWikiPage(pageId: string, repoRoot?: string) {
  const q = new URLSearchParams();
  if (repoRoot) q.set("repoRoot", repoRoot);
  const qs = q.toString();
  return api<WikiPage>(`/wiki/pages/${encodeURIComponent(pageId)}${qs ? `?${qs}` : ""}`);
}

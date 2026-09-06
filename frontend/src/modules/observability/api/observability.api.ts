import { api } from "@/services/http/client";

export type OtelStatus = {
  enabled: boolean;
  serviceName: string;
  endpoint?: string;
  insecure?: boolean;
};

export function getOtelStatus() {
  return api<OtelStatus>("/observability/otel/status");
}

export type RagProfile = {
  spaceId: string;
  ftsAvailable: boolean;
  ftsEngine?: string;
  defaultRetrievalMode: string;
  hybridAvailable?: boolean;
  vectorAvailable?: boolean;
  vectorPointCount?: number;
  embedderKind?: string;
  embedderDim?: number;
  lspAvailable?: boolean;
  databaseDialect?: string;
  documentCount: number;
  chunkCount: number;
  pathEntryCount?: number;
  symbolCount?: number;
  fallbackQueryCount: number;
};

export function getRagProfile() {
  return api<RagProfile>("/rag/profile");
}

export type RebuildSymbolsResponse = {
  paths: number;
  symbols: number;
  files: number;
};

export function rebuildRAGSymbols(body: { repoRoot: string; spaceId?: string }) {
  return api<RebuildSymbolsResponse>("/rag/symbols/rebuild", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export type RagLspPositionQuery = {
  repoRoot: string;
  path: string;
  line: number;
  character?: number;
  spaceId?: string;
  text?: string;
};

export type RagLspLocation = {
  path: string;
  uri?: string;
  line: number;
  character?: number;
};

export type RagLspHoverResponse = {
  contents: string;
  kind?: string;
  server?: string;
  path?: string;
};

export type RagLspDefinitionResponse = {
  locations: RagLspLocation[];
  server?: string;
  path?: string;
};

export type RagLspReferencesResponse = {
  locations: RagLspLocation[];
  server?: string;
  path?: string;
  source: string;
  truncated?: boolean;
};

export function postRagLspHover(body: RagLspPositionQuery) {
  return api<RagLspHoverResponse>("/rag/lsp/hover", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function postRagLspDefinition(body: RagLspPositionQuery) {
  return api<RagLspDefinitionResponse>("/rag/lsp/definition", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function postRagLspReferences(body: RagLspPositionQuery & { limit?: number }) {
  return api<RagLspReferencesResponse>("/rag/lsp/references", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

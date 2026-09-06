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
